// Package gittest creates throwaway git repositories for tests.
package gittest

import (
	"os/exec"
	"testing"
)

// NewRepository creates a git repository in a temporary directory, makes one
// empty commit, adds each of tags to it as a lightweight tag, and returns the
// repository's path.
//
// git runs with the user's global and system configuration turned off, so
// settings such as commit signing can't affect the test.
func NewRepository(t *testing.T, tags ...string) string {
	t.Helper()

	dir := t.TempDir()

	Run(t, dir, "init", "--quiet")
	Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "Initial commit")

	for _, tag := range tags {
		Run(t, dir, "tag", tag)
	}

	return dir
}

// Run runs git with args in dir, isolated from the user's configuration, and
// fails the test if it fails.
func Run(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "git", args...) //nolint:gosec // G204: args come from the test that calls Run.
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=semvu tests",
		"GIT_AUTHOR_EMAIL=tests@example.com",
		"GIT_COMMITTER_NAME=semvu tests",
		"GIT_COMMITTER_EMAIL=tests@example.com",
	)

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
