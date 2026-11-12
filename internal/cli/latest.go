package cli

import (
	"github.com/spf13/cobra"
)

const latestLong = `Print the latest version in a repository's git tags.

svu latest reads the tags of the git repository that contains the current
directory, or --directory if given. Only tags on the checked-out commit's
history count, unless --global is given. It skips any tag that isn't a valid
version, ignores a leading "v", and prints the latest by svu's ordering.
svu modify reads tags the same way when given no version.

With --tag-prefix, only tags starting with the prefix count, and the prefix
is ignored. With --released, pre-releases are skipped too. With --initial,
that version is printed if no tag matches.`

const latestExample = `  svu latest                         # 1.3.0-rc-002
  svu latest --released              # 1.2.0
  svu latest --with-v-prefix         # v1.3.0-rc-002
  svu latest --global                # including tags on other branches
  svu latest --tag-prefix api/       # 0.4.0, from the tag api/v0.4.0
  svu l --directory path/to/repository`

func newLatestCommand(env environment) *cobra.Command {
	var (
		tags     tagFlags
		released bool
		prefix   vPrefix
	)

	cmd := &cobra.Command{
		Use:               "latest",
		Aliases:           []string{"l"},
		Short:             "Print the latest version in a repository's git tags",
		Long:              latestLong,
		Example:           latestExample,
		Args:              noArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, _ []string) error {
			releaseFlag := ""
			if released {
				releaseFlag = "--released"
			}

			q, err := tags.query(releaseFlag, false)
			if err != nil {
				return err
			}

			v, err := latestTag(cmd.Context(), env.readTags, q)
			if err != nil {
				return err
			}

			return writeResult(cmd.OutOrStdout(), prefix.format(v))
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	tags.add(cmd)

	flags.BoolVar(&released, "released", false, "Skip pre-releases, and print the latest release")
	setHelpGroup(flags, "released", groupFilter)

	prefix.addFlag(cmd)

	return cmd
}
