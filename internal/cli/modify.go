package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tedslittlerobot/semvu/internal/git"
	"github.com/tedslittlerobot/semvu/internal/semver"
)

const modifyLong = `Modify a version and print the result.

semvu modify reads one version:

  - from the version argument, or from standard input if the argument is "-";
  - with no argument, from the latest git tag that is a valid version, found
    the same way as semvu latest finds it (see semvu help latest).

It applies any modifiers, then prints the result on standard output. A
version modifier on its own removes any stage. Add a stage modifier, as in
--minor --alpha, to start that stage at 001. Stages never go backwards,
unless --allow-stage-regression is given: then --alpha, --beta or --rc may
move a pre-release back to an earlier stage. Read from git tags, it carries
on after the highest index already tagged at that stage, so 1.2.3-rc-001
goes back to 1.2.3-alpha-003 if 1.2.3-alpha-002 is tagged. A release still
can't go back into a stage.

The result keeps the "v" of the version it read, from the argument, standard
input, the tag or --initial: v1.2.3 gives v1.2.4, and 1.2.3 gives 1.2.4. Use
--with-v or --without-v to choose instead.

--hotfix patches the latest release, skipping pre-releases, to fix what's
already out while a pre-release is in progress.

Read from git tags, a version can't move on from a pre-release that hasn't
been released: with 1.2.3-rc-001 the latest tag, --minor fails until 1.2.3 is
tagged on some branch. Release it with --release, or add --skip-release to
move on anyway. This applies to --auto too.

--for-major reads only the tags with that MAJOR version, to work on an older
major version: with v1.4.2 and v2.0.0 tagged, --for-major 1 --patch gives
v1.4.3. The result must stay in that major version, so --major can't be used
with it.

--auto works out the bump from the Conventional Commits since the latest tag:
a breaking change bumps MAJOR (MINOR before 1.0.0), feat bumps MINOR, and fix
and perf bump PATCH. If the latest tag is a pre-release that already covers
the bump, it carries on from it. If no commit needs a new version, semvu modify
fails with exit status 8. --explain says on stderr how the bump was chosen.

--apply also tags the checked-out commit with the new version, exactly as
it's printed (after any --tag-prefix). --apply-annotate makes an annotated
tag, and opens your editor for its message. Neither pushes the tag. If the
version is already tagged, nothing is tagged or printed, and semvu modify fails
with exit status 9.`

const modifyExample = `  semvu modify 1.2.3 --minor            # 1.3.0
  semvu modify 1.2.3 --minor --alpha    # 1.3.0-alpha-001
  semvu modify 1.2.3-rc-001 --rc        # 1.2.3-rc-002
  semvu modify 1.2.3-rc-001 --release   # 1.2.3
  echo v1.2.3 | semvu modify - --patch  # v1.2.4
  semvu modify v1.2.3 --without-v       # 1.2.3
  semvu m --patch                       # the latest git tag, bumped, as in v1.2.4
  semvu m --hotfix                      # 1.2.4, while 1.3.0-rc-002 is in progress
  semvu m --for-major 1 --patch         # v1.4.3, while v2.0.0 is out
  semvu m --initial 0.0.0 --minor       # 0.1.0 in a repository with no tags yet
  semvu m --auto --explain              # the bump the commits since the latest tag need
  semvu m --auto --rc                   # the same, as a release candidate
  semvu m --auto --apply                # tag the checked-out commit with it
  semvu modify 1.2.3-rc-001 --alpha --allow-stage-regression   # 1.2.3-alpha-001`

func newModifyCommand(env environment) *cobra.Command {
	var (
		tags    tagFlags
		vstyle  vFlags
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

			if err := vstyle.validate(); err != nil {
				return err
			}

			if flag := tags.changed(cmd); flag != "" && len(args) > 0 {
				return gitFlagWithVersionError(flag)
			}

			if mods.skipsRelease() && len(args) > 0 {
				return gitFlagWithVersionError("--" + skipReleaseFlag)
			}

			if tags.forMajor.set && mods.bumpsMajor() {
				return &userError{
					code:     ExitUsage,
					headline: "--major can't be used with --for-major.",
					detail: fmt.Sprintf("--for-major %d works within major version %d, and --major would leave it.",
						tags.forMajor.major, tags.forMajor.major),
					hint: "To start a new major version, remove `--for-major`.",
				}
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

			var (
				v      semver.Version
				source string // The text v came from, for its "v".
			)

			if mods.auto() {
				v, source, err = runAuto(cmd, args, env, q, mods, explain)
			} else {
				v, source, err = modifyVersion(cmd, args, env, q, mods)
			}

			if err != nil {
				return err
			}

			out := vstyle.format(v, source)

			// Tag before printing: on failure, stdout must stay empty.
			if err := apply.tag(cmd, env, q, v, out); err != nil {
				return err
			}

			return writeResult(cmd.OutOrStdout(), out)
		},
	}

	// Flags appear in help in the order they are defined here, by group.
	flags := cmd.Flags()
	flags.SortFlags = false

	tags.add(cmd)

	mods = newModifiers(flags)

	vstyle.add(cmd)

	flags.BoolVar(&explain, "explain", false, "With --auto, say on stderr how the bump was chosen")
	setHelpGroup(flags, "explain", groupOutput)

	apply.add(cmd)

	return cmd
}

