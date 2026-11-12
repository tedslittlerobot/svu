package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const validateLong = `Check that a version is valid.

svu validate prints nothing. It exits with status 0 if the version is valid,
or 3 if it isn't, and then explains why on standard error. Give the version
as the argument, or "-" to read it from standard input.

A leading "v" is accepted, as everywhere in svu.`

const validateExample = `  svu validate 1.2.3-rc-001
  svu validate 1.2.3-alpha-1                 # exit status 3, and the reason
  echo "$TAG" | svu validate -
  svu validate "$TAG" 2>/dev/null && echo valid`

func newValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:               "validate <version>",
		Short:             "Check that a version is valid, by exit status",
		Long:              validateLong,
		Example:           validateExample,
		Args:              exactlyOneVersion,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			input, where := args[0], ""

			if input == stdinArgument {
				var err error

				input, err = readStdin(cmd.InOrStdin())
				if err != nil {
					return err
				}

				where = " on standard input"
			}

			// The exit status is the answer, so there is no main output.
			_, err := parseVersion(input, where, false)

			return err
		},
	}
}

// exactlyOneVersion checks that svu validate was given a version.
func exactlyOneVersion(_ *cobra.Command, args []string) error {
	switch len(args) {
	case 1:
		return nil
	case 0:
		return &userError{
			code:     ExitUsage,
			headline: "svu validate needs a version.",
			hint:     "Give one, as in `svu validate 1.2.3`, or `-` to read it from standard input.",
		}
	default:
		return &userError{
			code:     ExitUsage,
			headline: fmt.Sprintf("svu validate was given %d versions, but it takes one.", len(args)),
			hint:     "Check one version at a time, as in `svu validate 1.2.3`.",
		}
	}
}
