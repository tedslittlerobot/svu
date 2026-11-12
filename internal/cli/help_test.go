package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

//nolint:misspell // The help names the --color alias.
const rootHelp = `svu (Semantic Version Utilities) works with semantic versions.

svu uses a strict subset of SemVer 2.0.0: MAJOR.MINOR.PATCH, optionally
followed by a stage (alpha, beta or rc) and a three-digit index, as in
1.2.3-rc-001.

Usage:
  svu <command>

Examples:
  svu modify 1.2.3 --minor     # 1.3.0
  svu m --patch --rc           # the latest git tag's next patch, as an rc
  svu latest --released        # the latest released version in git tags
  svu help modify              # how to modify a version

Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Show help for svu or one of its commands
  latest, l   Print the latest version in a repository's git tags
  modify, m   Modify a version, such as bumping it or moving it to a new stage
  version     Print detailed build information for svu

Flags:
  -h, --help         Show this help
  -v, --version      Print svu's version
      --colour when  Use colour in help and errors: auto (the default), always or never (alias --color)

Run svu help <command> for help with a command.
`

//nolint:misspell // The help names the --color alias.
const modifyHelp = `Modify a version and print the result.

svu modify reads one version:

  - from the version argument, or from standard input if the argument is "-";
  - with no argument, from the latest git tag that is a valid version, found
    the same way as svu latest finds it (see svu help latest).

It applies any modifiers, then prints the result on standard output. A
version modifier on its own removes any stage. Add a stage modifier, as in
--minor --alpha, to start that stage at 001. Stages never go backwards.

--hotfix patches the latest release, skipping pre-releases, to fix what's
already out while a pre-release is in progress.

Usage:
  svu modify [version] [flags]

Aliases:
  modify, m

Examples:
  svu modify 1.2.3 --minor            # 1.3.0
  svu modify 1.2.3 --minor --alpha    # 1.3.0-alpha-001
  svu modify 1.2.3-rc-001 --rc        # 1.2.3-rc-002
  svu modify 1.2.3-rc-001 --release   # 1.2.3
  echo v1.2.3 | svu modify - --patch  # 1.2.4
  svu m --patch --with-v-prefix       # the latest git tag, bumped, as in v1.2.4
  svu m --hotfix                      # 1.2.4, while 1.3.0-rc-002 is in progress
  svu m --initial 0.0.0 --minor       # 0.1.0 in a repository with no tags yet

Input:
      --directory path     Read tags from the git repository containing path
      --global             Read every tag in the repository, not only those on the checked-out commit's history
      --tag-prefix prefix  Only read tags starting with prefix, such as api/ for api/v1.2.3
      --initial version    Use version if no tag matches, as in a new repository

Version modifiers (use at most one):
      --major              Bump MAJOR, and reset MINOR and PATCH to 0
      --minor              Bump MINOR, and reset PATCH to 0
      --patch              Bump PATCH
      --hotfix             Bump PATCH of the latest release, skipping pre-releases

Stage modifiers (use at most one):
      --alpha              Move to alpha-001, or to the next alpha
      --beta               Move to beta-001, or to the next beta
      --rc                 Move to rc-001, or to the next rc
      --release            Remove the stage from a pre-release

Output:
      --with-v-prefix      Print the version with a leading "v" (alias --vp)

Other:
  -h, --help               Show this help

Global flags:
      --colour when        Use colour in help and errors: auto (the default), always or never (alias --color)
`

