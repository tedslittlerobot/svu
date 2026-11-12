package cli

import (
	"slices"
	"strings"
	"testing"
)

// TestVFlags checks where the "v" of a printed version comes from: the
// version as it was read, unless --with-v or --without-v says otherwise.
func TestVFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		args  []string
		stdin string
		tags  []string
		want  string
	}{
		// From an argument.
		{name: "keeps no v from an argument", args: []string{"modify", "1.2.3", "--patch"}, want: "1.2.4\n"},
		{name: "keeps the v of an argument", args: []string{"modify", "v1.2.3", "--patch"}, want: "v1.2.4\n"},
		{name: "adds a v with --with-v", args: []string{"modify", "1.2.3", "--with-v"}, want: "v1.2.3\n"},
		{name: "drops the v with --without-v", args: []string{"modify", "v1.2.3", "--without-v"}, want: "1.2.3\n"},

		// From standard input.
		{name: "keeps the v from standard input", args: []string{"modify", "-", "--minor"}, stdin: "v2.0.0\n", want: "v2.1.0\n"},
		{name: "keeps no v from standard input", args: []string{"modify", "-", "--minor"}, stdin: "2.0.0\n", want: "2.1.0\n"},

		// From tags.
		{name: "keeps the v of the latest tag", args: []string{"modify", "--patch"}, tags: []string{"v1.2.3"}, want: "v1.2.4\n"},
		{name: "keeps no v from the latest tag", args: []string{"modify", "--patch"}, tags: []string{"1.2.3"}, want: "1.2.4\n"},
		{
			name: "follows the tag the version came from, not the others",
			args: []string{"latest"}, tags: []string{"v1.0.0", "1.1.0"}, want: "1.1.0\n",
		},
		{
			name: "looks past the tag prefix",
			args: []string{"latest", "--tag-prefix", "api/"}, tags: []string{"api/v0.4.0"}, want: "v0.4.0\n",
		},
		{
			name: "drops the v from a tag with --without-v",
			args: []string{"latest", "--without-v"}, tags: []string{"v1.2.3"}, want: "1.2.3\n",
		},
		{
			name: "adds a v to a tag with --with-v",
			args: []string{"latest", "--with-v"}, tags: []string{"1.2.3"}, want: "v1.2.3\n",
		},

		// From --initial.
		{name: "keeps the v of --initial", args: []string{"modify", "--initial", "v0.0.0", "--minor"}, want: "v0.1.0\n"},
		{name: "keeps no v from --initial", args: []string{"latest", "--initial", "0.1.0"}, want: "0.1.0\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inv := invocation{args: tt.args, stdin: strings.NewReader(tt.stdin)}
			if tt.tags != nil || slices.Contains(tt.args, "--initial") {
				inv.tags = fixedTags(t, "", tt.tags...)
			}

			assertResult(t, invoke(t, inv), result{stdout: tt.want})
		})
	}
}

func TestVFlagsAreMutuallyExclusive(t *testing.T) {
	t.Parallel()

	const want = "Error: --with-v and --without-v can't be used together.\n\n" +
		"  Without either, svu keeps the \"v\" of the version it read: from the argument,\n" +
		"  standard input or the tag.\n\n" +
		"Hint: Remove one of them.\n"

	for _, args := range [][]string{
		{"modify", "1.2.3", "--with-v", "--without-v"},
		{"latest", "--with-v", "--without-v"},
	} {
		assertResult(t, run(t, args...), result{code: ExitUsage, stderr: want})
	}
}
