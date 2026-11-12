package cli

import (
	"bytes"
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

// run invokes svm in-process with the given arguments and empty stdin, using
// testInfo as the build information.
func run(t *testing.T, args ...string) result {
	t.Helper()

	var stdout, stderr bytes.Buffer

	code := execute(t.Context(), args, Streams{
		In:  strings.NewReader(""),
		Out: &stdout,
		Err: &stderr,
	}, testInfo)

	return result{code: code, stdout: stdout.String(), stderr: stderr.String()}
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
			name:       "no arguments prints help",
			args:       nil,
			wantCode:   ExitOK,
			wantStdout: "Usage:",
		},
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
			name:       "unknown command fails",
			args:       []string{"frobnicate"},
			wantCode:   ExitFailure,
			wantStderr: `svm: unknown command "frobnicate"`,
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
