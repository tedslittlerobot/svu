// Package cli is svm's command-line interface: it parses arguments, dispatches
// to subcommands, and turns their results into output and exit codes.
//
// Commands here should stay thin. They translate between the terminal and the
// domain packages, and contain no version-manipulation logic of their own.
// Every command writes through the streams it is given, never through os.Stdout
// or os.Stderr directly, so that it can be tested with in-memory buffers.
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/tedslittlerobot/svm/internal/buildinfo"
	"github.com/tedslittlerobot/svm/internal/git"
)

// Process exit codes.
const (
	ExitOK      = 0
	ExitFailure = 1
)

// Streams holds the I/O handles available to commands.
type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// Execute runs svm with the given arguments (excluding the program name) and
// returns the process exit code.
func Execute(ctx context.Context, args []string, streams Streams) int {
	return execute(ctx, args, streams, buildinfo.Get(), git.Tags)
}

func execute(ctx context.Context, args []string, streams Streams, info buildinfo.Info, readTags tagReader) int {
	root := newRootCommand(streams, info, readTags)
	root.SetArgs(args)

	if err := root.ExecuteContext(ctx); err != nil {
		render(streams.Err, err)

		return ExitFailure
	}

	return ExitOK
}

const rootLong = `svm (SemVer Modifier) reads and modifies semantic version strings.

svm reads one version:

  - from the version argument, or from standard input if the argument is "-";
  - with no argument, from the latest git tag that is a valid version, in the
    repository that contains the current directory, or --directory if given.

Modifiers change the version before it is printed. Give at most one of each
kind:

  --major, --minor, --patch     bump that number, reset the ones after it to 0
                                and remove any stage
  --alpha, --beta, --rc         move to that stage at 001, or to the next index
                                if already there; stages never go backwards
  --release                     remove the stage from a pre-release

Together, as in --minor --alpha, they bump the version and start the stage
at 001.

svm prints the version on standard output, without a "v" prefix unless
--with-v-prefix is given.`

const rootExample = `  svm 1.2.3 --minor            # 1.3.0
  svm 1.2.3 --minor --alpha    # 1.3.0-alpha-001
  svm 1.2.3-rc-001 --rc        # 1.2.3-rc-002
  svm 1.2.3-rc-001 --release   # 1.2.3
  echo v1.2.3 | svm - --patch  # 1.2.4
  svm --patch --with-v-prefix  # the latest git tag, bumped, as in v1.2.4
  svm --directory path/to/repository`

func newRootCommand(streams Streams, info buildinfo.Info, readTags tagReader) *cobra.Command {
	var (
		opts    inputOptions
		vPrefix bool
		mods    *modifiers
	)

	root := &cobra.Command{
		Use:               "svm [version]",
		Short:             "svm (SemVer Modifier) reads and modifies semantic versions",
		Long:              rootLong,
		Example:           rootExample,
		Version:           info.Version,
		Args:              atMostOneVersion,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.directorySet = cmd.Flags().Changed("directory")

			// Check the flags before reading any input, which may run git.
			if err := mods.validate(); err != nil {
				return err
			}

			v, err := readVersion(cmd.Context(), args, cmd.InOrStdin(), opts, readTags)
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

		// Errors are reported once, by execute, rather than by cobra.
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	flags := root.Flags()
	mods = newModifiers(flags)
	flags.StringVar(&opts.directory, "directory", "",
		"read the latest version from the git tags of the repository containing this directory")
	flags.BoolVar(&vPrefix, "with-v-prefix", false, `print the version with a leading "v" (alias --vp)`)
	flags.SetNormalizeFunc(flagAliases)
	_ = root.MarkFlagDirname("directory") // Only fails if the flag doesn't exist.

	root.SetIn(streams.In)
	root.SetOut(streams.Out)
	root.SetErr(streams.Err)
	root.SetVersionTemplate("{{.Name}} {{.Version}}\n")

	root.AddCommand(
		newVersionCommand(info),
	)

	return root
}

// flagAliases maps alternative flag names to their canonical names.
func flagAliases(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	if name == "vp" {
		name = "with-v-prefix"
	}

	return pflag.NormalizedName(name)
}

// atMostOneVersion checks that svm was given no more than one version.
func atMostOneVersion(_ *cobra.Command, args []string) error {
	if len(args) <= 1 {
		return nil
	}

	return &userError{
		headline: fmt.Sprintf("svm was given %d versions, but it takes at most one.", len(args)),
		hint:     "Give a single version, as in svm 1.2.3.",
	}
}

// writeResult writes the result to stdout, as conventions/output.md requires:
// one version and a newline.
func writeResult(w io.Writer, version string) error {
	if _, err := fmt.Fprintln(w, version); err != nil {
		return &userError{
			headline: "svm could not write its output.",
			detail:   "The output stream was closed before svm finished writing.",
			hint:     reportHint,
			err:      err,
		}
	}

	return nil
}
