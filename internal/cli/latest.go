package cli

import (
	"github.com/spf13/cobra"
)

const latestLong = `Print the latest version in a repository's git tags.

svu latest reads the tags of the git repository that contains the current
directory, or --directory if given. It skips any tag that isn't a valid
version, ignores a leading "v", and prints the latest by svu's ordering.
It reads exactly the tags that svu modify reads when given no version.

With --release, pre-releases are skipped too.`

const latestExample = `  svu latest                         # 1.3.0-rc-002
  svu latest --release               # 1.2.0
  svu latest --with-v-prefix         # v1.3.0-rc-002
  svu l --directory path/to/repository`

func newLatestCommand(env environment) *cobra.Command {
	var (
		q      tagQuery
		prefix vPrefix
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
			v, err := latestTag(cmd.Context(), env.readTags, q)
			if err != nil {
				return err
			}

			return writeResult(cmd.OutOrStdout(), prefix.format(v))
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	addDirectoryFlag(cmd, &q.dir)

	flags.BoolVar(&q.releasesOnly, "release", false, "Skip pre-releases, and print the latest release")
	setHelpGroup(flags, "release", groupFilter)

	prefix.addFlag(cmd)

	return cmd
}
