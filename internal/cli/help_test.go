package cli

import (
	"strings"
	"testing"
)

//nolint:misspell // The help names the --color alias.
const rootHelp = `svm (SemVer Modifier) reads and modifies semantic version strings.

svm reads one version:

  - from the version argument, or from standard input if the argument is "-";
  - with no argument, from the latest git tag that is a valid version, in the
    repository that contains the current directory, or --directory if given.

It applies any modifiers, then prints the result on standard output. A
version modifier on its own removes any stage. Add a stage modifier, as in
--minor --alpha, to start that stage at 001. Stages never go backwards.

Usage:
  svm [version] [flags]
  svm <command>

Examples:
  svm 1.2.3 --minor            # 1.3.0
  svm 1.2.3 --minor --alpha    # 1.3.0-alpha-001
  svm 1.2.3-rc-001 --rc        # 1.2.3-rc-002
  svm 1.2.3-rc-001 --release   # 1.2.3
  echo v1.2.3 | svm - --patch  # 1.2.4
  svm --patch --with-v-prefix  # the latest git tag, bumped, as in v1.2.4
  svm --directory path/to/repository

Commands:
  completion  Generate the autocompletion script for the specified shell
  version     Print detailed build information for svm

Input:
      --directory path  Read tags from the git repository containing path

Version modifiers (use at most one):
      --major           Bump MAJOR, and reset MINOR and PATCH to 0
      --minor           Bump MINOR, and reset PATCH to 0
      --patch           Bump PATCH

Stage modifiers (use at most one):
      --alpha           Move to alpha-001, or to the next alpha
      --beta            Move to beta-001, or to the next beta
      --rc              Move to rc-001, or to the next rc
      --release         Remove the stage from a pre-release

Output:
      --with-v-prefix   Print the version with a leading "v" (alias --vp)
      --colour when     Use colour on stderr: auto (the default), always or never (alias --color)

Other:
  -h, --help            Show this help
  -v, --version         Print svm's version

Run svm <command> --help for help with a command.
`

//nolint:misspell // The help names the --color alias.
const versionHelp = `Print detailed build information for svm

Usage:
  svm version [flags]

Flags:
  -h, --help         Show this help

Global flags:
      --colour when  Use colour on stderr: auto (the default), always or never (alias --color)
`

// Environment variables that other tools define, so they keep their spelling.
const (
	noColour    = "NO_COLOR"    //nolint:misspell // Defined by https://no-color.org.
	forceColour = "FORCE_COLOR" //nolint:misspell // Defined by other tools.
)

func TestHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		env  map[string]string
		want string
	}{
		{name: "prints help for svm to stderr", args: []string{"--help"}, want: rootHelp},
		{name: "prints help for -h", args: []string{"-h"}, want: rootHelp},
		{name: "prints help instead of running", args: []string{"1.2.3", "--major", "--help"}, want: rootHelp},
		{name: "prints help for a command", args: []string{"version", "--help"}, want: versionHelp},
		{
			name: "leaves out colour when the environment turns it off",
			args: []string{"--help"},
			env:  map[string]string{noColour: "1", forceColour: "1"},
			want: rootHelp,
		},
		{
			name: "leaves out colour with --colour=never",
			args: []string{"--help", "--colour=never"},
			env:  map[string]string{forceColour: "1"},
			want: rootHelp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, env: tt.env})
			assertResult(t, got, result{stderr: tt.want})
		})
	}
}

func TestHelpInColour(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		env  map[string]string
	}{
		{name: "uses colour with --colour=always", args: []string{"--help", "--colour=always"}},
		{name: "accepts the American alias", args: []string{"--help", "--color", "always"}}, //nolint:misspell // --color is the alias output.md requires.
		{name: "uses colour when the environment forces it", args: []string{"--help"}, env: map[string]string{forceColour: "1"}},
		{
			name: "lets --colour=always win over the environment",
			args: []string{"--help", "--colour=always"},
			env:  map[string]string{noColour: "1"},
		},
	}

	// Spans the help must contain when coloured, from conventions/output.md.
	wantSpans := []string{
		"\x1b[1mUsage:\x1b[0m", // Headings are bold.
		"\x1b[1mStage modifiers (use at most one):\x1b[0m",
		"  \x1b[36msvm\x1b[0m [version] [flags]", // Command names are cyan.
		"  \x1b[36mversion\x1b[0m     Print",
		"      \x1b[36m--directory\x1b[0m \x1b[2mpath\x1b[0m  Read", // Flags are cyan, placeholders dim.
		"  \x1b[36m-h, --help\x1b[0m            Show this help",
		"svm 1.2.3 --minor            \x1b[2m# 1.3.0\x1b[0m", // Example comments are dim.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, env: tt.env})

			if got.code != ExitOK || got.stdout != "" {
				t.Fatalf("exit code = %d, stdout = %q, want %d and empty", got.code, got.stdout, ExitOK)
			}

			for _, span := range wantSpans {
				if !strings.Contains(got.stderr, span) {
					t.Errorf("stderr does not contain %q", span)
				}
			}

			if plain := stripANSI(got.stderr); plain != rootHelp {
				t.Errorf("stderr without colour = %q, want %q", plain, rootHelp)
			}
		})
	}
}

func TestHelpCommandIsNotACommand(t *testing.T) {
	t.Parallel()

	assertResult(t, run(t, "help"), result{code: ExitFailure, stderr: `Error: "help" is not a valid version.

  help
  ^
  MAJOR, MINOR and PATCH must be whole numbers, as in 1.2.3.

Hint: To list svm's commands, run svm --help.
`})
}

func TestColourFlagRejectsUnknownModes(t *testing.T) {
	t.Parallel()

	assertResult(t, run(t, "1.2.3", "--colour=sometimes"), result{
		code:   ExitFailure,
		stderr: `svm: invalid argument "sometimes" for "--colour" flag: must be auto, always or never` + "\n",
	})
}

func TestColourFlagDoesNotAffectStdout(t *testing.T) {
	t.Parallel()

	assertResult(t, run(t, "1.2.3", "--colour=always", "--patch"), result{stdout: "1.2.4\n"})
}

// stripANSI removes SGR escape sequences such as "\x1b[1m".
func stripANSI(s string) string {
	var b strings.Builder

	for {
		start := strings.Index(s, "\x1b[")
		if start < 0 {
			break
		}

		end := strings.IndexByte(s[start:], 'm')
		if end < 0 {
			break
		}

		b.WriteString(s[:start])
		s = s[start+end+1:]
	}

	b.WriteString(s)

	return b.String()
}
