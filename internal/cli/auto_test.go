package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/tedslittlerobot/svu/internal/git"
	"github.com/tedslittlerobot/svu/internal/git/gittest"
)

// fixedMessages returns a messageReader that returns messages, and checks
// that it is asked for the commits since wantSince.
func fixedMessages(t *testing.T, wantSince string, messages ...string) messageReader {
	t.Helper()

	return func(_ context.Context, _, since string) ([]string, error) {
		if since != wantSince {
			t.Errorf("read commits since %q, want %q", since, wantSince)
		}

		return messages, nil
	}
}

func TestAuto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		tags       []string
		since      string
		messages   []string
		wantStdout string
		wantStderr string
	}{
		{
			name:       "bumps minor for a feat",
			args:       []string{"modify", "--auto"},
			tags:       []string{"v1.2.3"},
			since:      "v1.2.3",
			messages:   []string{"fix: a", "feat(api): b", "docs: c"},
			wantStdout: "1.3.0\n",
		},
		{
			name:       "bumps patch for a fix and starts a stage",
			args:       []string{"m", "--auto", "--rc", "--vp"},
			tags:       []string{"v1.2.3"},
			since:      "v1.2.3",
			messages:   []string{"fix: a", "chore: b"},
			wantStdout: "v1.2.4-rc-001\n",
		},
		{
			name:       "bumps major for a breaking change",
			args:       []string{"modify", "--auto"},
			tags:       []string{"v1.2.3"},
			since:      "v1.2.3",
			messages:   []string{"refactor!: drop a flag"},
			wantStdout: "2.0.0\n",
		},
		{
			name:       "explains a bump on stderr",
			args:       []string{"modify", "--auto", "--explain"},
			tags:       []string{"v1.2.3"},
			since:      "v1.2.3",
			messages:   []string{"fix: a", "feat!: b", "feat: c", "docs: d", "Tidy up"},
			wantStdout: "2.0.0\n",
			wantStderr: "5 commits since v1.2.3: 1 breaking change, 1 feat, 1 fix, 1 docs and 1 not following Conventional Commits.\n" +
				"That's a major bump, from 1.2.3 to 2.0.0.\n",
		},
		{
			name:       "explains the rule before 1.0.0",
			args:       []string{"modify", "--auto", "--explain"},
			tags:       []string{"0.4.2"},
			since:      "0.4.2",
			messages:   []string{"feat!: a"},
			wantStdout: "0.5.0\n",
			wantStderr: "1 commit since 0.4.2: 1 breaking change.\nBefore 1.0.0, a breaking change is a minor bump, from 0.4.2 to 0.5.0.\n",
		},
		{
			name:       "carries on a pre-release that covers the bump",
			args:       []string{"modify", "--auto", "--rc", "--explain"},
			tags:       []string{"v1.2.3", "v1.3.0-rc-001"},
			since:      "v1.3.0-rc-001",
			messages:   []string{"fix: a"},
			wantStdout: "1.3.0-rc-002\n",
			wantStderr: "1 commit since v1.3.0-rc-001: 1 fix.\n1.3.0-rc-001 already covers a patch bump, so it carries on, to 1.3.0-rc-002.\n",
		},
		{
			name:       "releases a pre-release that covers the bump",
			args:       []string{"modify", "--auto", "--explain"},
			tags:       []string{"v1.2.3", "v1.3.0-rc-001"},
			since:      "v1.3.0-rc-001",
			messages:   []string{"fix: a"},
			wantStdout: "1.3.0\n",
			wantStderr: "1 commit since v1.3.0-rc-001: 1 fix.\n1.3.0-rc-001 already covers a patch bump, so it's released, as 1.3.0.\n",
		},
		{
			name:       "bumps the release when a pre-release doesn't cover the bump",
			args:       []string{"modify", "--auto", "--rc", "--explain"},
			tags:       []string{"v1.2.3", "v1.3.0-rc-001"},
			since:      "v1.3.0-rc-001",
			messages:   []string{"feat!: a"},
			wantStdout: "2.0.0-rc-001\n",
			wantStderr: "1 commit since v1.3.0-rc-001: 1 breaking change.\n" +
				"1.3.0-rc-001 doesn't cover a major bump, so it's a major bump from the last release, 1.2.3, to 2.0.0-rc-001.\n",
		},
		{
			name:       "reads every commit when the version is --initial",
			args:       []string{"modify", "--auto", "--initial", "0.0.0", "--explain"},
			since:      "",
			messages:   []string{"feat: a", "fix: b"},
			wantStdout: "0.1.0\n",
			wantStderr: "2 commits in the repository: 1 feat and 1 fix.\nThat's a minor bump, from 0.0.0 to 0.1.0.\n",
		},
		{
			name:       "reads commits since a prefixed tag",
			args:       []string{"modify", "--auto", "--tag-prefix", "api/"},
			tags:       []string{"api/v0.4.0", "v9.0.0"},
			since:      "api/v0.4.0",
			messages:   []string{"perf: a"},
			wantStdout: "0.4.1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{
				args:     tt.args,
				tags:     fixedTags(t, "", tt.tags...),
				messages: fixedMessages(t, tt.since, tt.messages...),
			})
			assertResult(t, got, result{stdout: tt.wantStdout, stderr: tt.wantStderr})
		})
	}
}

