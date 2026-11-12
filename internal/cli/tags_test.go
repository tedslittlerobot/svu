package cli

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/tedslittlerobot/svu/internal/git"
	"github.com/tedslittlerobot/svu/internal/git/gittest"
)

func TestTagFlags(t *testing.T) {
	t.Parallel()

	// The checked-out branch has v1.2.3 and an rc in progress. Another branch
	// has v2.0.0, and a monorepo package has its own prefixed tags.
	reachable := []string{"v1.2.3", "v1.3.0-rc-001", "api/v0.4.0", "api/v0.5.0-beta-002"}
	others := []string{"v2.0.0", "api/v0.6.0"}

	tests := []struct {
		name string
		args []string
		want string
	}{
		// Scope.
		{name: "latest reads only tags on the checked-out history", args: []string{"latest"}, want: "1.3.0-rc-001\n"},
		{name: "latest reads every tag with --global", args: []string{"latest", "--global"}, want: "2.0.0\n"},
		{name: "modify reads only tags on the checked-out history", args: []string{"modify", "--patch"}, want: "1.3.1\n"},
		{name: "modify reads every tag with --global", args: []string{"m", "--global", "--patch"}, want: "2.0.1\n"},

		// Prefix.
		{name: "latest reads prefixed tags", args: []string{"latest", "--tag-prefix", "api/"}, want: "0.5.0-beta-002\n"},
		{
			name: "latest combines a prefix with --released and --global",
			args: []string{"l", "--tag-prefix", "api/", "--released", "--global"},
			want: "0.6.0\n",
		},
		{name: "modify bumps a prefixed tag", args: []string{"m", "--tag-prefix=api/", "--beta"}, want: "0.5.0-beta-003\n"},

		// Hotfix.
		{name: "hotfix patches the latest release, not the rc", args: []string{"modify", "--hotfix"}, want: "1.2.4\n"},
		{name: "hotfix can start a stage", args: []string{"m", "--hotfix", "--rc", "--vp"}, want: "v1.2.4-rc-001\n"},
		{name: "hotfix reads every release with --global", args: []string{"m", "--hotfix", "--global"}, want: "2.0.1\n"},
		{name: "hotfix works with a prefix", args: []string{"m", "--hotfix", "--tag-prefix", "api/"}, want: "0.4.1\n"},

		// Initial is ignored when a tag matches.
		{name: "initial is unused when a tag matches", args: []string{"latest", "--initial", "9.9.9"}, want: "1.3.0-rc-001\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, tags: scopedTags(t, "", reachable, others)})
			assertResult(t, got, result{stdout: tt.want})
		})
	}
}

func TestHotfixGivenAVersion(t *testing.T) {
	t.Parallel()

	assertResult(t, run(t, "modify", "v1.2.3", "--hotfix"), result{stdout: "1.2.4\n"})

	assertResult(t, run(t, "modify", "1.3.0-rc-001", "--hotfix"), result{
		code: ExitModification,
		stderr: "Error: \"1.3.0-rc-001\" is a pre-release, so it can't be hotfixed.\n\n" +
			"  --hotfix patches a released version.\n\n" +
			"Hint: Give the release to fix, as in svu modify 1.2.3 --hotfix, or use --patch.\n",
	})
}

func TestInitial(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		tags []string
		want string
	}{
		{name: "latest prints the initial version with no tags", args: []string{"latest", "--initial", "0.1.0"}, want: "0.1.0\n"},
		{name: "modify bumps the initial version", args: []string{"modify", "--initial", "0.0.0", "--minor"}, want: "0.1.0\n"},
		{
			name: "modify starts a stage from the initial version",
			args: []string{"m", "--initial=v0.0.0", "--patch", "--alpha", "--vp"},
			want: "v0.0.1-alpha-001\n",
		},
		{
			name: "latest uses the initial version when no tag is valid",
			args: []string{"latest", "--initial", "1.0.0"},
			tags: []string{"nightly"},
			want: "1.0.0\n",
		},
		{
			name: "latest uses the initial version when there is no release",
			args: []string{"latest", "--released", "--initial", "1.0.0"},
			tags: []string{"v2.0.0-rc-001"},
			want: "1.0.0\n",
		},
		{
			name: "hotfix uses the initial version when there is no release",
			args: []string{"m", "--hotfix", "--initial", "1.0.0"},
			tags: []string{"v2.0.0-rc-001"},
			want: "1.0.1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, tags: fixedTags(t, "", tt.tags...)})
			assertResult(t, got, result{stdout: tt.want})
		})
	}
}

func TestTagFlagErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		reachable []string
		others    []string
		readsTags bool
		code      int
		want      string
	}{
		{
			name: "rejects --global with a version",
			args: []string{"modify", "1.2.3", "--global"},
			code: ExitUsage,
			want: "Error: --global can't be used when a version is given.\n\n" +
				"  svu modify only reads git tags when no version is given.\n\n" +
				"Hint: Remove --global, or remove the version.\n",
		},
		{
			name: "rejects --tag-prefix with standard input",
			args: []string{"modify", "-", "--tag-prefix", "api/"},
			code: ExitUsage,
			want: "Error: --tag-prefix can't be used when a version is given.\n\n" +
				"  svu modify only reads git tags when no version is given.\n\n" +
				"Hint: Remove --tag-prefix, or remove the version.\n",
		},
		{
			name: "rejects --initial with a version",
			args: []string{"modify", "1.2.3", "--initial", "0.0.0"},
			code: ExitUsage,
			want: "Error: --initial can't be used when a version is given.\n\n" +
				"  svu modify only reads git tags when no version is given.\n\n" +
				"Hint: Remove --initial, or remove the version.\n",
		},
		{
			name: "rejects an invalid initial version",
			args: []string{"latest", "--initial", "1.2"},
			code: ExitInvalidVersion,
			want: "Error: \"1.2\" for --initial is not a valid version.\n\n" +
				"  1.2\n     ^\n" +
				"  A version must have three numbers, MAJOR.MINOR.PATCH, as in 1.2.3.\n",
		},
		{
			name: "rejects a pre-release initial version with --released",
			args: []string{"latest", "--released", "--initial", "0.1.0-rc-001"},
			code: ExitUsage,
			want: "Error: \"0.1.0-rc-001\" can't be the --initial version with --released.\n\n" +
				"  --released only gives release versions, so --initial must be one too.\n\n" +
				"Hint: Give a release, as in --initial 0.1.0.\n",
		},
		{
			name: "rejects a pre-release initial version with --hotfix",
			args: []string{"modify", "--hotfix", "--initial", "0.1.0-rc-001"},
			code: ExitUsage,
			want: "Error: \"0.1.0-rc-001\" can't be the --initial version with --hotfix.\n\n" +
				"  --hotfix only gives release versions, so --initial must be one too.\n\n" +
				"Hint: Give a release, as in --initial 0.1.0.\n",
		},
		{
			name:      "points to --global when other branches have versions",
			args:      []string{"latest"},
			reachable: []string{"nightly"},
			others:    []string{"v2.0.0"},
			readsTags: true,
			code:      ExitNoMatchingTag,
			want: "Error: The git repository has no tags that are valid versions.\n\n" +
				"  None of its tags on the checked-out commit's history is a valid version, such as v1.2.3 or v1.2.3-rc-001.\n\n" +
				"Hint: Tags that aren't on the checked-out commit's history have versions. To include them, use --global.\n",
		},
		{
			name:      "points to --global for a release on another branch",
			args:      []string{"latest", "--released"},
			reachable: []string{"v1.0.0-rc-001"},
			others:    []string{"v0.9.0"},
			readsTags: true,
			code:      ExitNoMatchingTag,
			want: "Error: The git repository has no released versions.\n\n" +
				"  Its latest version, \"1.0.0-rc-001\", is a pre-release.\n\n" +
				"Hint: Tags that aren't on the checked-out commit's history have versions. To include them, use --global.\n",
		},
		{
			name:      "names the prefix when no prefixed tag matches",
			args:      []string{"latest", "--tag-prefix", "api/", "--global"},
			reachable: []string{"v1.0.0"},
			readsTags: true,
			code:      ExitNoMatchingTag,
			want: "Error: The git repository has no tags that are valid versions.\n\n" +
				"  It has no tags starting with \"api/\".\n\n" +
				"Hint: Tag a release, as in git tag api/v0.1.0, or use --initial 0.0.0 to start from scratch.\n",
		},
		{
			name:      "explains that a hotfix needs a release",
			args:      []string{"modify", "--hotfix", "--global"},
			reachable: []string{"v1.0.0-rc-001"},
			readsTags: true,
			code:      ExitNoMatchingTag,
			want: "Error: The git repository has no released versions.\n\n" +
				"  Its latest version, \"1.0.0-rc-001\", is a pre-release.\n\n" +
				"Hint: A hotfix patches the latest release, and there isn't one yet. Use --patch instead.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inv := invocation{args: tt.args, stdin: strings.NewReader("1.2.3")}
			if tt.readsTags {
				inv.tags = scopedTags(t, "", tt.reachable, tt.others)
			}

			assertResult(t, invoke(t, inv), result{code: tt.code, stderr: tt.want})
		})
	}
}

