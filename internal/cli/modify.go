package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/tedslittlerobot/svu/internal/semver"
)

const modifyLong = `Modify a version and print the result.

svu modify reads one version:

  - from the version argument, or from standard input if the argument is "-";
  - with no argument, from the latest git tag that is a valid version, found
    the same way as svu latest finds it (see svu help latest).

It applies any modifiers, then prints the result on standard output. A
version modifier on its own removes any stage. Add a stage modifier, as in
--minor --alpha, to start that stage at 001. Stages never go backwards.

--hotfix patches the latest release, skipping pre-releases, to fix what's
already out while a pre-release is in progress.

--auto works out the bump from the Conventional Commits since the latest tag:
a breaking change bumps MAJOR (MINOR before 1.0.0), feat bumps MINOR, and fix
and perf bump PATCH. If the latest tag is a pre-release that already covers
the bump, it carries on from it. If no commit needs a new version, svu modify
fails with exit status 8. --explain says on stderr how the bump was chosen.

--apply also tags the checked-out commit with the new version, named like the
latest tag (with a "v" if it has one). --apply-annotate makes an annotated
tag, and opens your editor for its message. Neither pushes the tag. If the
version is already tagged, nothing is tagged or printed, and svu modify fails
with exit status 9.`

const modifyExample = `  svu modify 1.2.3 --minor            # 1.3.0
  svu modify 1.2.3 --minor --alpha    # 1.3.0-alpha-001
  svu modify 1.2.3-rc-001 --rc        # 1.2.3-rc-002
  svu modify 1.2.3-rc-001 --release   # 1.2.3
  echo v1.2.3 | svu modify - --patch  # 1.2.4
  svu m --patch --with-v-prefix       # the latest git tag, bumped, as in v1.2.4
  svu m --hotfix                      # 1.2.4, while 1.3.0-rc-002 is in progress
  svu m --initial 0.0.0 --minor       # 0.1.0 in a repository with no tags yet
  svu m --auto --explain              # the bump the commits since the latest tag need
  svu m --auto --rc                   # the same, as a release candidate
  svu m --auto --apply                # tag the checked-out commit with it`

func newModifyCommand(env environment) *cobra.Command {
	var (
		tags    tagFlags
		prefix  vPrefix
		explain bool
		apply   applyFlags
		mods    *modifiers
	)

	cmd := &cobra.Command{
		Use:               "modify [version]",
		Aliases:           []string{"m"},
		Short:             "Modify a version, such as bumping it or moving it to a new stage",
		Long:              modifyLong,
		Example:           modifyExample,
		Args:              atMostOneVersion,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check the flags before reading any input, which may run git.
			if err := mods.validate(); err != nil {
				return err
			}

			if flag := tags.changed(cmd); flag != "" && len(args) > 0 {
				return gitFlagWithVersionError(flag)
			}

			releaseFlag := ""
			if mods.hotfix() {
				releaseFlag = hotfixReleaseFlag
			}

			q, err := tags.query(releaseFlag, true)
			if err != nil {
				return err
			}

			if explain && !mods.auto() {
				return &userError{
					code:     ExitUsage,
					headline: "--explain only works with --auto.",
					detail:   "--explain says how --auto chose the bump from the commits.",
					hint:     "Add `--auto`, or remove `--explain`.",
				}
			}

			var v semver.Version

			if mods.auto() {
				v, err = runAuto(cmd, args, env, q, mods, explain)
			} else {
				v, err = readVersion(cmd.Context(), args, cmd.InOrStdin(), q, env.readTags)
				if err == nil {
					v, err = mods.apply(v)
				}
			}

			if err != nil {
				return err
			}

			// Tag before printing: on failure, stdout must stay empty.
			if err := apply.tag(cmd, env, q, v, prefix); err != nil {
				return err
			}

			return writeResult(cmd.OutOrStdout(), prefix.format(v))
		},
	}

	// Flags appear in help in the order they are defined here, by group.
	flags := cmd.Flags()
	flags.SortFlags = false

	tags.add(cmd)

	mods = newModifiers(flags)

	prefix.addFlag(cmd)

	flags.BoolVar(&explain, "explain", false, "With --auto, say on stderr how the bump was chosen")
	setHelpGroup(flags, "explain", groupOutput)

	apply.add(cmd)

	return cmd
}

// runAuto works out the version for svu modify --auto: the bump comes from
// the Conventional Commits since the latest tag.
func runAuto(
	cmd *cobra.Command, args []string, env environment, q tagQuery, mods *modifiers, explain bool,
) (semver.Version, error) {
	if len(args) > 0 {
		return semver.Version{}, &userError{
			code:     ExitUsage,
			headline: "--auto can't be used when a version is given.",
			detail:   "--auto works out the bump from the commits since the latest tag, so it reads the tags itself.",
			hint:     "Remove the version, or choose the bump with `--major`, `--minor` or `--patch`.",
		}
	}

	var out io.Writer
	if explain {
		out = cmd.ErrOrStderr()
	}

	v, err := autoVersion(cmd.Context(), env, q, mods.stage(), out)
	if err != nil {
		var ue *userError
		if errors.As(err, &ue) {
			return semver.Version{}, ue
		}

		return semver.Version{}, mods.modifyError(err)
	}

	return v, nil
}

// atMostOneVersion checks that svu modify was given no more than one version.
func atMostOneVersion(_ *cobra.Command, args []string) error {
	if len(args) <= 1 {
		return nil
	}

	return &userError{
		code:     ExitUsage,
		headline: fmt.Sprintf("svu modify was given %d versions, but it takes at most one.", len(args)),
		hint:     "Give a single version, as in `svu modify 1.2.3`.",
	}
}

// writeResult writes a command's main output to stdout, followed by a
// newline. conventions/output.md says it is the only thing on stdout.
func writeResult(w io.Writer, output string) error {
	if _, err := fmt.Fprintln(w, output); err != nil {
		return &userError{
			code:     ExitInternal,
			headline: "svu could not write its output.",
			detail:   "The output stream was closed before svu finished writing.",
			hint:     reportHint,
			err:      err,
		}
	}

	return nil
}

// vPrefix is the --with-v-prefix flag, shared by every command that prints a
// version.
type vPrefix bool

// addFlag adds --with-v-prefix, and its alias --vp, to cmd.
func (p *vPrefix) addFlag(cmd *cobra.Command) {
	cmd.Flags().BoolVar((*bool)(p), "with-v-prefix", false, `Print the version with a leading "v" (alias --vp)`)
	setHelpGroup(cmd.Flags(), "with-v-prefix", groupOutput)
}

// format writes v as a command prints it: without a "v" unless asked for.
func (p vPrefix) format(v semver.Version) string {
	if p {
		return "v" + v.String()
	}

	return v.String()
}
