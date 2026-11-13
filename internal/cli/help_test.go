package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

//nolint:misspell // The help names the --color alias.
const rootHelp = `semvu (Semantic Version Utilities) works with semantic versions.

semvu uses a strict subset of SemVer 2.0.0: MAJOR.MINOR.PATCH, optionally
followed by a stage (alpha, beta or rc) and a three-digit index, as in
1.2.3-rc-001.

Usage:
  semvu <command>

Examples:
  semvu modify 1.2.3 --minor     # 1.3.0
  semvu m --patch --rc           # the latest git tag's next patch, as an rc
  semvu latest --released        # the latest released version in git tags
  semvu help modify              # how to modify a version

Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Show help for semvu or one of its commands
  latest, l   Print the latest version in a repository's git tags
  modify, m   Modify a version, such as bumping it or moving it to a new stage
  validate    Check that a version is valid, by exit status
  version     Print detailed build information for semvu

Flags:
  -h, --help         Show this help
  -v, --version      Print semvu's version
      --colour when  Use colour in help and errors: auto (the default), always or never (alias --color)

Run semvu help <command> for help with a command.
`

//nolint:misspell // The help names the --color alias.
const modifyHelp = `Modify a version and print the result.

semvu modify reads one version:

  - from the version argument, or from standard input if the argument is "-";
  - with no argument, from the latest git tag that is a valid version, found
    the same way as semvu latest finds it (see semvu help latest).

It applies any modifiers, then prints the result on standard output. A
version modifier on its own removes any stage. Add a stage modifier, as in
--minor --alpha, to start that stage at 001. Stages never go backwards,
unless --allow-stage-regression is given: then --alpha, --beta or --rc may
move a pre-release back to an earlier stage. Read from git tags, it carries
on after the highest index already tagged at that stage, so 1.2.3-rc-001
goes back to 1.2.3-alpha-003 if 1.2.3-alpha-002 is tagged. A release still
can't go back into a stage.

The result keeps the "v" of the version it read, from the argument, standard
input, the tag or --initial: v1.2.3 gives v1.2.4, and 1.2.3 gives 1.2.4. Use
--with-v or --without-v to choose instead.

--no-git makes sure semvu modify never uses git: a version must be given, and
flags that read or make tags, such as --auto and --apply, can't be used.

--hotfix patches the latest release, skipping pre-releases, to fix what's
already out while a pre-release is in progress.

Read from git tags, a version can't move on from a pre-release that hasn't
been released: with 1.2.3-rc-001 the latest tag, --minor fails until 1.2.3 is
tagged on some branch. Release it with --release, or add --skip-release to
move on anyway. This applies to --auto too.

--for-major reads only the tags with that MAJOR version, to work on an older
major version: with v1.4.2 and v2.0.0 tagged, --for-major 1 --patch gives
v1.4.3. The result must stay in that major version, so --major can't be used
with it.

--auto works out the bump from the Conventional Commits since the latest tag:
a breaking change bumps MAJOR (MINOR before 1.0.0), feat bumps MINOR, and fix
and perf bump PATCH. If the latest tag is a pre-release that already covers
the bump, it carries on from it. If no commit needs a new version, semvu modify
fails with exit status 8. --explain says on stderr how the bump was chosen.

--apply also tags the checked-out commit with the new version, exactly as
it's printed (after any --tag-prefix). --apply-annotate makes an annotated
tag, and opens your editor for its message. Neither pushes the tag. If the
version is already tagged, nothing is tagged or printed, and semvu modify fails
with exit status 9.

Usage:
  semvu modify [version] [flags]

Aliases:
  modify, m

Examples:
  semvu modify 1.2.3 --minor            # 1.3.0
  semvu modify 1.2.3 --minor --alpha    # 1.3.0-alpha-001
  semvu modify 1.2.3-rc-001 --rc        # 1.2.3-rc-002
  semvu modify 1.2.3-rc-001 --release   # 1.2.3
  echo v1.2.3 | semvu modify - --patch  # v1.2.4
  semvu modify v1.2.3 --without-v       # 1.2.3
  semvu modify "$VERSION" --patch --no-git   # never falls back to the git tags
  semvu m --patch                       # the latest git tag, bumped, as in v1.2.4
  semvu m --hotfix                      # 1.2.4, while 1.3.0-rc-002 is in progress
  semvu m --for-major 1 --patch         # v1.4.3, while v2.0.0 is out
  semvu m --initial 0.0.0 --minor       # 0.1.0 in a repository with no tags yet
  semvu m --auto --explain              # the bump the commits since the latest tag need
  semvu m --auto --rc                   # the same, as a release candidate
  semvu m --auto --apply                # tag the checked-out commit with it
  semvu modify 1.2.3-rc-001 --alpha --allow-stage-regression   # 1.2.3-alpha-001

Input:
      --directory path          Read tags from the git repository containing path
      --global                  Read every tag in the repository, not only those on the checked-out commit's history
      --tag-prefix prefix       Only read tags starting with prefix, such as api/ for api/v1.2.3
      --initial version         Use version if no tag matches, as in a new repository
      --no-git                  Never use git: read no tags or commits, and make no tags

Filter:
      --for-major major         Only read tags with this MAJOR version, as in 1 for v1.4.2

Version modifiers (use at most one):
      --major                   Bump MAJOR, and reset MINOR and PATCH to 0
      --minor                   Bump MINOR, and reset PATCH to 0
      --patch                   Bump PATCH
      --hotfix                  Bump PATCH of the latest release, skipping pre-releases
      --auto                    Work out the bump from the Conventional Commits since the latest tag

Version options:
      --skip-release            Let a new version start before the latest pre-release is released

Stage modifiers (use at most one):
      --alpha                   Move to alpha-001, or to the next alpha
      --beta                    Move to beta-001, or to the next beta
      --rc                      Move to rc-001, or to the next rc
      --release                 Remove the stage from a pre-release

Stage options:
      --allow-stage-regression  Let --alpha, --beta or --rc go back to an earlier stage

Output:
      --with-v                  Always print the version with a leading "v"
      --without-v               Never print a leading "v"
      --explain                 With --auto, say on stderr how the bump was chosen

Tagging:
      --apply                   Tag the checked-out commit with the new version, as git tag does
      --apply-annotate          Make an annotated tag, writing its message in your editor, as git tag --annotate does

Other:
  -h, --help                    Show this help

Global flags:
      --colour when             Use colour in help and errors: auto (the default), always or never (alias --color)
`

