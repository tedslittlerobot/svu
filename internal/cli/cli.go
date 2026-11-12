// Package cli is svu's command-line interface: it parses arguments, dispatches
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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/tedslittlerobot/svm/internal/buildinfo"
	"github.com/tedslittlerobot/svm/internal/colour"
	"github.com/tedslittlerobot/svm/internal/git"
	"github.com/tedslittlerobot/svm/internal/semver"
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

// environment is what svu needs from the world outside its streams. Tests
// replace it to make runs deterministic.
type environment struct {
	info     buildinfo.Info
	readTags tagReader
	// getenv returns an environment variable, or "" if it is unset.
	getenv func(string) string
}

// Execute runs svu with the given arguments (excluding the program name) and
// returns the process exit code.
func Execute(ctx context.Context, args []string, streams Streams) int {
	return execute(ctx, args, streams, environment{
		info:     buildinfo.Get(),
		readTags: git.Tags,
		getenv:   os.Getenv,
	})
}

func execute(ctx context.Context, args []string, streams Streams, env environment) int {
	var colourMode colourFlag

	root := newRootCommand(streams, env, &colourMode)
	root.SetArgs(args)

	if err := root.ExecuteContext(ctx); err != nil {
		// If --colour itself failed to parse, the mode is still auto.
		on := colour.Enabled(colourMode.mode, env.getenv, streams.Err)
		render(streams.Err, err, colour.New(on))

		return ExitFailure
	}

	return ExitOK
}

const rootLong = `svu (Semantic Version Utilities) works with semantic versions.

svu uses a strict subset of SemVer 2.0.0: MAJOR.MINOR.PATCH, optionally
followed by a stage (alpha, beta or rc) and a three-digit index, as in
1.2.3-rc-001.`

const rootExample = `  svu modify 1.2.3 --minor     # 1.3.0
  svu m --patch --rc           # the latest git tag's next patch, as an rc
  svu latest --release         # the latest released version in git tags
  svu help modify              # how to modify a version`

// newRootCommand builds svu's command tree. colourMode receives the value of
// --colour once flags are parsed.
func newRootCommand(streams Streams, env environment, colourMode *colourFlag) *cobra.Command {
	root := &cobra.Command{
		Use:     "svu",
		Short:   "svu (Semantic Version Utilities) works with semantic versions",
		Long:    rootLong,
		Example: rootExample,
		Version: env.info.Version,
		// svu runs a command. On its own it shows help; given anything else,
		// it says that isn't a command. This replaces cobra's own message, so
		// that it follows conventions/output.md.
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}

			return unknownCommandError(cmd, args[0])
		},

		// Errors are reported once, by execute, rather than by cobra.
		SilenceErrors: true,
		SilenceUsage:  true,

		// cobra only sets its default when it reports unknown commands
		// itself, which svu doesn't let it do.
		SuggestionsMinimumDistance: 2,
	}

	root.PersistentFlags().Var(colourMode, "colour",
		"Use colour on stderr: auto (the default), always or never (alias --color)") //nolint:misspell // --color is the alias output.md requires.

	// Defined here, rather than by cobra, to control their wording.
	flags := root.Flags()
	flags.SortFlags = false
	flags.BoolP("help", "h", false, "Show this help")
	flags.BoolP("version", "v", false, "Print svu's version")

	root.SetGlobalNormalizationFunc(flagAliases)

	root.SetIn(streams.In)
	root.SetOut(streams.Out)
	root.SetErr(streams.Err)
	// The version is --version's main output, so it goes to stdout bare.
	root.SetVersionTemplate("{{.Version}}\n")
	root.SetFlagErrorFunc(flagError)

	// Help goes to stderr, coloured as conventions/output.md describes, for
	// --help, -h and svu help alike.
	help := func(cmd *cobra.Command) {
		on := colour.Enabled(colourMode.mode, env.getenv, cmd.ErrOrStderr())
		renderHelp(cmd.ErrOrStderr(), cmd, colour.New(on))
	}
	root.SetHelpFunc(func(cmd *cobra.Command, _ []string) { help(cmd) })
	root.SetUsageFunc(func(cmd *cobra.Command) error {
		help(cmd)

		return nil
	})
	root.SetHelpCommand(newHelpCommand())

	root.AddCommand(
		newLatestCommand(env),
		newModifyCommand(env),
		newVersionCommand(env.info),
	)

	return root
}

// unknownCommandError reports that name isn't one of svu's commands, and
// suggests what was probably meant.
func unknownCommandError(root *cobra.Command, name string) *userError {
	ue := &userError{headline: fmt.Sprintf("%q is not an svu command.", name)}

	suggestions := root.SuggestionsFor(name)

	switch {
	case isVersion(name):
		ue.hint = fmt.Sprintf("To modify a version, run `svu modify %s`.", name)
	case len(suggestions) > 0:
		ue.hint = fmt.Sprintf("Did you mean `svu %s`?", suggestions[0])
	default:
		ue.hint = "To list svu's commands, run `svu --help`."
	}

	return ue
}

func isVersion(s string) bool {
	_, err := semver.Parse(s)

	return err == nil
}

// noArgs checks that a command was given no arguments.
func noArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}

	return &userError{
		headline: fmt.Sprintf("%s takes no arguments, but was given %q.", cmd.CommandPath(), args[0]),
		hint:     fmt.Sprintf("Run `%s` on its own.", cmd.CommandPath()),
	}
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
		return valueRuleError("must be auto, always or never")
	}

	f.mode = mode

	return nil
}