// modifyVersion reads the version semvu modify starts from, and applies the
// modifiers to it. It also returns the text the version came from, as
// readVersion does.
func modifyVersion(
	cmd *cobra.Command, args []string, env environment, q tagQuery, mods *modifiers,
) (semver.Version, string, error) {
	v, source, err := readVersion(cmd.Context(), args, cmd.InOrStdin(), q, env.readTags)
	if err != nil {
		return semver.Version{}, "", err
	}

	// Going back a stage carries on from the versions already tagged, on any
	// branch, so as not to repeat one. A version given directly has no tags.
	var taken []string

	if mods.regresses() && len(args) == 0 {
		taken, err = env.readTags(cmd.Context(), q.dir, git.TagOptions{All: true, Prefix: q.prefix})
		if err != nil {
			return semver.Version{}, "", tagsError(err, q)
		}
	}

	next, err := mods.apply(v, taken)
	if err != nil {
		return semver.Version{}, "", err
	}

	if len(args) == 0 {
		if err := checkSkippedRelease(cmd.Context(), env.readTags, q, mods, v, next); err != nil {
			return semver.Version{}, "", err
		}
	}

	return next, source, nil
}

// checkSkippedRelease checks that going from the version read from git tags
// to next doesn't skip from's release (§Skipping a release), unless
// --skip-release was given. A release tagged on any branch counts, so the
// check reads every tag.
func checkSkippedRelease(
	ctx context.Context, readTags tagReader, q tagQuery, mods *modifiers, from, next semver.Version,
) error {
	if mods.skipsRelease() || !from.SkipsRelease(next) {
		return nil
	}

	release := from.Released()

	tags, err := readTags(ctx, q.dir, git.TagOptions{All: true, Prefix: q.prefix})
	if err != nil {
		return tagsError(err, q)
	}

	if tagName(tags, release) != "" {
		return nil
	}

	return &userError{
		code:     ExitModification,
		headline: fmt.Sprintf("%q has not been released yet.", release.String()),
		detail: fmt.Sprintf("The latest version is the pre-release %q, and %q would move on without releasing it.",
			from.String(), next.String()),
		hint: fmt.Sprintf("Release it first with `--release`, or add `--%s` to move on anyway.", skipReleaseFlag),
	}
}

// runAuto works out the version for semvu modify --auto: the bump comes from
// the Conventional Commits since the latest tag. It also returns the text the
// version came from, as readVersion does.
func runAuto(
	cmd *cobra.Command, args []string, env environment, q tagQuery, mods *modifiers, explain bool,
) (semver.Version, string, error) {
	if len(args) > 0 {
		return semver.Version{}, "", &userError{
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

	v, source, err := autoVersion(cmd.Context(), env, q, mods, out)
	if err != nil {
		var ue *userError
		if errors.As(err, &ue) {
			return semver.Version{}, "", ue
		}

		return semver.Version{}, "", mods.modifyError(err)
	}

	return v, source, nil
}

// atMostOneVersion checks that semvu modify was given no more than one version.
func atMostOneVersion(_ *cobra.Command, args []string) error {
	if len(args) <= 1 {
		return nil
	}

	return &userError{
		code:     ExitUsage,
		headline: fmt.Sprintf("semvu modify was given %d versions, but it takes at most one.", len(args)),
		hint:     "Give a single version, as in `semvu modify 1.2.3`.",
	}
}

// writeResult writes a command's main output to stdout, followed by a
// newline. conventions/output.md says it is the only thing on stdout.
func writeResult(w io.Writer, output string) error {
	if _, err := fmt.Fprintln(w, output); err != nil {
		return &userError{
			code:     ExitInternal,
			headline: "semvu could not write its output.",
			detail:   "The output stream was closed before semvu finished writing.",
			hint:     reportHint,
			err:      err,
		}
	}

	return nil
}

// vFlags are --with-v and --without-v, shared by every command that prints a
// version.
type vFlags struct {
	with    bool
	without bool
}

// add adds --with-v and --without-v to cmd.
func (f *vFlags) add(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.BoolVar(&f.with, "with-v", false, `Always print the version with a leading "v"`)
	flags.BoolVar(&f.without, "without-v", false, `Never print a leading "v"`)

	setHelpGroup(flags, "with-v", groupOutput)
	setHelpGroup(flags, "without-v", groupOutput)
}

// validate checks that --with-v and --without-v weren't both given.
func (f vFlags) validate() error {
	if !f.with || !f.without {
		return nil
	}

	return &userError{
		code:     ExitUsage,
		headline: "--with-v and --without-v can't be used together.",
		detail: `Without either, semvu keeps the "v" of the version it read: from the argument,` +
			"\nstandard input or the tag.",
		hint: "Remove one of them.",
	}
}

// format writes v as a command prints it. source is the text v came from,
// such as the argument or the tag's name without --tag-prefix. Without
// --with-v or --without-v, the result has a "v" if source has one.
func (f vFlags) format(v semver.Version, source string) string {
	if f.with || (!f.without && strings.HasPrefix(source, "v")) {
		return "v" + v.String()
	}

	return v.String()
}
