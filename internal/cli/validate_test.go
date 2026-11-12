package cli

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		args  []string
		stdin string
		want  result
	}{
		{name: "accepts a release", args: []string{"validate", "1.2.3"}},
		{name: "accepts a prefixed pre-release", args: []string{"validate", "v1.2.3-rc-001"}},
		{name: "reads standard input with -", args: []string{"validate", "-"}, stdin: "1.2.3\n"},
		{
			name: "rejects an invalid version and explains why",
			args: []string{"validate", "1.2.3-alpha-1"},
			want: result{code: ExitInvalidVersion, stderr: "Error: \"1.2.3-alpha-1\" is not a valid version.\n\n" +
				"  1.2.3-alpha-1\n              ^\n" +
				"  The stage index must be exactly three digits, from 001 to 999.\n\n" +
				"Hint: Did you mean \"1.2.3-alpha-001\"?\n"},
		},
		{
			name:  "rejects an invalid version on standard input",
			args:  []string{"validate", "-"},
			stdin: "latest\n",
			want: result{code: ExitInvalidVersion, stderr: "Error: \"latest\" on standard input is not a valid version.\n\n" +
				"  latest\n  ^\n" +
				"  MAJOR, MINOR and PATCH must be whole numbers, as in 1.2.3.\n"},
		},
		{
			name: "doesn't suggest modify's flags for a word",
			args: []string{"validate", "patch"},
			want: result{code: ExitInvalidVersion, stderr: "Error: \"patch\" is not a valid version.\n\n" +
				"  patch\n  ^\n" +
				"  MAJOR, MINOR and PATCH must be whole numbers, as in 1.2.3.\n"},
		},
		{
			name:  "rejects empty standard input",
			args:  []string{"validate", "-"},
			stdin: "\n",
			want: result{code: ExitInvalidVersion, stderr: "Error: Standard input is empty.\n\n" +
				"  With \"-\" as the argument, svu reads one version from standard input.\n\n" +
				"Hint: Pipe a version into svu, as in echo 1.2.3 | svu modify -.\n"},
		},
		{
			name: "needs a version",
			args: []string{"validate"},
			want: result{code: ExitUsage, stderr: "Error: svu validate needs a version.\n\n" +
				"Hint: Give one, as in svu validate 1.2.3, or - to read it from standard input.\n"},
		},
		{
			name: "takes only one version",
			args: []string{"validate", "1.2.3", "1.2.4"},
			want: result{code: ExitUsage, stderr: "Error: svu validate was given 2 versions, but it takes one.\n\n" +
				"Hint: Check one version at a time, as in svu validate 1.2.3.\n"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, stdin: strings.NewReader(tt.stdin)})
			assertResult(t, got, tt.want)
		})
	}
}
