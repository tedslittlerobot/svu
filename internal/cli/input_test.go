package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tedslittlerobot/svm/internal/git"
	"github.com/tedslittlerobot/svm/internal/git/gittest"
)

const reportLine = "Hint: If this keeps happening, please report it at https://github.com/tedslittlerobot/svm/issues.\n"

func TestInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		stdin      string
		tags       func(t *testing.T) tagReader
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		// From an argument.
		{
			name:       "prints a version given as an argument",
			args:       []string{"1.2.3"},
			wantStdout: "1.2.3\n",
		},
		{
			name:       "removes the v prefix by default",
			args:       []string{"v1.2.3-rc-001"},
			wantStdout: "1.2.3-rc-001\n",
		},
		{
			name:       "adds a v prefix with --with-v-prefix",
			args:       []string{"--with-v-prefix", "1.2.3"},
			wantStdout: "v1.2.3\n",
		},
		{
			name:       "accepts --vp as an alias for --with-v-prefix",
			args:       []string{"v1.2.3-beta-010", "--vp"},
			wantStdout: "v1.2.3-beta-010\n",
		},
		{
			name:     "rejects an invalid argument and points at the problem",
			args:     []string{"1.2.3-alpha-1"},
			wantCode: ExitFailure,
			wantStderr: `Error: "1.2.3-alpha-1" is not a valid version.

  1.2.3-alpha-1
              ^
  The stage index must be exactly three digits, from 001 to 999.

Hint: Did you mean "1.2.3-alpha-001"?
`,
		},
		{
			name:     "rejects an invalid argument with no suggestion",
			args:     []string{"1.2.3-dev-001"},
			wantCode: ExitFailure,
			wantStderr: `Error: "1.2.3-dev-001" is not a valid version.

  1.2.3-dev-001
        ^
  The stage must be alpha, beta or rc, in lowercase.
`,
		},
		{
			name:     "suggests help for an argument that looks like a command",
			args:     []string{"frobnicate"},
			wantCode: ExitFailure,
			wantStderr: `Error: "frobnicate" is not a valid version.

  frobnicate
  ^
  MAJOR, MINOR and PATCH must be whole numbers, as in 1.2.3.

Hint: To list svm's commands, run svm --help.
`,
		},
		{
			name:     "leaves out the pointer for an argument it can't line up",
			args:     []string{"1.2.3\t"},
			wantCode: ExitFailure,
			wantStderr: `Error: "1.2.3\t" is not a valid version.

  A version looks like 1.2.3, or 1.2.3-rc-001 for a pre-release.

Hint: Did you mean "1.2.3"?
`,
		},
		{
			name:     "rejects an empty argument",
			args:     []string{""},
			wantCode: ExitFailure,
			wantStderr: `Error: The version is empty.

  A version looks like 1.2.3, or 1.2.3-rc-001 for a pre-release.
`,
		},
		{
			name:     "rejects more than one argument",
			args:     []string{"1.2.3", "1.2.4"},
			wantCode: ExitFailure,
			wantStderr: `Error: svm was given 2 versions, but it takes at most one.

Hint: Give a single version, as in svm 1.2.3.
`,
		},

		// From standard input.
		{
			name:       "reads the version from standard input when the argument is -",
			args:       []string{"-"},
			stdin:      "v1.2.3\n",
			wantStdout: "1.2.3\n",
		},
		{
			name:       "adds a v prefix to a version from standard input",
			args:       []string{"--vp", "-"},
			stdin:      "1.2.3-rc-002",
			wantStdout: "v1.2.3-rc-002\n",
		},
		{
			name:       "ignores whitespace around standard input",
			args:       []string{"-"},
			stdin:      "  1.2.3 \r\n\n",
			wantStdout: "1.2.3\n",
		},
		{
			name:     "rejects an invalid version on standard input",
			args:     []string{"-"},
			stdin:    "V1.2.3\n",
			wantCode: ExitFailure,
			wantStderr: `Error: "V1.2.3" on standard input is not a valid version.

  V1.2.3
  ^
  The "v" prefix must be lowercase.

Hint: Did you mean "v1.2.3"?
`,
		},
		{
			name:     "does not suggest help for a word on standard input",
			args:     []string{"-"},
			stdin:    "latest\n",
			wantCode: ExitFailure,
			wantStderr: `Error: "latest" on standard input is not a valid version.

  latest
  ^
  MAJOR, MINOR and PATCH must be whole numbers, as in 1.2.3.
`,
		},
		{
			name:     "rejects empty standard input",
			args:     []string{"-"},
			stdin:    " \n",
			wantCode: ExitFailure,
			wantStderr: `Error: Standard input is empty.

  With "-" as the argument, svm reads one version from standard input.

Hint: Pipe a version into svm, as in echo 1.2.3 | svm -.
`,
		},
		{
			name:     "rejects more than one line on standard input",
			args:     []string{"-"},
			stdin:    "1.2.3\n1.2.4\n",
			wantCode: ExitFailure,
			wantStderr: `Error: Standard input has more than one line.

  With "-" as the argument, svm reads one version from standard input.
`,
		},
		{
			name:     "rejects --directory with an argument",
			args:     []string{"--directory", ".", "1.2.3"},
			wantCode: ExitFailure,
			wantStderr: `Error: --directory can't be used when a version is given.

  svm only reads git tags when no version is given.

Hint: Remove --directory, or remove the version.
`,
		},
		{
			name:     "rejects --directory with standard input",
			args:     []string{"--directory", ".", "-"},
			stdin:    "1.2.3",
			wantCode: ExitFailure,
			wantStderr: `Error: --directory can't be used when a version is given.

  svm only reads git tags when no version is given.

Hint: Remove --directory, or remove the version.
`,
		},

		// From git tags.
		{
			name: "prints the latest valid tag in the current directory",
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "", "v1.9.0", "nightly", "v1.10.0-rc-001", "v1.2.3-alpha-1")
			},
			wantStdout: "1.10.0-rc-001\n",
		},
		{
			name: "adds a v prefix to the latest tag",
			args: []string{"--with-v-prefix"},
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "", "1.0.0")
			},
			wantStdout: "v1.0.0\n",
		},
		{
			name: "reads tags in the directory given by --directory",
			args: []string{"--directory", "path/to/repository"},
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "path/to/repository", "v0.1.0")
			},
			wantStdout: "0.1.0\n",
		},
		{
			name: "reports that there is no input outside a git repository",
			tags: func(*testing.T) tagReader {
				return failingTags(git.ErrNotRepository)
			},
			wantCode: ExitFailure,
			wantStderr: `Error: svm has no valid input.

  No version was given, so svm reads the latest version from git tags,
  but the current directory is not in a git repository.

Hint: Give a version, as in svm 1.2.3, or run svm inside a git repository.
`,
		},
		{
			name: "reports that there is no input when --directory is not in a git repository",
			args: []string{"--directory", "elsewhere"},
			tags: func(*testing.T) tagReader {
				return failingTags(git.ErrNotRepository)
			},
			wantCode: ExitFailure,
			wantStderr: `Error: svm has no valid input.

  No version was given, so svm reads the latest version from git tags,
  but "elsewhere" is not in a git repository.

Hint: Give a version, as in svm 1.2.3, or set --directory to a git repository.
`,
		},
		{
			name: "reports a repository with no tags",
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "")
			},
			wantCode: ExitFailure,
			wantStderr: `Error: The git repository has no tags that are valid versions.

  It has no tags.

Hint: Give a version, as in svm 0.1.0, or tag a release, as in git tag v0.1.0.
`,
		},
		{
			name: "reports a repository with no valid tags",
			args: []string{"--directory", "elsewhere"},
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "elsewhere", "nightly", "1.2")
			},
			wantCode: ExitFailure,
			wantStderr: `Error: The git repository at "elsewhere" has no tags that are valid versions.

  None of its tags is a valid version, such as v1.2.3 or v1.2.3-rc-001.

Hint: Give a version, as in svm 0.1.0, or tag a release, as in git tag v0.1.0.
`,
		},
		{
			name: "reports a missing --directory",
			args: []string{"--directory", "missing"},
			tags: func(*testing.T) tagReader {
				return failingTags(fmt.Errorf("%w: missing", git.ErrDirectoryNotFound))
			},
			wantCode: ExitFailure,
			wantStderr: `Error: "missing" does not exist.

  --directory must be a directory in a git repository.
`,
		},
		{
			name: "reports a --directory that is a file",
			args: []string{"--directory", "go.mod"},
			tags: func(*testing.T) tagReader {
				return failingTags(fmt.Errorf("%w: go.mod", git.ErrNotDirectory))
			},
			wantCode: ExitFailure,
			wantStderr: `Error: "go.mod" is not a directory.

  --directory must be a directory in a git repository.
`,
		},
		{
			name: "reports that git is not installed",
			tags: func(*testing.T) tagReader {
				return failingTags(git.ErrNotInstalled)
			},
			wantCode: ExitFailure,
			wantStderr: `Error: svm could not find git.

  No version was given, so svm needs git to read the latest version from the tags.

Hint: Install git, or give a version, as in svm 1.2.3.
`,
		},
		{
			name: "passes on git's own explanation when git fails",
			tags: func(*testing.T) tagReader {
				return failingTags(&git.CommandError{Stderr: "fatal: detected dubious ownership", Err: errors.New("exit status 128")})
			},
			wantCode: ExitFailure,
			wantStderr: `Error: git could not list the tags in the current directory.

  fatal: detected dubious ownership
`,
		},
		{
			name: "asks for a report when reading tags fails unexpectedly",
			args: []string{"--directory", "elsewhere"},
			tags: func(*testing.T) tagReader {
				return failingTags(errors.New("checking directory: permission denied"))
			},
			wantCode:   ExitFailure,
			wantStderr: "Error: svm could not read the git tags in \"elsewhere\".\n\n" + reportLine,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inv := invocation{args: tt.args, stdin: strings.NewReader(tt.stdin)}
			if tt.tags != nil {
				inv.tags = tt.tags(t)
			}

			assertResult(t, invoke(t, inv), result{code: tt.wantCode, stdout: tt.wantStdout, stderr: tt.wantStderr})
		})
	}
}