//nolint:misspell // The help names the --color alias.
const latestHelp = `Print the latest version in a repository's git tags.

svu latest reads the tags of the git repository that contains the current
directory, or --directory if given. Only tags on the checked-out commit's
history count, unless --global is given. It skips any tag that isn't a valid
version, ignores a leading "v", and prints the latest by svu's ordering.
svu modify reads tags the same way when given no version.

With --tag-prefix, only tags starting with the prefix count, and the prefix
is ignored. With --released, pre-releases are skipped too. With --initial,
that version is printed if no tag matches.

Usage:
  svu latest [flags]

Aliases:
  latest, l

Examples:
  svu latest                         # 1.3.0-rc-002
  svu latest --released              # 1.2.0
  svu latest --with-v-prefix         # v1.3.0-rc-002
  svu latest --global                # including tags on other branches
  svu latest --tag-prefix api/       # 0.4.0, from the tag api/v0.4.0
  svu l --directory path/to/repository

Input:
      --directory path     Read tags from the git repository containing path
      --global             Read every tag in the repository, not only those on the checked-out commit's history
      --tag-prefix prefix  Only read tags starting with prefix, such as api/ for api/v1.2.3
      --initial version    Use version if no tag matches, as in a new repository

Filter:
      --released           Skip pre-releases, and print the latest release

Output:
      --with-v-prefix      Print the version with a leading "v" (alias --vp)

Other:
  -h, --help               Show this help

Global flags:
      --colour when        Use colour in help and errors: auto (the default), always or never (alias --color)
`

