package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

const modifyLong = `Modify a version and print the result.

svu modify reads one version:

  - from the version argument, or from standard input if the argument is "-";
  - with no argument, from the latest git tag that is a valid version, in the
    repository that contains the current directory, or --directory if given.

It applies any modifiers, then prints the result on standard output. A
version modifier on its own removes any stage. Add a stage modifier, as in
--minor --alpha, to start that stage at 001. Stages never go backwards.`

const modifyExample = `  svu modify 1.2.3 --minor            # 1.3.0
  svu modify 1.2.3 --minor --alpha    # 1.3.0-alpha-001
  svu modify 1.2.3-rc-001 --rc        # 1.2.3-rc-002
  svu modify 1.2.3-rc-001 --release   # 1.2.3
  echo v1.2.3 | svu modify - --patch  # 1.2.4
  svu m --patch --with-v-prefix       # the latest git tag, bumped, as in v1.2.4`

func newModifyCommand(env environment) *cobra.Command {
	var (
		opts    inputOptions
		vPrefix bool
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
			opts.directorySet = cmd.Flags().Changed("directory")

			// Check the flags before reading any input, which may run git.
			if err := mods.validate(); err != nil {
				return err
			}

			v, err := readVersion(cmd.Context(), args, cmd.InOrStdin(), opts, env.readTags)
			if err != nil {
				return err
			}

			v, err = mods.apply(v)
			if err != nil {
				return err
			}

			out := v.String()
			if vPrefix {
				out = "v" + out
			}

			return writeResult(cmd.OutOrStdout(), out)
		},
	}

	// Flags appear in help in the order they are defined here, by group.
	flags := cmd.Flags()
	flags.SortFlags = false

	flags.StringVar(&opts.directory, "directory", "", "Read tags from the git repository containing `path`")
	setHelpGroup(flags, "directory", groupInput)
	_ = cmd.MarkFlagDirname("directory") // Only fails if the flag doesn't exist.

	mods = newModifiers(flags)

	flags.BoolVar(&vPrefix, "with-v-prefix", false, `Print the version with a leading "v" (alias --vp)`)
	setHelpGroup(flags, "with-v-prefix", groupOutput)

	return cmd
}

// atMostOneVersion checks that svu modify was given no more than one version.
func atMostOneVersion(_ *cobra.Command, args []string) error {
	if len(args) <= 1 {
		return nil
	}

	return &userError{
		headline: fmt.Sprintf("svu modify was given %d versions, but it takes at most one.", len(args)),
		hint:     "Give a single version, as in `svu modify 1.2.3`.",
	}
}

// writeResult writes a command's main output to stdout, followed by a
// newline. conventions/output.md says it is the only thing on stdout.
func writeResult(w io.Writer, output string) error {
	if _, err := fmt.Fprintln(w, output); err != nil {
		return &userError{
			headline: "svu could not write its output.",
			detail:   "The output stream was closed before svu finished writing.",
			hint:     reportHint,
			err:      err,
		}
	}

	return nil
}
