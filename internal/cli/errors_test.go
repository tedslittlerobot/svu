package cli

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/tedslittlerobot/semvu/internal/colour"
)

func TestUserErrorKeepsTheCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("disk on fire")

	tests := []struct {
		name string
		err  *userError
		want string
	}{
		{name: "describes itself by its headline", err: &userError{headline: "Something failed."}, want: "Something failed."},
		{
			name: "includes the cause in its description",
			err:  &userError{headline: "Something failed.", err: cause},
			want: "Something failed.: disk on fire",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}

			if got, want := errors.Unwrap(tt.err), tt.err.err; !errors.Is(got, want) {
				t.Errorf("Unwrap() = %v, want %v", got, want)
			}
		})
	}
}

func TestRenderNeverShowsTheCause(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer

	render(&b, &userError{headline: "Something failed.", err: errors.New("disk on fire")}, colour.Styler{})

	if got, want := b.String(), "Error: Something failed.\n"; got != want {
		t.Errorf("render() wrote %q, want %q", got, want)
	}
}

func TestFlagErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "rejects an unknown flag",
			args: []string{"modify", "--frobnicate"},
			want: "Error: \"--frobnicate\" is not a flag of semvu modify.\n\n" +
				"Hint: To list the flags of semvu modify, run semvu modify --help.\n",
		},
		{
			name: "rejects an unknown short flag",
			args: []string{"modify", "-x"},
			want: "Error: \"-x\" is not a flag of semvu modify.\n\n" +
				"Hint: To list the flags of semvu modify, run semvu modify --help.\n",
		},
		{
			name: "points a flag given to the wrong command at the right one",
			args: []string{"1.2.3", "--patch"},
			want: "Error: \"--patch\" is not a flag of semvu.\n\nHint: Did you mean semvu modify --patch?\n",
		},
		{
			name: "finds the right command for an output flag",
			args: []string{"--with-v"},
			want: "Error: \"--with-v\" is not a flag of semvu.\n\nHint: Did you mean semvu latest --with-v?\n",
		},
		{
			name: "suggests a flag of the same command that the name is close to",
			args: []string{"latest", "--release"},
			want: "Error: \"--release\" is not a flag of semvu latest.\n\nHint: Did you mean --released?\n",
		},
		{
			name: "suggests a flag that the name is the start of",
			args: []string{"modify", "--tag", "api/"},
			want: "Error: \"--tag\" is not a flag of semvu modify.\n\nHint: Did you mean --tag-prefix?\n",
		},
		{
			name: "rejects a flag with no value",
			args: []string{"modify", "--directory"},
			want: "Error: --directory needs a value.\n\nHint: Give one, as in --directory path.\n",
		},
		{
			name: "rejects a value for a switch",
			args: []string{"modify", "1.2.3", "--major=maybe"},
			want: "Error: \"maybe\" is not a valid value for --major.\n\n" +
				"  --major is a switch: it is on when given, and takes no value.\n\n" +
				"Hint: Use --major on its own.\n",
		},
		{
			name: "rejects an unknown colour mode",
			args: []string{"--colour=sometimes"},
			want: "Error: \"sometimes\" is not a valid value for --colour.\n\n  --colour must be auto, always or never.\n",
		},
		{
			name: "rejects bad flag syntax",
			args: []string{"modify", "---patch"},
			want: "Error: \"---patch\" is not a valid flag.\n\n" +
				"Hint: To list the flags of semvu modify, run semvu modify --help.\n",
		},
		{
			name: "writes cobra's own errors as sentences",
			args: []string{"completion", "bash", "extra"},
			want: "Error: Unknown command \"extra\" for \"semvu completion bash\".\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertResult(t, run(t, tt.args...), result{code: ExitUsage, stderr: tt.want})
		})
	}
}

