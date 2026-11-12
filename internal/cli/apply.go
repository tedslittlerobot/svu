package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tedslittlerobot/svu/internal/git"
	"github.com/tedslittlerobot/svu/internal/semver"
)

// tagCreator tags the checked-out commit in the repository containing dir.
// A nil editor makes a lightweight tag; otherwise git asks for a message in
// the user's editor. git.CreateTag is the real implementation.
type tagCreator func(ctx context.Context, dir, name string, editor *git.Editor) error

// applyFlags are svu modify's flags for tagging the new version.
type applyFlags struct {
	apply    bool
	annotate bool
}

// add adds --apply and --apply-annotate to cmd.
func (f *applyFlags) add(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.BoolVar(&f.apply, "apply", false, "Tag the checked-out commit with the new version, as git tag does")
	flags.BoolVar(&f.annotate, "apply-annotate", false,
		"Make an annotated tag, writing its message in your editor, as git tag --annotate does")

	setHelpGroup(flags, "apply", groupTagging)
	setHelpGroup(flags, "apply-annotate", groupTagging)
}

// tag tags the checked-out commit with v, if --apply or --apply-annotate was
// given. The tag's name is the --tag-prefix, then printed: v as the command
// prints it. If v is already tagged, with or without a "v", nothing is tagged.
func (f *applyFlags) tag(cmd *cobra.Command, env environment, q tagQuery, v semver.Version, printed string) error {
	if !f.apply && !f.annotate {
		return nil
	}

	tags, err := env.readTags(cmd.Context(), q.dir, git.TagOptions{All: true, Prefix: q.prefix})
	if err != nil {
		if errors.Is(err, git.ErrNotRepository) || errors.Is(err, git.ErrNotInstalled) {
			return applyError(err, q, "", f.annotate)
		}

		return tagsError(err, q)
	}

	if existing := tagName(tags, v); existing != "" {
		return &userError{
			code:     ExitTagExists,
			headline: fmt.Sprintf("%s is already tagged, as %q.", v, q.prefix+existing),
			detail:   "--apply only makes new tags, so the version wasn't tagged or printed.",
			hint:     fmt.Sprintf("To see where it points, run `git show %s`.", q.prefix+existing),
		}
	}

	name := q.prefix + printed

	var editor *git.Editor

	if f.annotate {
		// The editor gets the terminal. stdout is kept for the version.
		editor = &git.Editor{In: cmd.InOrStdin(), Out: cmd.ErrOrStderr()}
	}

	if err := env.createTag(cmd.Context(), q.dir, name, editor); err != nil {
		return applyError(err, q, name, f.annotate)
	}

	return nil
}

// applyError explains a failure to tag name. annotate is true if git was
// running the user's editor.
func applyError(err error, q tagQuery, name string, annotate bool) *userError {
	var cmdErr *git.CommandError

	switch {
	case errors.Is(err, git.ErrTagExists):
		return &userError{
			code:     ExitTagExists,
			headline: fmt.Sprintf("The tag %q already exists.", name),
			detail:   "--apply only makes new tags, so the version wasn't tagged or printed.",
			hint:     fmt.Sprintf("To see where it points, run `git show %s`.", name),
			err:      err,
		}
	case errors.Is(err, git.ErrNotRepository):
		headline := "The current directory is not in a git repository."
		if q.dir != "" {
			headline = fmt.Sprintf("%q is not in a git repository.", q.dir)
		}

		return &userError{
			code:     ExitRepository,
			headline: headline,
			detail:   "--apply tags the new version, so svu needs a git repository.",
			hint:     "Run svu inside a git repository, or leave out `--apply`.",
			err:      err,
		}
	case errors.Is(err, git.ErrNotInstalled):
		return &userError{
			code:     ExitGit,
			headline: "svu could not find git.",
			detail:   "--apply needs git to tag the new version.",
			hint:     "Install git, or leave out `--apply`.",
			err:      err,
		}
	case errors.As(err, &cmdErr):
		// With an editor, git's own explanation is already on the terminal.
		ue := &userError{
			code:     ExitGit,
			headline: fmt.Sprintf("git could not make the tag %q.", name),
			detail:   cmdErr.Stderr,
			err:      err,
		}

		if annotate {
			ue.hint = "If your editor didn't open, or the message was empty, check git's core.editor setting, " +
				"or the GIT_EDITOR, VISUAL and EDITOR variables."
		}

		return ue
	default:
		return tagsError(err, q)
	}
}
