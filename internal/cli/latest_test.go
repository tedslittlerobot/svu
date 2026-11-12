package cli

import (
	"fmt"
	"testing"

	"github.com/tedslittlerobot/svm/internal/git"
	"github.com/tedslittlerobot/svm/internal/git/gittest"
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
		{name: "prints the latest release with --release", args: []string{"latest", "--release"}, tags: tags, want: "1.10.0\n"},
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
		args []string
		tags func(t *testing.T) tagReader
		want string
	}{
		{
			name: "rejects arguments",
			args: []string{"latest", "1.2.3"},
			want: "Error: svu latest takes no arguments, but was given \"1.2.3\".\n\nHint: Run svu latest on its own.\n",
		},
		{
			name: "reports a repository with no tags",
			args: []string{"latest"},
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "")
			},
			want: "Error: The git repository has no tags that are valid versions.\n\n" +
				"  It has no tags.\n\n" +
				"Hint: Tag a release, as in git tag v0.1.0.\n",
		},
		{
			name: "reports a repository with no valid tags",
			args: []string{"latest", "--directory", "elsewhere"},
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "elsewhere", "nightly")
			},
			want: "Error: The git repository at \"elsewhere\" has no tags that are valid versions.\n\n" +
				"  None of its tags is a valid version, such as v1.2.3 or v1.2.3-rc-001.\n\n" +
				"Hint: Tag a release, as in git tag v0.1.0.\n",
		},
		{
			name: "reports a repository with only pre-releases for --release",
			args: []string{"latest", "--release"},
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "", "v1.0.0-beta-003", "v1.0.0-rc-001")
			},
			want: "Error: The git repository has no released versions.\n\n" +
				"  Its latest version, \"1.0.0-rc-001\", is a pre-release.\n\n" +
				"Hint: Leave out --release to include pre-releases.\n",
		},
		{
			name: "reports the current directory outside a repository",
			args: []string{"latest"},
			tags: func(*testing.T) tagReader { return failingTags(git.ErrNotRepository) },
			want: "Error: The current directory is not in a git repository.\n\n" +
				"  svu reads the latest version from the repository's git tags.\n\n" +
				"Hint: Run svu inside a git repository, or set --directory to one.\n",
		},
		{
			name: "reports --directory outside a repository",
			args: []string{"latest", "--directory", "elsewhere"},
			tags: func(*testing.T) tagReader { return failingTags(git.ErrNotRepository) },
			want: "Error: \"elsewhere\" is not in a git repository.\n\n" +
				"  svu reads the latest version from the repository's git tags.\n\n" +
				"Hint: Set --directory to a directory in a git repository.\n",
		},
		{
			name: "reports a missing --directory",
			args: []string{"latest", "--directory", "missing"},
			tags: func(*testing.T) tagReader {
				return failingTags(fmt.Errorf("%w: missing", git.ErrDirectoryNotFound))
			},
			want: "Error: \"missing\" does not exist.\n\n  --directory must be a directory in a git repository.\n",
		},
		{
			name: "reports that git is not installed",
			args: []string{"latest"},
			tags: func(*testing.T) tagReader { return failingTags(git.ErrNotInstalled) },
			want: "Error: svu could not find git.\n\n" +
				"  svu needs git to read the latest version from the tags.\n\n" +
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

			assertResult(t, invoke(t, inv), result{code: ExitFailure, stderr: tt.want})
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
		{name: "prints the latest tag", args: []string{"latest", "--directory", repo}, want: "0.10.0-rc-001\n"},
		{name: "prints the latest release", args: []string{"l", "--directory", repo, "--release"}, want: "0.9.1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, tags: git.Tags})
			assertResult(t, got, result{stdout: tt.want})
		})
	}
}
