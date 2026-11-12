package cli

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/tedslittlerobot/semvu/internal/git"
	"github.com/tedslittlerobot/semvu/internal/git/gittest"
)

func TestLatest(t *testing.T) {
	t.Parallel()

	tags := []string{"v1.9.0", "v1.10.0", "v1.10.1-rc-002", "nightly", "v1.2.3-alpha-1", "2.0.0-alpha-001"}

	tests := []struct {
		name    string
		args    []string
		wantDir string
		tags    []string
		want    string
	}{
		{name: "prints the latest version, pre-releases included", args: []string{"latest"}, tags: tags, want: "2.0.0-alpha-001\n"},
		{name: "answers to its alias", args: []string{"l"}, tags: tags, want: "2.0.0-alpha-001\n"},
		{name: "prints the latest release with --released", args: []string{"latest", "--released"}, tags: tags, want: "v1.10.0\n"},
		{name: "removes the tag's v with --without-v", args: []string{"latest", "--without-v"}, tags: tags, want: "2.0.0-alpha-001\n"},
		{name: "combines --without-v with --released", args: []string{"l", "--released", "--without-v"}, tags: tags, want: "1.10.0\n"},
		{
			name:    "reads tags in the directory given by --directory",
			args:    []string{"l", "--directory", "path/to/repository"},
			wantDir: "path/to/repository",
			tags:    []string{"0.1.0"},
			want:    "0.1.0\n",
		},
		{
			name: "matches the version modify reads with no version",
			args: []string{"modify"},
			tags: tags,
			want: "2.0.0-alpha-001\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, tags: fixedTags(t, tt.wantDir, tt.tags...)})
			assertResult(t, got, result{stdout: tt.want})
		})
	}
}

func TestLatestErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code int
		args []string
		tags func(t *testing.T) tagReader
		want string
	}{
		{
			name: "rejects arguments",
			code: ExitUsage,
			args: []string{"latest", "1.2.3"},
			want: "Error: semvu latest takes no arguments, but was given \"1.2.3\".\n\nHint: Run semvu latest on its own.\n",
		},
		{
			name: "reports a repository with no tags",
			code: ExitNoMatchingTag,
			args: []string{"latest"},
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "")
			},
			want: "Error: The git repository has no tags that are valid versions.\n\n" +
				"  It has no tags on the checked-out commit's history.\n\n" +
				"Hint: Tag a release, as in git tag v0.1.0, or use --initial 0.0.0 to start from scratch.\n",
		},
		{
			name: "reports a repository with no valid tags",
			code: ExitNoMatchingTag,
			args: []string{"latest", "--directory", "elsewhere"},
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "elsewhere", "nightly")
			},
			want: "Error: The git repository at \"elsewhere\" has no tags that are valid versions.\n\n" +
				"  None of its tags on the checked-out commit's history is a valid version, such as v1.2.3 or v1.2.3-rc-001.\n\n" +
				"Hint: Tag a release, as in git tag v0.1.0, or use --initial 0.0.0 to start from scratch.\n",
		},
		{
			name: "reports a repository with only pre-releases for --released",
			code: ExitNoMatchingTag,
			args: []string{"latest", "--released"},
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "", "v1.0.0-beta-003", "v1.0.0-rc-001")
			},
			want: "Error: The git repository has no released versions.\n\n" +
				"  Its latest version, \"1.0.0-rc-001\", is a pre-release.\n\n" +
				"Hint: Leave out --released to use pre-releases too.\n",
		},
		{
			name: "reports the current directory outside a repository",
			code: ExitRepository,
			args: []string{"latest"},
			tags: func(*testing.T) tagReader { return failingTags(git.ErrNotRepository) },
			want: "Error: The current directory is not in a git repository.\n\n" +
				"  semvu reads the latest version from the repository's git tags.\n\n" +
				"Hint: Run semvu inside a git repository, or set --directory to one.\n",
		},
		{
			name: "reports --directory outside a repository",
			code: ExitRepository,
			args: []string{"latest", "--directory", "elsewhere"},
			tags: func(*testing.T) tagReader { return failingTags(git.ErrNotRepository) },
			want: "Error: \"elsewhere\" is not in a git repository.\n\n" +
				"  semvu reads the latest version from the repository's git tags.\n\n" +
				"Hint: Set --directory to a directory in a git repository.\n",
		},
		{
			name: "reports a missing --directory",
			code: ExitRepository,
			args: []string{"latest", "--directory", "missing"},
			tags: func(*testing.T) tagReader {
				return failingTags(fmt.Errorf("%w: missing", git.ErrDirectoryNotFound))
			},
			want: "Error: \"missing\" does not exist.\n\n  --directory must be a directory in a git repository.\n",
		},
		{
			name: "reports that git is not installed",
			code: ExitGit,
			args: []string{"latest"},
			tags: func(*testing.T) tagReader { return failingTags(git.ErrNotInstalled) },
			want: "Error: semvu could not find git.\n\n" +
				"  semvu needs git to read the latest version from the tags.\n\n" +
				"Hint: Install git.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inv := invocation{args: tt.args}
			if tt.tags != nil {
				inv.tags = tt.tags(t)
			}

			assertResult(t, invoke(t, inv), result{code: tt.code, stderr: tt.want})
		})
	}
}

