package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/tedslittlerobot/semvu/internal/git"
	"github.com/tedslittlerobot/semvu/internal/semver"
)

func TestModifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		args  []string
		stdin string
		tags  func(t *testing.T) tagReader
		want  string
	}{
		// Version modifiers.
		{name: "bumps the major version", args: []string{"1.2.3", "--major"}, want: "2.0.0\n"},
		{name: "bumps the minor version", args: []string{"1.2.3", "--minor"}, want: "1.3.0\n"},
		{name: "bumps the patch version", args: []string{"1.2.3", "--patch"}, want: "1.2.4\n"},
		{name: "removes the suffix when bumping", args: []string{"1.2.3-rc-004", "--patch"}, want: "1.2.4\n"},

		// Stage modifiers.
		{name: "releases a pre-release", args: []string{"1.2.3-rc-001", "--release"}, want: "1.2.3\n"},
		{name: "increments the index in the same stage", args: []string{"1.2.3-rc-001", "--rc"}, want: "1.2.3-rc-002\n"},
		{name: "moves to a later stage at 001", args: []string{"1.2.3-alpha-099", "--rc"}, want: "1.2.3-rc-001\n"},
		{name: "moves from alpha to beta", args: []string{"1.2.3-alpha-002", "--beta"}, want: "1.2.3-beta-001\n"},

		// Both.
		{name: "bumps the version and starts a stage", args: []string{"1.2.3", "--minor", "--alpha"}, want: "1.3.0-alpha-001\n"},
		{
			name: "starts any stage on a new version, even an earlier one",
			args: []string{"1.2.3-rc-002", "--major", "--alpha"},
			want: "2.0.0-alpha-001\n",
		},
		{name: "treats --release with a bump as no stage", args: []string{"1.2.3", "--minor", "--release"}, want: "1.3.0\n"},

		// With other options and inputs.
		{name: "adds a v prefix to the modified version", args: []string{"v1.2.3", "--patch", "--with-v"}, want: "v1.2.4\n"},
		{name: "ignores modifiers set to false", args: []string{"1.2.3", "--major=false", "--minor"}, want: "1.3.0\n"},
		{name: "modifies a version from standard input", args: []string{"-", "--beta"}, stdin: "1.2.3-alpha-001\n", want: "1.2.3-beta-001\n"},
		{
			name: "modifies the latest git tag",
			args: []string{"--patch", "--rc"},
			tags: func(t *testing.T) tagReader {
				t.Helper()

				return fixedTags(t, "", "v1.9.0", "v1.10.0")
			},
			want: "v1.10.1-rc-001\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inv := invocation{args: modifyArgs(tt.args...), stdin: strings.NewReader(tt.stdin)}
			if tt.tags != nil {
				inv.tags = tt.tags(t)
			}

			assertResult(t, invoke(t, inv), result{stdout: tt.want})
		})
	}
}

func TestModifierErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code int
		args []string
		want string
	}{
		{
			name: "rejects two version modifiers",
			code: ExitUsage,
			args: []string{"1.2.3", "--major", "--minor"},
			want: `Error: --major and --minor can't be used together.

  Give at most one version modifier: --major, --minor, --patch, --hotfix or --auto.
`,
		},
		{
			name: "rejects three version modifiers",
			code: ExitUsage,
			args: []string{"1.2.3", "--patch", "--minor", "--major"},
			want: `Error: --major, --minor and --patch can't be used together.

  Give at most one version modifier: --major, --minor, --patch, --hotfix or --auto.
`,
		},
		{
			name: "rejects two stage modifiers",
			code: ExitUsage,
			args: []string{"1.2.3-alpha-001", "--release", "--beta"},
			want: `Error: --beta and --release can't be used together.

  Give at most one stage modifier: --alpha, --beta, --rc or --release.
`,
		},
		{
			name: "rejects conflicting modifiers before reading git tags",
			code: ExitUsage,
			args: []string{"--alpha", "--rc"},
			want: `Error: --alpha and --rc can't be used together.

  Give at most one stage modifier: --alpha, --beta, --rc or --release.
`,
		},
		{
			name: "refuses to release a release",
			code: ExitModification,
			args: []string{"1.2.3", "--release"},
			want: `Error: "1.2.3" has already been released.

  --release removes the stage from a pre-release, such as 1.2.3-rc-001.

Hint: To start a new version, use --major, --minor or --patch.
`,
		},
		{
			name: "refuses to take a release back to a stage",
			code: ExitModification,
			args: []string{"v1.2.3", "--alpha"},
			want: `Error: "1.2.3" has been released, so it can't go back to alpha.

  Once a version is released, it can't go back into any stage.

Hint: Add --major, --minor or --patch to start a new version at alpha, as in --patch --alpha.
`,
		},
		{
			name: "refuses to go back to an earlier stage",
			code: ExitModification,
			args: []string{"1.2.3-rc-001", "--alpha"},
			want: `Error: "1.2.3-rc-001" can't go back from rc to alpha.

  Stages only move forward: alpha, then beta, then rc, then the release.

Hint: Use --rc or --release, or add --major, --minor or --patch to start a new version.
`,
		},
		{
			name: "refuses to go past the last index",
			code: ExitModification,
			args: []string{"1.2.3-alpha-999", "--alpha"},
			want: `Error: "1.2.3-alpha-999" is the last alpha allowed.

  A stage allows at most 999 iterations.

Hint: Use --beta, --rc or --release, or add --major, --minor or --patch to start a new version.
`,
		},
		{
			name: "refuses to bump past the largest number",
			code: ExitModification,
			args: []string{"18446744073709551615.0.0", "--major"},
			want: `Error: The major version of "18446744073709551615.0.0" can't be bumped.

  MAJOR, MINOR and PATCH must be no larger than 18446744073709551615.
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertResult(t, run(t, modifyArgs(tt.args...)...), result{code: tt.code, stderr: tt.want})
		})
	}
}

func TestModifyErrorAsksForAReportWhenUnexpected(t *testing.T) {
	t.Parallel()

	m := newModifiers(pflag.NewFlagSet("test", pflag.ContinueOnError))

	tests := []struct {
		name string
		err  error
	}{
		{name: "for an unknown error", err: errors.New("something odd")},
		{name: "for an unknown stage problem", err: &semver.StageError{Problem: semver.StageProblem(0)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := m.modifyError(tt.err)

			if got.headline != "semvu could not modify the version." || got.hint != reportHint {
				t.Errorf("modifyError() = %+v, want the unexpected-failure message", got)
			}
		})
	}
}

func TestJoin(t *testing.T) {
	t.Parallel()

	tests := []struct {
		items []string
		want  string
	}{
		{items: nil, want: ""},
		{items: []string{"a"}, want: "a"},
		{items: []string{"a", "b"}, want: "a or b"},
		{items: []string{"a", "b", "c"}, want: "a, b or c"},
	}

	for _, tt := range tests {
		if got := join(tt.items, "or"); got != tt.want {
			t.Errorf("join(%q) = %q, want %q", tt.items, got, tt.want)
		}
	}
}

func TestStageRegression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		stdin     string
		reachable []string
		others    []string
		want      string
	}{
		{name: "goes back from rc to alpha", args: []string{"1.2.3-rc-001", "--alpha", "--allow-stage-regression"}, want: "1.2.3-alpha-001\n"},
		{name: "goes back from rc to beta", args: []string{"v1.2.3-rc-003", "--beta", "--allow-stage-regression"}, want: "v1.2.3-beta-001\n"},
		{
			name:  "goes back from a version on standard input without reading tags",
			args:  []string{"-", "--alpha", "--allow-stage-regression"},
			stdin: "1.2.3-beta-002\n",
			want:  "1.2.3-alpha-001\n",
		},
		{
			name: "still moves forward",
			args: []string{"1.2.3-alpha-001", "--rc", "--allow-stage-regression"},
			want: "1.2.3-rc-001\n",
		},
		{
			name: "still increments the index within a stage",
			args: []string{"1.2.3-rc-001", "--rc", "--allow-stage-regression"},
			want: "1.2.3-rc-002\n",
		},
		{
			name:      "carries on after the stage's highest tag",
			args:      []string{"--alpha", "--allow-stage-regression"},
			reachable: []string{"v1.2.3-alpha-001", "v1.2.3-rc-001"},
			want:      "v1.2.3-alpha-002\n",
		},
		{
			name:      "starts the stage at 001 when it has no tags",
			args:      []string{"--beta", "--allow-stage-regression"},
			reachable: []string{"v1.2.3-alpha-001", "v1.2.3-rc-001"},
			want:      "v1.2.3-beta-001\n",
		},
		{
			name:      "counts tags on other branches, so as not to repeat one",
			args:      []string{"--alpha", "--allow-stage-regression"},
			reachable: []string{"v1.2.3-alpha-001", "v1.2.3-rc-001"},
			others:    []string{"v1.2.3-alpha-004"},
			want:      "v1.2.3-alpha-005\n",
		},
		{
			name:      "counts only tags with the prefix",
			args:      []string{"--alpha", "--allow-stage-regression", "--tag-prefix", "api/"},
			reachable: []string{"api/v1.2.3-alpha-001", "api/v1.2.3-rc-001", "v1.2.3-alpha-007"},
			want:      "v1.2.3-alpha-002\n",
		},
		{
			name:      "works with --for-major",
			args:      []string{"--alpha", "--allow-stage-regression", "--for-major", "1"},
			reachable: []string{"v1.2.3-alpha-001", "v1.2.3-rc-001", "v2.0.0"},
			want:      "v1.2.3-alpha-002\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inv := invocation{args: modifyArgs(tt.args...), stdin: strings.NewReader(tt.stdin)}
			if tt.reachable != nil {
				inv.tags = scopedTags(t, "", tt.reachable, tt.others)
			}

			assertResult(t, invoke(t, inv), result{stdout: tt.want})
		})
	}
}

func TestStageRegressionErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		code int
		want string
	}{
		{
			name: "refuses to take a release back to a stage",
			args: []string{"1.2.3", "--alpha", "--allow-stage-regression"},
			code: ExitModification,
			want: `Error: "1.2.3" has been released, so it can't go back to alpha.

  Once a version is released, it can't go back into any stage, even with --allow-stage-regression.

Hint: To start a new version at alpha, remove --allow-stage-regression and add --major, --minor or --patch, as in --patch --alpha.
`,
		},
		{
			name: "refuses to go past the last index",
			args: []string{"1.2.3-rc-999", "--rc", "--allow-stage-regression"},
			code: ExitModification,
			want: `Error: "1.2.3-rc-999" is the last rc allowed.

  A stage allows at most 999 iterations.

Hint: Use --release, or add --major, --minor or --patch to start a new version.
`,
		},
		{
			name: "rejects a version modifier",
			args: []string{"1.2.3-rc-001", "--patch", "--alpha", "--allow-stage-regression"},
			code: ExitUsage,
			want: `Error: --allow-stage-regression can't be used with --patch.

  --allow-stage-regression moves a pre-release back to an earlier stage of the same version,
  and --patch changes the version.

Hint: Remove --allow-stage-regression.
`,
		},
		{
			name: "rejects --auto",
			args: []string{"--auto", "--alpha", "--allow-stage-regression"},
			code: ExitUsage,
			want: `Error: --allow-stage-regression can't be used with --auto.

  --allow-stage-regression moves a pre-release back to an earlier stage of the same version,
  and --auto changes the version.

Hint: Remove --allow-stage-regression.
`,
		},
		{
			name: "needs a stage modifier",
			args: []string{"1.2.3-rc-001", "--allow-stage-regression"},
			code: ExitUsage,
			want: `Error: --allow-stage-regression only works with --alpha, --beta or --rc.

  It lets a pre-release go back to an earlier stage, as in 1.2.3-rc-001 --alpha.

Hint: Add --alpha, --beta or --rc, or remove --allow-stage-regression.
`,
		},
		{
			name: "rejects --release",
			args: []string{"--release", "--allow-stage-regression"},
			code: ExitUsage,
			want: `Error: --allow-stage-regression only works with --alpha, --beta or --rc.

  It lets a pre-release go back to an earlier stage, as in 1.2.3-rc-001 --alpha.

Hint: Add --alpha, --beta or --rc, or remove --allow-stage-regression.
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertResult(t, run(t, modifyArgs(tt.args...)...), result{code: tt.code, stderr: tt.want})
		})
	}
}

func TestStageRegressionTagErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		reachable []string
		others    []string
		want      string
	}{
		{
			name:      "refuses to go past the last index already tagged",
			reachable: []string{"v1.2.3-alpha-999", "v1.2.3-rc-001"},
			want: `Error: "1.2.3-alpha-999" is the last alpha allowed.

  A stage allows at most 999 iterations.

Hint: Use --beta, --rc or --release, or add --major, --minor or --patch to start a new version.
`,
		},
		{
			name:      "counts the last index on another branch",
			reachable: []string{"v1.2.3-rc-001"},
			others:    []string{"v1.2.3-alpha-999"},
			want: `Error: "1.2.3-alpha-999" is the last alpha allowed.

  A stage allows at most 999 iterations.

Hint: Use --beta, --rc or --release, or add --major, --minor or --patch to start a new version.
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{
				args: modifyArgs("--alpha", "--allow-stage-regression"),
				tags: scopedTags(t, "", tt.reachable, tt.others),
			})
			assertResult(t, got, result{code: ExitModification, stderr: tt.want})
		})
	}
}

func TestStageRegressionReportsTagFailure(t *testing.T) {
	t.Parallel()

	// The latest tag reads fine, but reading every tag fails.
	tags := func(_ context.Context, _ string, opts git.TagOptions) ([]string, error) {
		if opts.All {
			return nil, git.ErrNotInstalled
		}

		return []string{"v1.2.3-rc-001"}, nil
	}

	got := invoke(t, invocation{args: modifyArgs("--alpha", "--allow-stage-regression"), tags: tags})
	assertResult(t, got, result{
		code: ExitGit,
		stderr: "Error: semvu could not find git.\n\n" +
			"  No version was given, so semvu needs git to read the latest version from the tags.\n\n" +
			"Hint: Install git, or give a version, as in semvu modify 1.2.3.\n",
	})
}