func TestInputFromRealGitRepository(t *testing.T) {
	t.Parallel()

	repo := gittest.NewRepository(t, "v0.9.0", "v0.10.0-beta-002", "nightly", "v0.10.0-rc-001", "0.9.1")
	empty := gittest.NewRepository(t)
	notRepo := t.TempDir()

	tests := []struct {
		name string
		args []string
		want result
	}{
		{
			name: "prints the latest valid tag",
			args: []string{"--directory", repo},
			want: result{stdout: "0.10.0-rc-001\n"},
		},
		{
			name: "prints the latest valid tag with a v prefix",
			args: []string{"--directory", repo, "--vp"},
			want: result{stdout: "v0.10.0-rc-001\n"},
		},
		{
			name: "reports a repository with no tags",
			args: []string{"--directory", empty},
			want: result{code: ExitFailure, stderr: fmt.Sprintf(
				"Error: The git repository at %q has no tags that are valid versions.\n\n"+
					"  It has no tags.\n\n"+
					"Hint: Give a version, as in svm 0.1.0, or tag a release, as in git tag v0.1.0.\n", empty)},
		},
		{
			name: "reports a directory that is not in a repository",
			args: []string{"--directory", notRepo},
			want: result{code: ExitFailure, stderr: fmt.Sprintf("Error: svm has no valid input.\n\n"+
				"  No version was given, so svm reads the latest version from git tags,\n"+
				"  but %q is not in a git repository.\n\n"+
				"Hint: Give a version, as in svm 1.2.3, or set --directory to a git repository.\n", notRepo)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, tags: git.Tags})
			assertResult(t, got, tt.want)
		})
	}
}

