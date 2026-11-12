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

	"github.com/tedslittlerobot/svm/internal/buildinfo"
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
	return execute(ctx, args, streams, buildinfo.Get())
}

func execute(ctx context.Context, args []string, streams Streams, info buildinfo.Info) int {
	root := newRootCommand(streams, info)
	root.SetArgs(args)

	if err := root.ExecuteContext(ctx); err != nil {
		_, _ = fmt.Fprintf(streams.Err, "svm: %v\n", err)

		return ExitFailure
	}

	return ExitOK
}

func newRootCommand(streams Streams, info buildinfo.Info) *cobra.Command {
	root := &cobra.Command{
		Use:     "svm",
		Short:   "svm (SemVer Modifier) reads and modifies semantic versions",
		Long:    "svm (SemVer Modifier) reads and modifies semantic version strings.",
		Version: info.Version,

		// Errors are reported once, by execute, rather than by cobra.
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.SetIn(streams.In)
	root.SetOut(streams.Out)
	root.SetErr(streams.Err)
	root.SetVersionTemplate("{{.Name}} {{.Version}}\n")

	root.AddCommand(
		newVersionCommand(info),
	)

	return root
}