//nolint:misspell // The help names the --color alias.
const latestHelp = `Print the latest version in a repository's git tags.

semvu latest reads the tags of the git repository that contains the current
directory, or --directory if given. Only tags on the checked-out commit's
history count, unless --global is given. It skips any tag that isn't a valid
version, ignores a leading "v", and prints the latest by semvu's ordering.
semvu modify reads tags the same way when given no version.

With --tag-prefix, only tags starting with the prefix count, and the prefix
is ignored. With --for-major, only tags with that MAJOR version count. With
--released, pre-releases are skipped too. With --initial, that version is
printed if no tag matches.

It prints the version as the tag has it, with or without a "v", unless
--with-v or --without-v says otherwise.

With --at-head, only tags on the checked-out commit itself count. It fails
(exit status 6) if that commit has no version tag, so a script can check
whether it has already tagged a commit.

Usage:
  semvu latest [flags]

Aliases:
  latest, l

Examples:
  semvu latest                         # v1.3.0-rc-002, from the tag v1.3.0-rc-002
  semvu latest --released              # v1.2.0
  semvu latest --without-v             # 1.3.0-rc-002
  semvu latest --global                # including tags on other branches
  semvu latest --tag-prefix api/       # v0.4.0, from the tag api/v0.4.0
  semvu latest --for-major 1           # v1.4.2, while v2.0.0 is out
  semvu latest --at-head               # fails if the checked-out commit has no version tag
  semvu l --directory path/to/repository

Input:
      --directory path     Read tags from the git repository containing path
      --global             Read every tag in the repository, not only those on the checked-out commit's history
      --tag-prefix prefix  Only read tags starting with prefix, such as api/ for api/v1.2.3
      --initial version    Use version if no tag matches, as in a new repository

Filter:
      --for-major major    Only read tags with this MAJOR version, as in 1 for v1.4.2
      --released           Skip pre-releases, and print the latest release
      --at-head            Only read tags on the checked-out commit itself, as in a check for whether it's tagged yet

Output:
      --with-v             Always print the version with a leading "v"
      --without-v          Never print a leading "v"

Other:
  -h, --help               Show this help

Global flags:
      --colour when        Use colour in help and errors: auto (the default), always or never (alias --color)
`

