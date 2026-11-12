// Package git reads information from git repositories by running the git
// command.
//
// It is an adapter: it knows how to ask git for things, but nothing about
// versions. Callers decide what the results mean.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

// Errors returned by Tags. Use errors.Is to check for them.
var (
	// ErrNotInstalled means the git command could not be found.
	ErrNotInstalled = errors.New("git is not installed")
	// ErrNotRepository means the directory is not inside a git repository.
	ErrNotRepository = errors.New("not a git repository")
	// ErrDirectoryNotFound means the directory does not exist.
	ErrDirectoryNotFound = errors.New("directory not found")
	// ErrNotDirectory means the path exists but is not a directory.
	ErrNotDirectory = errors.New("not a directory")
)

// CommandError reports that git ran but failed for a reason other than the
// ones covered by the sentinel errors.
type CommandError struct {
	// Args are the arguments git was run with.
	Args []string
	// Stderr is what git wrote to its standard error, trimmed.
	Stderr string
	// Err is the underlying error from running the command.
	Err error
}

// Error implements error.
func (e *CommandError) Error() string {
	return fmt.Sprintf("running git %s: %v: %s", strings.Join(e.Args, " "), e.Err, e.Stderr)
}

// Unwrap returns the underlying error.
func (e *CommandError) Unwrap() error {
	return e.Err
}

// TagOptions chooses which tags Tags returns.
type TagOptions struct {
	// All lists every tag in the repository. Otherwise, Tags lists only the
	// tags reachable from HEAD: those on the commit that is checked out, or
	// on its history.
	All bool
	// AtHead lists only the tags on the commit that is checked out, and
	// overrides All.
	AtHead bool
	// Prefix, if set, keeps only the tags whose names start with it, and
	// removes it from the names returned. For example, with Prefix "api/",
	// the tag "api/v1.2.3" is returned as "v1.2.3" and "v2.0.0" is skipped.
	Prefix string
}

// Tags returns the names of the tags in the git repository that contains
// dir, in no particular order. An empty dir means the current directory.
func Tags(ctx context.Context, dir string, opts TagOptions) ([]string, error) {
	return tags(ctx, "git", dir, opts)
}

// tags is Tags with the git executable injected, so tests can simulate git
// being missing or failing.
func tags(ctx context.Context, gitPath, dir string, opts TagOptions) ([]string, error) {
	if dir != "" {
		if err := checkDirectory(dir); err != nil {
			return nil, err
		}
	}

	args := []string{"tag", "--list"}

	switch {
	case opts.AtHead:
		args = append(args, "--points-at", "HEAD")
	case !opts.All:
		args = append(args, "--merged", "HEAD")
	}

	out, err := run(ctx, gitPath, dir, args...)

	var cmdErr *CommandError

	switch {
	case errors.As(err, &cmdErr) && strings.Contains(cmdErr.Stderr, "malformed object name") &&
		strings.Contains(cmdErr.Stderr, "HEAD"):
		// A repository with no commits yet has nothing on or reachable from HEAD.
		return []string{}, nil
	case err != nil:
		return nil, err
	}

	names := []string{}

	for name := range strings.FieldsSeq(out) {
		if rest, ok := strings.CutPrefix(name, opts.Prefix); ok {
			names = append(names, rest)
		}
	}

	return names, nil
}

// Messages returns the full messages of the commits reachable from HEAD but
// not from the tag since, newest first. Merge commits are left out. An empty
// since means every commit reachable from HEAD. dir is as for Tags.
func Messages(ctx context.Context, dir, since string) ([]string, error) {
	return messages(ctx, "git", dir, since)
}

// messages is Messages with the git executable injected, for tests.
func messages(ctx context.Context, gitPath, dir, since string) ([]string, error) {
	if dir != "" {
		if err := checkDirectory(dir); err != nil {
			return nil, err
		}
	}

	revisions := "HEAD"
	if since != "" {
		// The full ref, so a branch with the same name can't be picked instead.
		revisions = "refs/tags/" + since + "..HEAD"
	}

	// Each message ends with a NUL, which can't appear in a commit message.
	out, err := run(ctx, gitPath, dir, "log", "--no-merges", "--format=%B%x00", revisions, "--")

	var cmdErr *CommandError

	switch {
	case since == "" && errors.As(err, &cmdErr) && strings.Contains(cmdErr.Stderr, "'HEAD'"):
		// A repository with no commits yet has no history to read.
		return []string{}, nil
	case err != nil:
		return nil, err
	}

	list := []string{}

	for message := range strings.SplitSeq(out, "\x00") {
		if message = strings.TrimSpace(message); message != "" {
			list = append(list, message)
		}
	}

	return list, nil
}

func checkDirectory(dir string) error {
	info, err := os.Stat(dir)

	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("%w: %s", ErrDirectoryNotFound, dir)
	case err != nil:
		return fmt.Errorf("checking directory: %w", err)
	case !info.IsDir():
		return fmt.Errorf("%w: %s", ErrNotDirectory, dir)
	}

	return nil
}

// run runs git with args in dir and returns its standard output.
func run(ctx context.Context, gitPath, dir string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, gitPath, args...) //nolint:gosec // G204: gitPath is "git" or a test double, and args are fixed by this package.
	cmd.Dir = dir
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// Untranslated messages, so that errors can be recognised below, and no
	// prompting for credentials.
	cmd.Env = append(cmd.Environ(), "LC_ALL=C", "GIT_TERMINAL_PROMPT=0")

	err := cmd.Run()

	var exitErr *exec.ExitError

	switch {
	case err == nil:
		return stdout.String(), nil
	case errors.Is(err, exec.ErrNotFound):
		return "", ErrNotInstalled
	case errors.As(err, &exitErr) && strings.Contains(stderr.String(), "not a git repository"):
		return "", ErrNotRepository
	default:
		return "", &CommandError{Args: args, Stderr: strings.TrimSpace(stderr.String()), Err: err}
	}
}
