package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/tedslittlerobot/svu/internal/buildinfo"
	"github.com/tedslittlerobot/svu/internal/git"
)

// testInfo is fixed build information so command output is deterministic.
var testInfo = buildinfo.Info{
	Version:   "1.2.3",
	Commit:    "abc123",
	Date:      "2026-01-01T00:00:00Z",
	GoVersion: "go1.26.4",
	Platform:  "linux/amd64",
}

// result captures everything observable about one svu invocation.
type result struct {
	code   int
	stdout string
	stderr string
}

// invocation describes one in-process run of svu.
type invocation struct {
	args []string
	// stdin is what svu reads from standard input. nil means empty.
	stdin io.Reader
	// tags stands in for git. nil fails the test if svu reads git tags.
	tags tagReader
	// messages stands in for git's commit log. nil fails the test if svu
	// reads it.
	messages messageReader
	// createTag stands in for git tag. nil fails the test if svu tags.
	createTag tagCreator
	// env is the environment variables svu sees. nil means none.
	env map[string]string
}

func (inv invocation) getenv() func(string) string {
	return func(key string) string { return inv.env[key] }
}

// run invokes svu in-process with the given arguments, empty stdin and no
// access to git, using testInfo as the build information.
func run(t *testing.T, args ...string) result {
	t.Helper()

	return invoke(t, invocation{args: args})
}

// invoke runs svu in-process as described by inv.
func invoke(t *testing.T, inv invocation) result {
	t.Helper()

	if inv.stdin == nil {
		inv.stdin = strings.NewReader("")
	}

	if inv.tags == nil {
		inv.tags = func(context.Context, string, git.TagOptions) ([]string, error) {
			t.Error("svu read git tags, want it not to")

			return nil, errors.New("unexpected call to read git tags")
		}
	}

	if inv.messages == nil {
		inv.messages = func(context.Context, string, string) ([]string, error) {
			t.Error("svu read commit messages, want it not to")

			return nil, errors.New("unexpected call to read commit messages")
		}
	}

	if inv.createTag == nil {
		inv.createTag = func(context.Context, string, string, *git.Editor) error {
			t.Error("svu made a tag, want it not to")

			return errors.New("unexpected call to make a tag")
		}
	}

	var stdout, stderr bytes.Buffer

	code := execute(t.Context(), inv.args, Streams{
		In:  inv.stdin,
		Out: &stdout,
		Err: &stderr,
	}, environment{info: testInfo, readTags: inv.tags, readMessages: inv.messages, createTag: inv.createTag, getenv: inv.getenv()})

	return result{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

// fixedTags returns a tagReader for a repository where every tag is
// reachable from HEAD, and checks that it is asked about wantDir. Like git, it
// applies the prefix option.
func fixedTags(t *testing.T, wantDir string, tags ...string) tagReader {
	t.Helper()

	return scopedTags(t, wantDir, tags, nil)
}

// scopedTags returns a tagReader for a repository where reachable tags are on
// the checked-out commit's history and others are only on other branches.
func scopedTags(t *testing.T, wantDir string, reachable, others []string) tagReader {
	t.Helper()

	return func(_ context.Context, dir string, opts git.TagOptions) ([]string, error) {
		if dir != wantDir {
			t.Errorf("read tags in %q, want %q", dir, wantDir)
		}

		tags := slices.Clone(reachable)
		if opts.All {
			tags = append(tags, others...)
		}

		names := []string{}

		for _, tag := range tags {
			if rest, ok := strings.CutPrefix(tag, opts.Prefix); ok {
				names = append(names, rest)
			}
		}

		return names, nil
	}
}

// failingTags returns a tagReader that always fails with err.
func failingTags(err error) tagReader {
	return func(context.Context, string, git.TagOptions) ([]string, error) {
		return nil, err
	}
}

func TestRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		args        []string
		wantCode    int
		wantStdout  string // substring; "" means stdout must be empty
		wantStderr  string // substring; "" means stderr must be empty
		exactStdout bool
	}{
		{
			name:        "version flag prints the bare version",
			args:        []string{"--version"},
			wantCode:    ExitOK,
			wantStdout:  "1.2.3\n",
			exactStdout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := run(t, tt.args...)

			if got.code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", got.code, tt.wantCode)
			}

			assertOutput(t, "stdout", got.stdout, tt.wantStdout, tt.exactStdout)
			assertOutput(t, "stderr", got.stderr, tt.wantStderr, false)
		})
	}
}

func TestExecuteUsesRealBuildInfo(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	code := Execute(t.Context(), []string{"--version"}, Streams{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
	})

	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d (stderr: %q)", code, ExitOK, stderr.String())
	}

	want := buildinfo.Get().Version + "\n"
	if stdout.String() != want {
		t.Errorf("stdout = %q, want %q", stdout.String(), want)
	}
}

// assertOutput checks a captured stream. An empty want asserts the stream is
// empty; otherwise want must match exactly or appear as a substring.
func assertOutput(t *testing.T, stream, got, want string, exact bool) {
	t.Helper()

	switch {
	case want == "" && got != "":
		t.Errorf("%s = %q, want it empty", stream, got)
	case exact && got != want:
		t.Errorf("%s = %q, want %q", stream, got, want)
	case !exact && !strings.Contains(got, want):
		t.Errorf("%s = %q, want it to contain %q", stream, got, want)
	}
}
