package cli

import (
	"github.com/spf13/cobra"

	"github.com/tedslittlerobot/svu/internal/buildinfo"
)

func newVersionCommand(info buildinfo.Info) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print detailed build information for svu",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// The build information is this command's main output.
			return writeResult(cmd.OutOrStdout(), info.String())
		},
	}
}
