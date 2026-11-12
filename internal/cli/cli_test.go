package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/tedslittlerobot/svm/internal/buildinfo"
)

// testInfo is fixed build information so command output is deterministic.
var testInfo = buildinfo.Info{
	Version:   "1.2.3",
	Commit:    "abc123",
	Date:      "2026-01-01T00:00:00Z",
	GoVersion: "go1.26.4",
	Platform:  "linux/amd64",
}

// result captures everything observable about one svm invocation.
type result struct {
	code   int
	stdout string
	stderr string
}

// invocation describes one in-process run of svm.
type invocation struct {
	args []string
	// stdin is what svm reads from standard input. nil means empty.
	stdin io.Reader
	// tags stands in for git. nil fails the test if svm reads git tags.
	tags tagReader
}

// run invokes svm in-process with the given arguments, empty stdin and no
// access to git, using testInfo as the build information.
func run(t *testing.T, args ...string) result {
	t.Helper()

	return invoke(t, invocation{args: args})
}

// invoke runs svm in-process as described by inv.
func invoke(t *testing.T, inv invocation) result {
	t.Helper()

	if inv.stdin == nil {
		inv.stdin = strings.NewReader("")
	}

	if inv.tags == nil {
		inv.tags = func(context.Context, string) ([]string, error) {
			t.Error("svm read git tags, want it not to")

			return nil, errors.New("unexpected call to read git tags")
		}
	}

	var stdout, stderr bytes.Buffer

	code := execute(t.Context(), inv.args, Streams{
		In:  inv.stdin,
		Out: &stdout,
		Err: &stderr,
	}, testInfo, inv.tags)

	return result{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

// fixedTags returns a tagReader that always returns tags, and checks that it
// is asked about wantDir.
func fixedTags(t *testing.T, wantDir string, tags ...string) tagReader {
	t.Helper()

	return func(_ context.Context, dir string) ([]string, error) {
		if dir != wantDir {
			t.Errorf("read tags in %q, want %q", dir, wantDir)
		}

		return tags, nil
	}
}

// failingTags returns a tagReader that always fails with err.
func failingTags(err error) tagReader {
	return func(context.Context, string) ([]string, error) {
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
			name:       "help flag prints help",
			args:       []string{"--help"},
			wantCode:   ExitOK,
			wantStdout: "Usage:",
		},
		{
			name:        "version flag prints the bare version",
			args:        []string{"--version"},
			wantCode:    ExitOK,
			wantStdout:  "svm 1.2.3\n",
			exactStdout: true,
		},
		{
			name:       "help describes where the version comes from",
			args:       []string{"--help"},
			wantCode:   ExitOK,
			wantStdout: "from the latest git tag that is a valid version",
		},
		{
			name:       "an unknown command is read as an invalid version",
			args:       []string{"frobnicate"},
			wantCode:   ExitFailure,
			wantStderr: `Error: "frobnicate" is not a valid version.`,
		},
		{
			name:       "unknown flag fails",
			args:       []string{"--frobnicate"},
			wantCode:   ExitFailure,
			wantStderr: "svm: unknown flag: --frobnicate",
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

	want := "svm " + buildinfo.Get().Version + "\n"
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