func TestAutoErrors(t *testing.T) {
	t.Parallel()

	const chooseHint = "\nHint: To choose the bump yourself, use --patch, --minor or --major.\n"

	tests := []struct {
		name     string
		args     []string
		tags     []string
		messages messageReader
		code     int
		want     string
	}{
		{
			name: "reports no commits since the tag",
			args: []string{"modify", "--auto"}, tags: []string{"v1.2.3"},
			code: ExitNothingToRelease,
			want: "Error: There are no commits since v1.2.3.\n\n" +
				"  --auto works out the next version from the commits since the latest tag.\n" + chooseHint,
		},
		{
			name: "reports commits that need no new version",
			args: []string{"modify", "--auto"}, tags: []string{"v1.2.3"},
			messages: func(context.Context, string, string) ([]string, error) {
				return []string{"docs: a", "chore: b", "docs: c"}, nil
			},
			code: ExitNothingToRelease,
			want: "Error: None of the 3 commits since v1.2.3 needs a new version.\n\n" +
				"  Only breaking changes, feat, fix and perf commits change the version.\n" +
				"  These are 1 chore and 2 docs.\n" + chooseHint,
		},
		{
			name: "reports one commit that needs no new version",
			args: []string{"modify", "--auto"}, tags: []string{"v1.2.3"},
			messages: func(context.Context, string, string) ([]string, error) {
				return []string{"test: a"}, nil
			},
			code: ExitNothingToRelease,
			want: "Error: The commit since v1.2.3 doesn't need a new version.\n\n" +
				"  Only breaking changes, feat, fix and perf commits change the version.\n" +
				"  These are 1 test.\n" + chooseHint,
		},
		{
			name: "reports commits that don't follow Conventional Commits",
			args: []string{"modify", "--auto"}, tags: []string{"v1.2.3"},
			messages: func(context.Context, string, string) ([]string, error) {
				return []string{"Update README", "Fix it"}, nil
			},
			code: ExitNothingToRelease,
			want: "Error: None of the 2 commits since v1.2.3 follows Conventional Commits.\n\n" +
				"  --auto reads messages such as \"feat: add a flag\" or \"fix: handle empty input\".\n" +
				"  See https://www.conventionalcommits.org.\n" + chooseHint,
		},
		{
			name: "reports one commit that doesn't follow Conventional Commits",
			args: []string{"modify", "--auto"}, tags: []string{"v1.2.3"},
			messages: func(context.Context, string, string) ([]string, error) {
				return []string{"wip"}, nil
			},
			code: ExitNothingToRelease,
			want: "Error: The commit since v1.2.3 doesn't follow Conventional Commits.\n\n" +
				"  --auto reads messages such as \"feat: add a flag\" or \"fix: handle empty input\".\n" +
				"  See https://www.conventionalcommits.org.\n" + chooseHint,
		},
		{
			name: "rejects a version with --auto",
			args: []string{"modify", "1.2.3", "--auto"},
			code: ExitUsage,
			want: "Error: --auto can't be used when a version is given.\n\n" +
				"  --auto works out the bump from the commits since the latest tag, so it reads the tags itself.\n\n" +
				"Hint: Remove the version, or choose the bump with --major, --minor or --patch.\n",
		},
		{
			name: "rejects --explain without --auto",
			args: []string{"modify", "1.2.3", "--patch", "--explain"},
			code: ExitUsage,
			want: "Error: --explain only works with --auto.\n\n" +
				"  --explain says how --auto chose the bump from the commits.\n\n" +
				"Hint: Add --auto, or remove --explain.\n",
		},
		{
			name: "rejects --auto with another version modifier",
			args: []string{"modify", "--auto", "--minor"},
			code: ExitUsage,
			want: "Error: --minor and --auto can't be used together.\n\n" +
				"  Give at most one version modifier: --major, --minor, --patch, --hotfix or --auto.\n",
		},
		{
			name: "refuses to take a pending rc back to alpha",
			args: []string{"modify", "--auto", "--alpha"}, tags: []string{"v1.2.3", "v1.3.0-rc-001"},
			messages: func(context.Context, string, string) ([]string, error) {
				return []string{"fix: a"}, nil
			},
			code: ExitModification,
			want: "Error: \"1.3.0-rc-001\" can't go back from rc to alpha.\n\n" +
				"  Stages only move forward: alpha, then beta, then rc, then the release.\n\n" +
				"Hint: Use --rc or --release, or add --major, --minor or --patch to start a new version.\n",
		},
		{
			name: "reports a failure to read the commits",
			args: []string{"modify", "--auto"}, tags: []string{"v1.2.3"},
			messages: func(context.Context, string, string) ([]string, error) {
				return nil, &git.CommandError{Stderr: "fatal: bad revision", Err: errors.New("exit status 128")}
			},
			code: ExitGit,
			want: "Error: git could not read the commits in the current directory.\n\n  fatal: bad revision\n",
		},
		{
			name: "reports a missing repository when reading commits",
			args: []string{"modify", "--auto"}, tags: []string{"v1.2.3"},
			messages: func(context.Context, string, string) ([]string, error) {
				return nil, git.ErrNotInstalled
			},
			code: ExitGit,
			want: "Error: svu could not find git.\n\n" +
				"  No version was given, so svu needs git to read the latest version from the tags.\n\n" +
				"Hint: Install git, or give a version, as in svu modify 1.2.3.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inv := invocation{args: tt.args, messages: tt.messages}
			if tt.tags != nil {
				inv.tags = fixedTags(t, "", tt.tags...)
			}

			if inv.messages == nil && tt.tags != nil {
				inv.messages = fixedMessages(t, tt.tags[len(tt.tags)-1])
			}

			assertResult(t, invoke(t, inv), result{code: tt.code, stderr: tt.want})
		})
	}
}

func TestAutoInRealGitRepository(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepository(t, "v1.2.3")
	gittest.Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "fix: handle empty input")
	gittest.Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "feat(api): add a flag")
	gittest.Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "docs: explain")

	env := invocation{tags: git.Tags, messages: git.Messages}

	env.args = []string{"modify", "--auto", "--rc", "--vp", "--directory", dir}
	assertResult(t, invoke(t, env), result{stdout: "v1.3.0-rc-001\n"})

	gittest.Run(t, dir, "tag", "v1.3.0-rc-001")
	gittest.Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "fix: off by one")

	env.args = []string{"modify", "--auto", "--rc", "--directory", dir}
	assertResult(t, invoke(t, env), result{stdout: "1.3.0-rc-002\n"})

	env.args = []string{"modify", "--auto", "--directory", dir}
	assertResult(t, invoke(t, env), result{stdout: "1.3.0\n"})
}