func TestInputOnlyReadsStdinWhenAsked(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		tags func(t *testing.T) tagReader
		want string
	}{
		{name: "uses the argument without reading standard input", args: []string{"1.2.3"}, want: "1.2.3\n"},
		{
			name: "reads git tags without reading standard input",
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "", "v2.0.0")
			},
			want: "2.0.0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inv := invocation{args: tt.args, stdin: untouchableReader{t}}
			if tt.tags != nil {
				inv.tags = tt.tags(t)
			}

			assertResult(t, invoke(t, inv), result{stdout: tt.want})
		})
	}
}

// untouchableReader fails the test if it is read. It stands in for a pipe
// that never closes, which would make svm hang if it read it.
type untouchableReader struct{ t *testing.T }

func (r untouchableReader) Read([]byte) (int, error) {
	r.t.Error("svm read standard input, want it not to")

	return 0, io.EOF
}

func TestInputReadsPipedFiles(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "version")
	if err := os.WriteFile(path, []byte("v3.0.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = f.Close() })

	assertResult(t, invoke(t, invocation{args: []string{"-"}, stdin: f}), result{stdout: "3.0.0\n"})
}

func TestInputReportsUnreadableStdin(t *testing.T) {
	t.Parallel()

	got := invoke(t, invocation{args: []string{"-"}, stdin: errReader{}})

	assertResult(t, got, result{
		code:   ExitFailure,
		stderr: "Error: svm could not read standard input.\n\n" + reportLine,
	})
}

func TestOutputReportsWriteFailure(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer

	code := execute(t.Context(), []string{"1.2.3"}, Streams{
		In:  strings.NewReader(""),
		Out: errWriter{},
		Err: &stderr,
	}, environment{info: testInfo})

	want := "Error: svm could not write its output.\n\n" +
		"  The output stream was closed before svm finished writing.\n\n" + reportLine

	if code != ExitFailure {
		t.Errorf("exit code = %d, want %d", code, ExitFailure)
	}

	if stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// assertResult checks the exit code, stdout and stderr all match exactly.
func assertResult(t *testing.T, got, want result) {
	t.Helper()

	if got.code != want.code {
		t.Errorf("exit code = %d, want %d", got.code, want.code)
	}

	if got.stdout != want.stdout {
		t.Errorf("stdout = %q, want %q", got.stdout, want.stdout)
	}

	if got.stderr != want.stderr {
		t.Errorf("stderr = %q, want %q", got.stderr, want.stderr)
	}
}