func TestLatestFromRealGitRepository(t *testing.T) {
	t.Parallel()

	repo := gittest.NewRepository(t, "v0.9.0", "v0.10.0-beta-002", "nightly", "v0.10.0-rc-001", "0.9.1")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "prints the latest tag", args: []string{"latest", "--directory", repo}, want: "v0.10.0-rc-001\n"},
		{name: "prints the latest release", args: []string{"l", "--directory", repo, "--released"}, want: "0.9.1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, tags: git.Tags})
			assertResult(t, got, result{stdout: tt.want})
		})
	}
}

// headTags returns a tagReader for a repository where head are the tags on
// the checked-out commit, and history are other tags on its history.
func headTags(t *testing.T, head, history []string) tagReader {
	t.Helper()

	return func(_ context.Context, _ string, opts git.TagOptions) ([]string, error) {
		if opts.AtHead {
			return head, nil
		}

		return append(slices.Clone(head), history...), nil
	}
}

func TestLatestAtHead(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		head    []string
		history []string
		want    result
	}{
		{
			name:    "prints the version tag on the checked-out commit",
			args:    []string{"latest", "--at-head"},
			head:    []string{"v1.2.3", "deployed"},
			history: []string{"v1.3.0"},
			want:    result{stdout: "v1.2.3\n"},
		},
		{
			name:    "fails when the checked-out commit has no version tag",
			args:    []string{"latest", "--at-head"},
			history: []string{"v1.2.3"},
			want: result{code: ExitNoMatchingTag, stderr: "Error: The git repository has no tags that are valid versions.\n\n" +
				"  It has no tags on the checked-out commit.\n\n" +
				"Hint: The checked-out commit has no version tag yet. Tag it, or leave out --at-head.\n"},
		},
		{
			name: "rejects --at-head with --global",
			args: []string{"latest", "--at-head", "--global"},
			want: result{code: ExitUsage, stderr: "Error: --at-head and --global can't be used together.\n\n" +
				"  --at-head only reads tags on the checked-out commit, so there's nothing for --global to add.\n\n" +
				"Hint: Remove --global.\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, tags: headTags(t, tt.head, tt.history)})
			assertResult(t, got, tt.want)
		})
	}
}

func TestLatestAtHeadInRealGitRepository(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepository(t, "v1.0.0")
	gittest.Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "Untagged")

	got := invoke(t, invocation{args: []string{"latest", "--at-head", "--directory", dir}, tags: git.Tags})
	if got.code != ExitNoMatchingTag {
		t.Errorf("before tagging: exit code = %d, want %d", got.code, ExitNoMatchingTag)
	}

	gittest.Run(t, dir, "tag", "v1.0.1")

	got = invoke(t, invocation{args: []string{"latest", "--at-head", "--directory", dir}, tags: git.Tags})
	assertResult(t, got, result{stdout: "v1.0.1\n"})
}
