package git

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/tedslittlerobot/svu/internal/git/gittest"
)

func TestTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tags []string
		want []string
	}{
		{name: "lists no tags in a repository without any", tags: nil, want: []string{}},
		{
			name: "lists every tag, valid version or not",
			tags: []string{"v1.0.0", "nightly", "1.2.3-rc-001"},
			want: []string{"1.2.3-rc-001", "nightly", "v1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := gittest.NewRepository(t, tt.tags...)

			got, err := Tags(t.Context(), dir, TagOptions{})
			if err != nil {
				t.Fatalf("Tags() error = %v, want nil", err)
			}

			slices.Sort(got)

			if !slices.Equal(got, tt.want) {
				t.Errorf("Tags() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTagsFindsTheRepositoryFromASubdirectory(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepository(t, "v0.1.0")

	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}

	got, err := Tags(t.Context(), sub, TagOptions{})
	if err != nil {
		t.Fatalf("Tags() error = %v, want nil", err)
	}

	if want := []string{"v0.1.0"}; !slices.Equal(got, want) {
		t.Errorf("Tags() = %q, want %q", got, want)
	}
}

func TestTagsErrors(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		gitPath string
		dir     string
		want    error
	}{
		{name: "reports a directory outside any repository", gitPath: "git", dir: t.TempDir(), want: ErrNotRepository},
		{
			name: "reports a missing directory", gitPath: "git",
			dir: filepath.Join(t.TempDir(), "missing"), want: ErrDirectoryNotFound,
		},
		{name: "reports a path that is a file", gitPath: "git", dir: file, want: ErrNotDirectory},
		{
			name: "reports git not being installed", gitPath: "svu-test-git-that-does-not-exist",
			dir: t.TempDir(), want: ErrNotInstalled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tags(t.Context(), tt.gitPath, tt.dir, TagOptions{})
			if !errors.Is(err, tt.want) {
				t.Errorf("tags() = %q, %v, want error %v", got, err, tt.want)
			}
		})
	}
}

func TestTagsReportsOtherGitFailures(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fakeGit := filepath.Join(dir, "git")
	script := "#!/bin/sh\necho 'fatal: something broke' >&2\nexit 2\n"

	if err := os.WriteFile(fakeGit, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}

	_, err := tags(t.Context(), fakeGit, dir, TagOptions{})

	var cmdErr *CommandError
	if !errors.As(err, &cmdErr) {
		t.Fatalf("tags() error = %v, want *CommandError", err)
	}

	if want := "fatal: something broke"; cmdErr.Stderr != want {
		t.Errorf("CommandError.Stderr = %q, want %q", cmdErr.Stderr, want)
	}

	want := "running git tag --list --merged HEAD: exit status 2: fatal: something broke"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}

	var exitErr interface{ ExitCode() int }
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 2 {
		t.Errorf("tags() error does not unwrap to an exit code of 2: %v", err)
	}
}

func TestTagsScopeAndPrefix(t *testing.T) {
	t.Parallel()

	// main: v1.0.0, then a commit with v1.1.0-rc-001 and api/v2.0.0.
	// support, branched from v1.0.0: v1.0.1.
	dir := gittest.NewRepository(t, "v1.0.0")
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "Feature")
	gittest.Run(t, dir, "tag", "v1.1.0-rc-001")
	gittest.Run(t, dir, "tag", "api/v2.0.0")
	gittest.Run(t, dir, "checkout", "--quiet", "-b", "support", "v1.0.0")
	gittest.Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "Fix")
	gittest.Run(t, dir, "tag", "v1.0.1")

	tests := []struct {
		name string
		opts TagOptions
		want []string
	}{
		{name: "lists only tags reachable from HEAD by default", opts: TagOptions{}, want: []string{"v1.0.0", "v1.0.1"}},
		{
			name: "lists every tag with All",
			opts: TagOptions{All: true},
			want: []string{"api/v2.0.0", "v1.0.0", "v1.0.1", "v1.1.0-rc-001"},
		},
		{name: "keeps and strips a prefix", opts: TagOptions{All: true, Prefix: "api/"}, want: []string{"v2.0.0"}},
		{name: "finds nothing reachable with a prefix only used elsewhere", opts: TagOptions{Prefix: "api/"}, want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Tags(t.Context(), dir, tt.opts)
			if err != nil {
				t.Fatalf("Tags() error = %v, want nil", err)
			}

			slices.Sort(got)

			if !slices.Equal(got, tt.want) {
				t.Errorf("Tags(%+v) = %q, want %q", tt.opts, got, tt.want)
			}
		})
	}
}

func TestTagsInARepositoryWithNoCommits(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gittest.Run(t, dir, "init", "--quiet")

	for _, opts := range []TagOptions{{}, {All: true}} {
		got, err := Tags(t.Context(), dir, opts)
		if err != nil || len(got) != 0 {
			t.Errorf("Tags(%+v) = %q, %v, want no tags and no error", opts, got, err)
		}
	}
}
