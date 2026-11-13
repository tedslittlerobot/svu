package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tedslittlerobot/semvu/internal/git"
)

// noGitFlag stops semvu modify using git at all: it reads no tags or commits,
// and makes no tags.
const noGitFlag = "no-git"

// gitFlags are semvu modify's flags that need git, in the order help shows
// them. --no-git can't be used with any of them.
func gitFlags() []string {
	return append(append([]string{}, tagFlagNames...), autoFlag, skipReleaseFlag, "apply", "apply-annotate",
		pushFlag, remoteFlag)
}

// addNoGit adds --no-git to cmd.
func addNoGit(cmd *cobra.Command, noGit *bool) {
	flags := cmd.Flags()

	flags.BoolVar(noGit, noGitFlag, false, "Never use git: read no tags or commits, and make no tags")
	setHelpGroup(flags, noGitFlag, groupInput)
}

// validateNoGit checks that, with --no-git, no flag that needs git was given
// and there is a version to start from.
func validateNoGit(cmd *cobra.Command, args []string) error {
	for _, name := range gitFlags() {
		if cmd.Flags().Changed(name) {
			return &userError{
				code:     ExitUsage,
				headline: fmt.Sprintf("--%s can't be used with --%s.", name, noGitFlag),
				detail:   fmt.Sprintf("--%s stops semvu using git at all, and --%s needs it.", noGitFlag, name),
				hint:     fmt.Sprintf("Remove `--%s`, or remove `--%s`.", name, noGitFlag),
			}
		}
	}

	if len(args) > 0 {
		return nil
	}

	return &userError{
		code:     ExitUsage,
		headline: fmt.Sprintf("semvu modify --%s needs a version.", noGitFlag),
		detail: fmt.Sprintf("--%s stops semvu reading the latest version from git tags, so the version must be given.",
			noGitFlag),
		hint: fmt.Sprintf("Give a version, as in `semvu modify 1.2.3 --%s`, or pass `-` to read it from standard input.",
			noGitFlag),
	}
}

// errGitDisabled is returned by the git functions of an environment from
// withoutGit. validateNoGit should stop them ever being called, so reaching
// one is a bug.
var errGitDisabled = errors.New("git is disabled by --no-git")

// withoutGit returns env with every git function replaced by one that fails
// without running git, so that --no-git holds even if a check is missed.
func (env environment) withoutGit() environment {
	env.readTags = func(context.Context, string, git.TagOptions) ([]string, error) {
		return nil, errGitDisabled
	}
	env.readMessages = func(context.Context, string, string) ([]string, error) {
		return nil, errGitDisabled
	}
	env.createTag = func(context.Context, string, string, *git.Editor) error {
		return errGitDisabled
	}
	env.listRemotes = func(context.Context, string) ([]string, error) {
		return nil, errGitDisabled
	}
	env.pushTag = func(context.Context, string, string, string) error {
		return errGitDisabled
	}

	return env
}
