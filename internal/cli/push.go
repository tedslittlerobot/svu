package cli

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/spf13/cobra"

	"github.com/tedslittlerobot/semvu/internal/git"
)

// remoteLister returns the names of the remotes of the git repository
// containing dir. git.Remotes is the real implementation.
type remoteLister func(ctx context.Context, dir string) ([]string, error)

// tagPusher pushes the tag name to remote, from the repository containing
// dir. git.PushTag is the real implementation.
type tagPusher func(ctx context.Context, dir, remote, name string) error

// The flags for pushing the tag that --apply makes.
const (
	pushFlag   = "push"
	remoteFlag = "remote"
)

// addPush adds --push and --remote to cmd.
func (f *applyFlags) addPush(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.BoolVar(&f.push, pushFlag, false, "Push the new tag to the only remote, or to origin if there are several")
	flags.StringVar(&f.remote, remoteFlag, "", "Push to the remote called `name` instead")

	setHelpGroup(flags, pushFlag, groupTagging)
	setHelpGroup(flags, remoteFlag, groupTagging)
}

// validate checks that --push comes with --apply or --apply-annotate, and
// that --remote comes with --push and names a remote.
func (f *applyFlags) validate(cmd *cobra.Command) error {
	remoteGiven := cmd.Flags().Changed(remoteFlag)

	switch {
	case f.push && !f.apply && !f.annotate:
		return &userError{
			code:     ExitUsage,
			headline: "--push only works with --apply or --apply-annotate.",
			detail:   "--push pushes the tag that --apply makes.",
			hint:     "Add `--apply`, or remove `--push`.",
		}
	case remoteGiven && !f.push:
		return &userError{
			code:     ExitUsage,
			headline: "--remote only works with --push.",
			detail:   "--remote chooses the remote that --push sends the new tag to.",
			hint:     "Add `--push`, or remove `--remote`.",
		}
	case remoteGiven && f.remote == "":
		return &userError{
			code:     ExitUsage,
			headline: "--remote can't be empty.",
			detail:   "--remote names the remote that --push sends the new tag to.",
			hint:     "Give a remote's name, as in `--remote origin`.",
		}
	default:
		return nil
	}
}

// chooseRemote returns the remote --push sends the tag to, or "" without
// --push. It reads the repository's remotes, and picks one as
// git.ChooseRemote does.
func (f *applyFlags) chooseRemote(ctx context.Context, env environment, q tagQuery) (string, error) {
	if !f.push {
		return "", nil
	}

	remotes, err := env.listRemotes(ctx, q.dir)
	if err != nil {
		var cmdErr *git.CommandError
		if errors.As(err, &cmdErr) && cmdErr.Stderr != "" {
			return "", &userError{
				code:     ExitGit,
				headline: "git could not list the remotes in " + q.place() + ".",
				detail:   cmdErr.Stderr,
				err:      err,
			}
		}

		return "", applyError(err, q, "", f.annotate)
	}

	remote, err := git.ChooseRemote(remotes, f.remote)
	if err != nil {
		return "", remoteError(err, q, remotes, f.remote)
	}

	return remote, nil
}

// remoteError explains why no remote could be chosen from remotes. asked is
// --remote, or "" if it wasn't given.
func remoteError(err error, q tagQuery, remotes []string, asked string) *userError {
	remotes = slices.Sorted(slices.Values(remotes))

	quoted := make([]string, len(remotes))
	for i, r := range remotes {
		quoted[i] = fmt.Sprintf("%q", r)
	}

	ue := &userError{code: ExitNoRemote, err: err}

	switch {
	case errors.Is(err, git.ErrUnknownRemote) && len(remotes) > 0:
		suggestion := remotes[0]
		if slices.Contains(remotes, git.DefaultRemote) {
			suggestion = git.DefaultRemote
		}

		ue.headline = fmt.Sprintf("%s has no remote called %q.", q.repository(), asked)
		ue.detail = fmt.Sprintf("Its remotes are %s, so semvu made no tag.", join(quoted, "and"))
		if len(remotes) == 1 {
			ue.detail = fmt.Sprintf("Its only remote is %s, so semvu made no tag.", quoted[0])
		}

		ue.hint = fmt.Sprintf("Choose one of its remotes, as in `--remote %s`.", suggestion)
	case errors.Is(err, git.ErrAmbiguousRemote):
		ue.headline = fmt.Sprintf("%s has %d remotes, and none is called %q.", q.repository(), len(remotes), git.DefaultRemote)
		ue.detail = fmt.Sprintf("Its remotes are %s. With several, --push uses %q,\nso semvu made no tag.",
			join(quoted, "and"), git.DefaultRemote)
		ue.hint = fmt.Sprintf("Choose one, as in `--remote %s`.", remotes[0])
	default:
		// ErrNoRemotes, or a remote asked for when there are none.
		ue.headline = q.repository() + " has no remotes."
		ue.detail = "--push sends the new tag to a remote, so semvu made no tag."

		name := asked
		if name == "" {
			name = git.DefaultRemote
		}

		ue.hint = fmt.Sprintf("Add one, as in `git remote add %s <url>`, or leave out `--push`.", name)
	}

	return ue
}

// pushError explains a failure to push the tag name to remote. The tag has
// been made by then, so the hint says how to push it by hand.
func pushError(err error, q tagQuery, remote, name string) *userError {
	command := fmt.Sprintf("git push %s %s", remote, name)
	if q.dir != "" {
		command = fmt.Sprintf("git -C %q push %s %s", q.dir, remote, name)
	}

	ue := &userError{
		code:     ExitGit,
		headline: fmt.Sprintf("semvu made the tag %q, but git could not push it to %q.", name, remote),
		hint:     fmt.Sprintf("To push it, run `%s`.", command),
		err:      err,
	}

	var cmdErr *git.CommandError
	if errors.As(err, &cmdErr) {
		ue.detail = cmdErr.Stderr
	}

	return ue
}