//nolint:misspell // The help names the --color alias.
const versionHelp = `Print detailed build information for svu

Usage:
  svu version [flags]

Flags:
  -h, --help         Show this help

Global flags:
      --colour when  Use colour in help and errors: auto (the default), always or never (alias --color)
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
		// toStdout is true for the help command, whose main output is help.
		toStdout bool
	}{
		{name: "shows general help with no command", args: nil, want: rootHelp, toStdout: true},
		{name: "shows general help for --help", args: []string{"--help"}, want: rootHelp},
		{name: "shows general help for -h", args: []string{"-h"}, want: rootHelp},
		{name: "shows general help for the help command", args: []string{"help"}, want: rootHelp, toStdout: true},
		{name: "shows modify help for modify --help", args: []string{"modify", "--help"}, want: modifyHelp},
		{name: "shows modify help for the alias", args: []string{"m", "-h"}, want: modifyHelp},
		{name: "shows modify help for help modify", args: []string{"help", "modify"}, want: modifyHelp, toStdout: true},
		{name: "shows modify help for help with the alias", args: []string{"help", "m"}, want: modifyHelp, toStdout: true},
		{
			name: "shows help instead of modifying",
			args: []string{"modify", "1.2.3", "--major", "--help"},
			want: modifyHelp,
		},
		{name: "shows help for the version command", args: []string{"version", "--help"}, want: versionHelp},
		{name: "shows help for the latest command", args: []string{"latest", "--help"}, want: latestHelp},
		{name: "shows help for help l", args: []string{"help", "l"}, want: latestHelp, toStdout: true},
		{name: "shows help for help version", args: []string{"help", "version"}, want: versionHelp, toStdout: true},
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

			want := result{stderr: tt.want}
			if tt.toStdout {
				want = result{stdout: tt.want}
			}

			assertResult(t, invoke(t, invocation{args: tt.args, env: tt.env}), want)
		})
	}
}

func TestHelpInColour(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		args  []string
		env   map[string]string
		plain string
		spans []string
	}{
		{
			name:  "colours general help with --colour=always",
			args:  []string{"--help", "--colour=always"},
			plain: rootHelp,
			spans: []string{
				"\x1b[1mUsage:\x1b[0m",                                  // Headings are bold.
				"  \x1b[36msvu\x1b[0m \x1b[2m<command>\x1b[0m",          // Command names are cyan,
				"  \x1b[36mmodify, m\x1b[0m   Modify",                   // aliases included.
				"  \x1b[36m-h, --help\x1b[0m         Show",              // Flags are cyan.
				"      \x1b[36m--colour\x1b[0m \x1b[2mwhen\x1b[0m  Use", // Placeholders are dim.
				"svu modify 1.2.3 --minor     \x1b[2m# 1.3.0\x1b[0m",    // Example comments are dim.
				"Run \x1b[36msvu help <command>\x1b[0m for help",
			},
		},
		{
			name:  "colours modify help",
			args:  []string{"modify", "--help", "--colour=always"},
			plain: modifyHelp,
			spans: []string{
				"\x1b[1mStage modifiers (use at most one):\x1b[0m",
				"  \x1b[36msvu modify\x1b[0m \x1b[2m[version] [flags]\x1b[0m",
				"\x1b[1mAliases:\x1b[0m\n  \x1b[36mmodify, m\x1b[0m",
				"      \x1b[36m--directory\x1b[0m \x1b[2mpath\x1b[0m     Read",
				"\n\x1b[36m--hotfix\x1b[0m patches the latest release",     // Flags in descriptions are cyan,
				"(alias \x1b[36m--vp\x1b[0m)",                              // in flag descriptions too,
				"as in\n\x1b[36m--minor\x1b[0m \x1b[36m--alpha\x1b[0m, to", // even at the start of a line.
			},
		},
		{
			name:  "accepts the American alias",
			args:  []string{"--help", "--color", "always"}, //nolint:misspell // --color is the alias output.md requires.
			plain: rootHelp,
			spans: []string{"\x1b[1mUsage:\x1b[0m"},
		},
		{
			name:  "uses colour when the environment forces it",
			args:  []string{"modify", "--help"},
			env:   map[string]string{forceColour: "1"},
			plain: modifyHelp,
			spans: []string{"\x1b[1mUsage:\x1b[0m"},
		},
		{
			name:  "lets --colour=always win over the environment",
			args:  []string{"--help", "--colour=always"},
			env:   map[string]string{noColour: "1"},
			plain: rootHelp,
			spans: []string{"\x1b[1mUsage:\x1b[0m"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, env: tt.env})

			if got.code != ExitOK || got.stdout != "" {
				t.Fatalf("exit code = %d, stdout = %q, want %d and empty", got.code, got.stdout, ExitOK)
			}

			for _, span := range tt.spans {
				if !strings.Contains(got.stderr, span) {
					t.Errorf("stderr does not contain %q", span)
				}
			}

			if plain := stripANSI(got.stderr); plain != tt.plain {
				t.Errorf("stderr without colour = %q, want %q", plain, tt.plain)
			}
		})
	}
}

func TestHelpCommandColoursStdout(t *testing.T) {
	t.Parallel()

	got := invoke(t, invocation{args: []string{"help", "--colour=always"}})

	if got.code != ExitOK || got.stderr != "" {
		t.Fatalf("exit code = %d, stderr = %q, want %d and empty", got.code, got.stderr, ExitOK)
	}

	if !strings.Contains(got.stdout, "\x1b[1mUsage:\x1b[0m") {
		t.Errorf("stdout = %q, want coloured help", got.stdout)
	}

	if plain := stripANSI(got.stdout); plain != rootHelp {
		t.Errorf("stdout without colour = %q, want %q", plain, rootHelp)
	}
}

func TestCommandGroupsShowHelpOnStdout(t *testing.T) {
	t.Parallel()

	got := invoke(t, invocation{args: []string{"completion"}})

	if got.code != ExitOK || got.stderr != "" {
		t.Fatalf("exit code = %d, stderr = %q, want %d and empty", got.code, got.stderr, ExitOK)
	}

	if want := "Generate the autocompletion script for svu"; !strings.HasPrefix(got.stdout, want) {
		t.Errorf("stdout = %q, want it to start with %q", got.stdout, want)
	}

	// A shell still gets its script on stdout.
	if got := invoke(t, invocation{args: []string{"completion", "bash"}}); !strings.Contains(got.stdout, "bash completion") {
		t.Errorf("svu completion bash stdout = %q, want a bash completion script", got.stdout[:min(len(got.stdout), 80)])
	}
}

func TestUnknownCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "rejects an unknown command",
			args: []string{"frobnicate"},
			want: "Error: \"frobnicate\" is not an svu command.\n\nHint: To list svu's commands, run svu --help.\n",
		},
		{
			name: "suggests a command close to a typo",
			args: []string{"modfy"},
			want: "Error: \"modfy\" is not an svu command.\n\nHint: Did you mean svu modify?\n",
		},
		{
			name: "suggests modify for a bare version",
			args: []string{"v1.2.3"},
			want: "Error: \"v1.2.3\" is not an svu command.\n\nHint: To modify a version, run svu modify v1.2.3.\n",
		},
		{
			name: "rejects an unknown shell for completion",
			args: []string{"completion", "nope"},
			want: "Error: \"nope\" is not a command of svu completion.\n\nHint: To list them, run svu help completion.\n",
		},
		{
			name: "suggests a shell close to a typo",
			args: []string{"completion", "bsah"},
			want: "Error: \"bsah\" is not a command of svu completion.\n\nHint: Did you mean svu completion bash?\n",
		},
		{
			name: "rejects help for an unknown command",
			args: []string{"help", "frobnicate"},
			want: "Error: \"frobnicate\" is not an svu command.\n\nHint: To list svu's commands, run svu --help.\n",
		},
		{
			name: "rejects help for an unknown subcommand",
			args: []string{"help", "modify", "extra"},
			want: "Error: \"modify extra\" is not an svu command.\n\nHint: To list svu's commands, run svu --help.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertResult(t, run(t, tt.args...), result{code: ExitUsage, stderr: tt.want})
		})
	}
}

func TestColourFlagRejectsUnknownModes(t *testing.T) {
	t.Parallel()

	assertResult(t, run(t, "modify", "1.2.3", "--colour=sometimes"), result{
		code:   ExitUsage,
		stderr: "Error: \"sometimes\" is not a valid value for --colour.\n\n  --colour must be auto, always or never.\n",
	})
}

func TestColourFlagDoesNotAffectStdout(t *testing.T) {
	t.Parallel()

	assertResult(t, run(t, "modify", "1.2.3", "--colour=always", "--patch"), result{stdout: "1.2.4\n"})
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

func TestHelpCommandCompletesCommandNames(t *testing.T) {
	t.Parallel()

	// cobra's hidden __complete command is what shells call on Tab.
	got := invoke(t, invocation{args: []string{"__complete", "help", ""}})

	for _, want := range []string{"modify\tModify a version", "version\tPrint detailed build information"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("completions = %q, want them to contain %q", got.stdout, want)
		}
	}

	got = invoke(t, invocation{args: []string{"__complete", "help", "modify", ""}})
	if strings.Contains(got.stdout, "modify\t") {
		t.Errorf("completions after a command = %q, want no more command names", got.stdout)
	}
}

// TestExamplesUseRealFlags parses every example in every command's help, so
// that renaming a flag can't leave an example that no longer works.
func TestExamplesUseRealFlags(t *testing.T) {
	t.Parallel()

	var colourMode colourFlag

	root := newRootCommand(Streams{}, environment{info: testInfo}, &colourMode)

	var commands []*cobra.Command

	var collect func(*cobra.Command)
	collect = func(c *cobra.Command) {
		commands = append(commands, c)
		for _, sub := range c.Commands() {
			collect(sub)
		}
	}
	collect(root)

	checked := 0

	for _, c := range commands {
		for line := range strings.SplitSeq(c.Example, "\n") {
			code, _, _ := strings.Cut(line, "#")
			_, code, _ = strings.Cut(code, "svu ") // Skip anything piped in, as in "echo 1.2.3 |".

			args := strings.Fields(code)
			if len(args) == 0 {
				continue
			}

			target, rest, err := root.Find(args)
			if err != nil {
				t.Errorf("example %q: %v", line, err)

				continue
			}

			target.InitDefaultHelpFlag()

			if err := target.ParseFlags(rest); err != nil {
				t.Errorf("example %q doesn't parse: %v", strings.TrimSpace(line), err)
			}

			checked++
		}
	}

	if checked < 10 {
		t.Errorf("checked %d examples, want every command's examples", checked)
	}
}
