package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tedslittlerobot/svm/internal/git"
	"github.com/tedslittlerobot/svm/internal/semver"
)

// tagReader returns the tag names of the git repository containing dir, where
// "" means the current directory. git.Tags is the real implementation.
type tagReader func(ctx context.Context, dir string) ([]string, error)

// addDirectoryFlag adds --directory, which every command that reads git tags
// shares, to cmd.
func addDirectoryFlag(cmd *cobra.Command, dir *string) {
	cmd.Flags().StringVar(dir, "directory", "", "Read tags from the git repository containing `path`")
	setHelpGroup(cmd.Flags(), "directory", groupInput)
	_ = cmd.MarkFlagDirname("directory") // Only fails if the flag doesn't exist.
}

// tagQuery describes which version to find in a repository's git tags, and
// why, so that errors can be worded for the command that asked.
type tagQuery struct {
	// dir is the --directory flag, or "" for the current directory.
	dir string
	// releasesOnly skips pre-releases.
	releasesOnly bool
	// fallback is true when tags are read only because no version was given,
	// as svu modify does. Errors then suggest giving a version instead.
	fallback bool
}

// place names the directory the query reads, for messages.
func (q tagQuery) place() string {
	if q.dir == "" {
		return "the current directory"
	}

	return fmt.Sprintf("%q", q.dir)
}

// repository names the repository the query reads, for messages.
func (q tagQuery) repository() string {
	if q.dir == "" {
		return "The git repository"
	}

	return fmt.Sprintf("The git repository at %q", q.dir)
}

// latestTag returns the latest git tag that is a valid version: the same
// filtering and ordering for every command that reads tags.
func latestTag(ctx context.Context, readTags tagReader, q tagQuery) (semver.Version, error) {
	tags, err := readTags(ctx, q.dir)
	if err != nil {
		return semver.Version{}, tagsError(err, q)
	}

	latest, ok := semver.Latest(tags)
	if !ok {
		return semver.Version{}, noVersionTagsError(q, len(tags))
	}

	if !q.releasesOnly {
		return latest, nil
	}

	if release, ok := semver.LatestRelease(tags); ok {
		return release, nil
	}

	return semver.Version{}, &userError{
		headline: q.repository() + " has no released versions.",
		detail:   fmt.Sprintf("Its latest version, %q, is a pre-release.", latest.String()),
		hint:     "Leave out `--release` to include pre-releases.",
	}
}

func noVersionTagsError(q tagQuery, count int) *userError {
	detail := "None of its tags is a valid version, such as v1.2.3 or v1.2.3-rc-001."
	if count == 0 {
		detail = "It has no tags."
	}

	hint := "Tag a release, as in `git tag v0.1.0`."
	if q.fallback {
		hint = "Give a version, as in `svu modify 0.1.0`, or tag a release, as in `git tag v0.1.0`."
	}

	return &userError{
		headline: q.repository() + " has no tags that are valid versions.",
		detail:   detail,
		hint:     hint,
	}
}

func tagsError(err error, q tagQuery) *userError {
	var cmdErr *git.CommandError

	switch {
	case errors.Is(err, git.ErrNotRepository):
		return notRepositoryError(err, q)
	case errors.Is(err, git.ErrDirectoryNotFound):
		return &userError{
			headline: q.place() + " does not exist.",
			detail:   "--directory must be a directory in a git repository.",
			err:      err,
		}
	case errors.Is(err, git.ErrNotDirectory):
		return &userError{
			headline: q.place() + " is not a directory.",
			detail:   "--directory must be a directory in a git repository.",
			err:      err,
		}
	case errors.Is(err, git.ErrNotInstalled):
		ue := &userError{
			headline: "svu could not find git.",
			detail:   "svu needs git to read the latest version from the tags.",
			hint:     "Install git.",
			err:      err,
		}

		if q.fallback {
			ue.detail = "No version was given, so svu needs git to read the latest version from the tags."
			ue.hint = "Install git, or give a version, as in `svu modify 1.2.3`."
		}

		return ue
	case errors.As(err, &cmdErr) && cmdErr.Stderr != "":
		return &userError{
			headline: "git could not list the tags in " + q.place() + ".",
			detail:   cmdErr.Stderr,
			err:      err,
		}
	default:
		return &userError{headline: "svu could not read the git tags in " + q.place() + ".", hint: reportHint, err: err}
	}
}

func notRepositoryError(err error, q tagQuery) *userError {
	if q.fallback {
		hint := "Give a version, as in `svu modify 1.2.3`, or run svu inside a git repository."
		if q.dir != "" {
			hint = "Give a version, as in `svu modify 1.2.3`, or set `--directory` to a git repository."
		}

		return &userError{
			headline: "svu has no valid input.",
			detail: "No version was given, so svu reads the latest version from git tags,\n" +
				"but " + q.place() + " is not in a git repository.",
			hint: hint,
			err:  err,
		}
	}

	headline := fmt.Sprintf("%q is not in a git repository.", q.dir)
	hint := "Set `--directory` to a directory in a git repository."

	if q.dir == "" {
		headline = "The current directory is not in a git repository."
		hint = "Run svu inside a git repository, or set `--directory` to one."
	}

	return &userError{
		headline: headline,
		detail:   "svu reads the latest version from the repository's git tags.",
		hint:     hint,
		err:      err,
	}
}