func TestTagFlagsInRealGitRepository(t *testing.T) {
	t.Parallel()

	// main: v1.0.0, then v1.1.0-rc-001. support (from v1.0.0): v1.0.1.
	dir := gittest.NewRepository(t, "v1.0.0")
	gittest.Run(t, dir, "branch", "-M", "main")
	gittest.Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "Feature")
	gittest.Run(t, dir, "tag", "v1.1.0-rc-001")
	gittest.Run(t, dir, "checkout", "--quiet", "-b", "support", "v1.0.0")
	gittest.Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "Fix")
	gittest.Run(t, dir, "tag", "v1.0.1")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "latest sees only the support branch", args: []string{"latest"}, want: "1.0.1\n"},
		{name: "latest sees everything with --global", args: []string{"latest", "--global"}, want: "1.1.0-rc-001\n"},
		{name: "a patch on the support branch follows its own tags", args: []string{"modify", "--patch"}, want: "1.0.2\n"},
		{name: "a global hotfix patches the latest release", args: []string{"modify", "--hotfix", "--global"}, want: "1.0.2\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			args := slices.Concat(tt.args, []string{"--directory", dir})
			assertResult(t, invoke(t, invocation{args: args, tags: git.Tags}), result{stdout: tt.want})
		})
	}
}

// TestExitCodes pins one example of each exit code, because scripts depend
// on them.
func TestExitCodes(t *testing.T) {
	t.Parallel()

	noCommits := func(context.Context, string, string) ([]string, error) { return nil, nil }

	tests := []struct {
		name     string
		args     []string
		tags     tagReader
		messages messageReader
		want     int
	}{
		{name: "success", args: []string{"modify", "1.2.3"}, want: ExitOK},
		{name: "internal failure", args: []string{"latest"}, tags: failingTags(errors.New("boom")), want: ExitInternal},
		{name: "usage", args: []string{"modify", "--frobnicate"}, want: ExitUsage},
		{name: "invalid version", args: []string{"modify", "1.2"}, want: ExitInvalidVersion},
		{name: "modification not allowed", args: []string{"modify", "1.2.3", "--release"}, want: ExitModification},
		{name: "no repository", args: []string{"latest"}, tags: failingTags(git.ErrNotRepository), want: ExitRepository},
		{name: "no matching tag", args: []string{"latest"}, tags: failingTags(nil), want: ExitNoMatchingTag},
		{name: "git failed", args: []string{"latest"}, tags: failingTags(git.ErrNotInstalled), want: ExitGit},
		{
			name: "nothing to release", args: []string{"modify", "--auto"},
			tags: scopedTags(t, "", []string{"v1.0.0"}, nil), messages: noCommits, want: ExitNothingToRelease,
		},
	}

	seen := map[int]string{}

	for _, tt := range tests {
		if other, ok := seen[tt.want]; ok {
			t.Fatalf("%q and %q both expect exit code %d, want one example of each", tt.name, other, tt.want)
		}

		seen[tt.want] = tt.name

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := invoke(t, invocation{args: tt.args, tags: tt.tags, messages: tt.messages}).code; got != tt.want {
				t.Errorf("exit code = %d, want %d", got, tt.want)
			}
		})
	}
}
