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
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/tedslittlerobot/svm/internal/buildinfo"
	"github.com/tedslittlerobot/svm/internal/colour"
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

// environment is what svm needs from the world outside its streams. Tests
// replace it to make runs deterministic.
type environment struct {
	info     buildinfo.Info
	readTags tagReader
	// getenv returns an environment variable, or "" if it is unset.
	getenv func(string) string
}

// Execute runs svm with the given arguments (excluding the program name) and
// returns the process exit code.
func Execute(ctx context.Context, args []string, streams Streams) int {
	return execute(ctx, args, streams, environment{
		info:     buildinfo.Get(),
		readTags: git.Tags,
		getenv:   os.Getenv,
	})
}

func execute(ctx context.Context, args []string, streams Streams, env environment) int {
	root := newRootCommand(streams, env)
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

It applies any modifiers, then prints the result on standard output. A
version modifier on its own removes any stage. Add a stage modifier, as in
--minor --alpha, to start that stage at 001. Stages never go backwards.`

const rootExample = `  svm 1.2.3 --minor            # 1.3.0
  svm 1.2.3 --minor --alpha    # 1.3.0-alpha-001
  svm 1.2.3-rc-001 --rc        # 1.2.3-rc-002
  svm 1.2.3-rc-001 --release   # 1.2.3
  echo v1.2.3 | svm - --patch  # 1.2.4
  svm --patch --with-v-prefix  # the latest git tag, bumped, as in v1.2.4
  svm --directory path/to/repository`

func newRootCommand(streams Streams, env environment) *cobra.Command {
	var (
		opts       inputOptions
		vPrefix    bool
		mods       *modifiers
		colourMode colourFlag
	)

	root := &cobra.Command{
		Use:               "svm [version]",
		Short:             "svm (SemVer Modifier) reads and modifies semantic versions",
		Long:              rootLong,
		Example:           rootExample,
		Version:           env.info.Version,
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

		// Errors are reported once, by execute, rather than by cobra.
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	// Flags appear in help in the order they are defined here, by group.
	flags := root.Flags()
	flags.SortFlags = false

	flags.StringVar(&opts.directory, "directory", "", "Read tags from the git repository containing `path`")
	setHelpGroup(flags, "directory", groupInput)
	_ = root.MarkFlagDirname("directory") // Only fails if the flag doesn't exist.

	mods = newModifiers(flags)

	flags.BoolVar(&vPrefix, "with-v-prefix", false, `Print the version with a leading "v" (alias --vp)`)
	setHelpGroup(flags, "with-v-prefix", groupOutput)

	root.PersistentFlags().Var(&colourMode, "colour",
		"Use colour on stderr: auto (the default), always or never (alias --color)") //nolint:misspell // --color is the alias output.md requires.
	setHelpGroup(root.PersistentFlags(), "colour", groupOutput)

	// Defined here, rather than by cobra, to control their wording.
	flags.BoolP("help", "h", false, "Show this help")
	flags.BoolP("version", "v", false, "Print svm's version")

	root.SetGlobalNormalizationFunc(flagAliases)

	root.SetIn(streams.In)
	root.SetOut(streams.Out)
	root.SetErr(streams.Err)
	root.SetVersionTemplate("{{.Name}} {{.Version}}\n")

	// Help goes to stderr, coloured as conventions/output.md describes. Only
	// --help shows it: the hidden placeholder replaces cobra's "help"
	// command, so "svm help" is read as a version like any other word.
	help := func(cmd *cobra.Command) {
		on := colour.Enabled(colourMode.mode, env.getenv, cmd.ErrOrStderr())
		renderHelp(cmd.ErrOrStderr(), cmd, colour.New(on))
	}
	root.SetHelpFunc(func(cmd *cobra.Command, _ []string) { help(cmd) })
	root.SetUsageFunc(func(cmd *cobra.Command) error {
		help(cmd)

		return nil
	})
	root.SetHelpCommand(&cobra.Command{Use: "__help", Hidden: true})

	root.AddCommand(
		newVersionCommand(env.info),
	)

	return root
}

// flagAliases maps alternative flag names to their canonical names.
func flagAliases(_ *pflag.FlagSet, name string) pflag.NormalizedName {
	switch name {
	case "vp":
		name = "with-v-prefix"
	case "color": //nolint:misspell // --color is the conventional alias that output.md requires.
		name = "colour"
	}

	return pflag.NormalizedName(name)
}

// colourFlag is the value of --colour.
type colourFlag struct {
	mode colour.Mode
}

// String implements pflag.Value.
func (f *colourFlag) String() string { return f.mode.String() }

// Type implements pflag.Value.
func (f *colourFlag) Type() string { return "when" }

// Set implements pflag.Value.
func (f *colourFlag) Set(s string) error {
	mode, err := colour.ParseMode(s)
	if err != nil {
		return errors.New("must be auto, always or never")
	}

	f.mode = mode

	return nil
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