//nolint:misspell // The help names the --color alias.
const validateHelp = `Check that a version is valid.

semvu validate prints nothing. It exits with status 0 if the version is valid,
or 3 if it isn't, and then explains why on standard error. Give the version
as the argument, or "-" to read it from standard input.

A leading "v" is accepted, as everywhere in semvu.

Usage:
  semvu validate <version> [flags]

Examples:
  semvu validate 1.2.3-rc-001
  semvu validate 1.2.3-alpha-1                 # exit status 3, and the reason
  echo "$TAG" | semvu validate -
  semvu validate "$TAG" 2>/dev/null && echo valid

Flags:
  -h, --help         Show this help

Global flags:
      --colour when  Use colour in help and errors: auto (the default), always or never (alias --color)
`

//nolint:misspell // The help names the --color alias.
const versionHelp = `Print detailed build information for semvu

Usage:
  semvu version [flags]

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
		{name: "shows help for the validate command", args: []string{"validate", "-h"}, want: validateHelp},
		{name: "shows help for help validate", args: []string{"help", "validate"}, want: validateHelp, toStdout: true},
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
				"  \x1b[36msemvu\x1b[0m \x1b[2m<command>\x1b[0m",        // Command names are cyan,
				"  \x1b[36mmodify, m\x1b[0m   Modify",                   // aliases included.
				"  \x1b[36m-h, --help\x1b[0m         Show",              // Flags are cyan.
				"      \x1b[36m--colour\x1b[0m \x1b[2mwhen\x1b[0m  Use", // Placeholders are dim.
				"semvu modify 1.2.3 --minor     \x1b[2m# 1.3.0\x1b[0m",  // Example comments are dim.
				"Run \x1b[36msemvu help <command>\x1b[0m for help",
			},
		},
		{
			name:  "colours modify help",
			args:  []string{"modify", "--help", "--colour=always"},
			plain: modifyHelp,
			spans: []string{
				"\x1b[1mStage modifiers (use at most one):\x1b[0m",
				"  \x1b[36msemvu modify\x1b[0m \x1b[2m[version] [flags]\x1b[0m",
				"\x1b[1mAliases:\x1b[0m\n  \x1b[36mmodify, m\x1b[0m",
				"      \x1b[36m--directory\x1b[0m \x1b[2mpath\x1b[0m          Read",
				"\n\x1b[36m--hotfix\x1b[0m patches the latest release",     // Flags in descriptions are cyan,
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

	if want := "Generate the autocompletion script for semvu"; !strings.HasPrefix(got.stdout, want) {
		t.Errorf("stdout = %q, want it to start with %q", got.stdout, want)
	}

	// A shell still gets its script on stdout.
	if got := invoke(t, invocation{args: []string{"completion", "bash"}}); !strings.Contains(got.stdout, "bash completion") {
		t.Errorf("semvu completion bash stdout = %q, want a bash completion script", got.stdout[:min(len(got.stdout), 80)])
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
			want: "Error: \"frobnicate\" is not an semvu command.\n\nHint: To list semvu's commands, run semvu --help.\n",
		},
		{
			name: "suggests a command close to a typo",
			args: []string{"modfy"},
			want: "Error: \"modfy\" is not an semvu command.\n\nHint: Did you mean semvu modify?\n",
		},
		{
			name: "suggests modify for a bare version",
			args: []string{"v1.2.3"},
			want: "Error: \"v1.2.3\" is not an semvu command.\n\nHint: To modify a version, run semvu modify v1.2.3.\n",
		},
		{
			name: "rejects an unknown shell for completion",
			args: []string{"completion", "nope"},
			want: "Error: \"nope\" is not a command of semvu completion.\n\nHint: To list them, run semvu help completion.\n",
		},
		{
			name: "suggests a shell close to a typo",
			args: []string{"completion", "bsah"},
			want: "Error: \"bsah\" is not a command of semvu completion.\n\nHint: Did you mean semvu completion bash?\n",
		},
		{
			name: "rejects help for an unknown command",
			args: []string{"help", "frobnicate"},
			want: "Error: \"frobnicate\" is not an semvu command.\n\nHint: To list semvu's commands, run semvu --help.\n",
		},
		{
			name: "rejects help for an unknown subcommand",
			args: []string{"help", "modify", "extra"},
			want: "Error: \"modify extra\" is not an semvu command.\n\nHint: To list semvu's commands, run semvu --help.\n",
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
			_, code, _ = strings.Cut(code, "semvu ") // Skip anything piped in, as in "echo 1.2.3 |".

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
