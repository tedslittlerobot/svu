package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tedslittlerobot/svm/internal/buildinfo"
)

func newVersionCommand(info buildinfo.Info) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print detailed build information for svu",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := fmt.Fprintln(cmd.OutOrStdout(), info); err != nil {
				return fmt.Errorf("writing version: %w", err)
			}

			return nil
		},
	}
}
