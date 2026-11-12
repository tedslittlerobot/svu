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
that version is printed if no tag matches.

It prints the version as the tag has it, with or without a "v", unless
--with-v or --without-v says otherwise.

With --at-head, only tags on the checked-out commit itself count. It fails
(exit status 6) if that commit has no version tag, so a script can check
whether it has already tagged a commit.`

const latestExample = `  svu latest                         # v1.3.0-rc-002, from the tag v1.3.0-rc-002
  svu latest --released              # v1.2.0
  svu latest --without-v             # 1.3.0-rc-002
  svu latest --global                # including tags on other branches
  svu latest --tag-prefix api/       # v0.4.0, from the tag api/v0.4.0
  svu latest --at-head               # fails if the checked-out commit has no version tag
  svu l --directory path/to/repository`

func newLatestCommand(env environment) *cobra.Command {
	var (
		tags     tagFlags
		released bool
		atHead   bool
		vstyle   vFlags
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
			if err := vstyle.validate(); err != nil {
				return err
			}

			releaseFlag := ""
			if released {
				releaseFlag = "--released"
			}

			q, err := tags.query(releaseFlag, false)
			if err != nil {
				return err
			}

			if atHead && tags.global {
				return &userError{
					code:     ExitUsage,
					headline: "--at-head and --global can't be used together.",
					detail:   "--at-head only reads tags on the checked-out commit, so there's nothing for --global to add.",
					hint:     "Remove `--global`.",
				}
			}

			q.atHead = atHead

			v, name, err := latestTag(cmd.Context(), env.readTags, q)
			if err != nil {
				return err
			}

			return writeResult(cmd.OutOrStdout(), vstyle.format(v, q.source(name)))
		},
	}

	flags := cmd.Flags()
	flags.SortFlags = false

	tags.add(cmd)

	flags.BoolVar(&released, "released", false, "Skip pre-releases, and print the latest release")
	setHelpGroup(flags, "released", groupFilter)

	flags.BoolVar(&atHead, "at-head", false,
		"Only read tags on the checked-out commit itself, as in a check for whether it's tagged yet")
	setHelpGroup(flags, "at-head", groupFilter)

	vstyle.add(cmd)

	return cmd
}