func TestErrorsInColour(t *testing.T) {
	t.Parallel()

	const (
		boldRed  = "\x1b[1m\x1b[31m"
		red      = "\x1b[31m"
		boldCyan = "\x1b[1m\x1b[36m"
		green    = "\x1b[32m"
		dim      = "\x1b[2m"
		reset    = "\x1b[0m"
	)

	tests := []struct {
		name string
		code int
		args []string
		want string
	}{
		{
			name: "colours the parts of a parse error",
			code: ExitInvalidVersion,
			args: []string{"modify", "1.2.3-alpha-1", "--colour=always"},
			want: boldRed + "Error:" + reset + reset + " " +
				boldRed + `"1.2.3-alpha-1"` + reset + reset + red + " is not a valid version." + reset + "\n\n" +
				"  1.2.3-alpha-" + red + "1" + reset + "\n" +
				"              " + red + "^" + reset + "\n" +
				"  The stage index must be exactly three digits, from 001 to 999.\n\n" +
				boldCyan + "Hint:" + reset + reset + " Did you mean " + green + `"1.2.3-alpha-001"` + reset + "?\n",
		},
		{
			name: "colours suggested commands and flags in a hint",
			code: ExitModification,
			args: []string{"modify", "1.2.3-rc-001", "--alpha", "--colour=always"},
			want: boldRed + "Error:" + reset + reset + " " +
				boldRed + `"1.2.3-rc-001"` + reset + reset + red + " can't go back from rc to alpha." + reset + "\n\n" +
				"  Stages only move forward: alpha, then beta, then rc, then the release.\n\n" +
				boldCyan + "Hint:" + reset + reset + " Use " + green + "--rc" + reset + " or " + green + "--release" + reset +
				", or add " + green + "--major" + reset + ", " + green + "--minor" + reset + " or " + green + "--patch" + reset +
				" to start a new version.\n",
		},
		{
			name: "colours flag errors",
			code: ExitUsage,
			args: []string{"--frobnicate", "--colour=always"},
			want: boldRed + "Error:" + reset + reset + " " +
				boldRed + `"--frobnicate"` + reset + reset + red + " is not a flag of semvu." + reset + "\n\n" +
				boldCyan + "Hint:" + reset + reset + " To list the flags of semvu, run " + green + "semvu --help" + reset + ".\n",
		},
		{
			name: "colours an error from an environment that forces colour",
			code: ExitUsage,
			args: []string{"nope"},
			want: boldRed + "Error:" + reset + reset + " " +
				boldRed + `"nope"` + reset + reset + red + " is not an semvu command." + reset + "\n\n" +
				boldCyan + "Hint:" + reset + reset + " To list semvu's commands, run " + green + "semvu --help" + reset + ".\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, env: map[string]string{forceColour: "1"}})
			assertResult(t, got, result{code: tt.code, stderr: tt.want})
		})
	}
}

func TestRenderStylesURLsDim(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer

	render(&b, &userError{headline: "Something failed.", hint: reportHint}, colour.New(true))

	want := "\x1b[2mhttps://github.com/tedslittlerobot/semvu/issues\x1b[0m."
	if !strings.Contains(b.String(), want) {
		t.Errorf("render() wrote %q, want it to contain %q", b.String(), want)
	}
}

func TestTokenise(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  string
		want []token
	}{
		{name: "keeps plain text whole", msg: "plain", want: []token{{tokenText, "plain"}}},
		{
			name: "finds code, quotes and URLs",
			msg:  "run `semvu --help`, see \"a \\\"b\\\"\" at https://example.com/x.",
			want: []token{
				{tokenText, "run "}, {tokenCode, "semvu --help"}, {tokenText, ", see "},
				{tokenQuoted, `"a \"b\""`}, {tokenText, " at "}, {tokenURL, "https://example.com/x"}, {tokenText, "."},
			},
		},
		{name: "leaves an unclosed backtick alone", msg: "a `b", want: []token{{tokenText, "a `b"}}},
		{name: "leaves an unclosed quote alone", msg: `a "b`, want: []token{{tokenText, `a "b`}}},
		{
			name: "ends a URL at a space",
			msg:  "see https://x.y now",
			want: []token{{tokenText, "see "}, {tokenURL, "https://x.y"}, {tokenText, " now"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tokenise(tt.msg); !slices.Equal(got, tt.want) {
				t.Errorf("tokenise(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}
