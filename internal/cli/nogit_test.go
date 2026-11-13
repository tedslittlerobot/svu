package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/tedslittlerobot/semvu/internal/git"
)

// The tests here use invoke's default git functions, which fail the test if
// semvu runs git at all.

func TestNoGit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		args  []string
		stdin string
		want  string
	}{
		{name: "modifies a version argument", args: []string{"modify", "1.2.3", "--patch", "--no-git"}, want: "1.2.4\n"},
		{name: "modifies a version from standard input", args: []string{"m", "-", "--minor", "--no-git"}, stdin: "v1.2.3\n", want: "v1.3.0\n"},
		{name: "prints a version as it is", args: []string{"m", "v1.2.3", "--no-git", "--without-v"}, want: "1.2.3\n"},
		{
			name: "bumps a pre-release without checking for its release",
			args: []string{"modify", "1.2.3-rc-001", "--minor", "--rc", "--no-git"},
			want: "1.3.0-rc-001\n",
		},
		{
			name: "goes back a stage without reading tags",
			args: []string{"modify", "1.2.3-rc-001", "--alpha", "--allow-stage-regression", "--no-git"},
			want: "1.2.3-alpha-001\n",
		},
		{name: "hotfixes a release", args: []string{"modify", "1.2.3", "--hotfix", "--no-git"}, want: "1.2.4\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := invoke(t, invocation{args: tt.args, stdin: strings.NewReader(tt.stdin)})
			assertResult(t, got, result{stdout: tt.want})
		})
	}
}

func TestNoGitErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "needs a version",
			args: []string{"modify", "--patch", "--no-git"},
			want: `Error: semvu modify --no-git needs a version.

  --no-git stops semvu reading the latest version from git tags, so the version must be given.

Hint: Give a version, as in semvu modify 1.2.3 --no-git, or pass - to read it from standard input.
`,
		},
		{
			name: "rejects --auto",
			args: []string{"modify", "--auto", "--no-git"},
			want: `Error: --auto can't be used with --no-git.

  --no-git stops semvu using git at all, and --auto needs it.

Hint: Remove --auto, or remove --no-git.
`,
		},
		{
			name: "rejects --apply, even with a version",
			args: []string{"modify", "1.2.3", "--patch", "--apply", "--no-git"},
			want: `Error: --apply can't be used with --no-git.

  --no-git stops semvu using git at all, and --apply needs it.

Hint: Remove --apply, or remove --no-git.
`,
		},
		{
			name: "rejects a tag flag before asking for a version",
			args: []string{"modify", "--tag-prefix", "api/", "--patch", "--no-git"},
			want: `Error: --tag-prefix can't be used with --no-git.

  --no-git stops semvu using git at all, and --tag-prefix needs it.

Hint: Remove --tag-prefix, or remove --no-git.
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertResult(t, run(t, tt.args...), result{code: ExitUsage, stderr: tt.want})
		})
	}
}

func TestNoGitRejectsEveryGitFlag(t *testing.T) {
	t.Parallel()

	// A value each flag accepts, so that only --no-git is wrong.
	values := map[string][]string{
		"directory":      {"--directory", "."},
		"global":         {"--global"},
		"tag-prefix":     {"--tag-prefix", "api/"},
		"initial":        {"--initial", "0.0.0"},
		forMajorFlag:     {"--" + forMajorFlag, "1"},
		autoFlag:         {"--" + autoFlag},
		skipReleaseFlag:  {"--" + skipReleaseFlag, "--minor"},
		"apply":          {"--apply"},
		"apply-annotate": {"--apply-annotate"},
		pushFlag:         {"--" + pushFlag},
		remoteFlag:       {"--" + remoteFlag, "origin"},
	}

	for _, name := range gitFlags() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			flag, ok := values[name]
			if !ok {
				t.Fatalf("no test value for --%s", name)
			}

			got := run(t, append([]string{"modify", "1.2.3", "--no-git"}, flag...)...)

			want := "Error: --" + name + " can't be used with --no-git.\n"
			if got.code != ExitUsage || !strings.HasPrefix(got.stderr, want) || got.stdout != "" {
				t.Errorf("semvu modify --no-git --%s = %d, %q, %q; want %d, empty stdout, stderr starting %q",
					name, got.code, got.stdout, got.stderr, ExitUsage, want)
			}
		})
	}
}

func TestWithoutGitNeverRunsGit(t *testing.T) {
	t.Parallel()

	env := environment{}.withoutGit()

	if _, err := env.readTags(t.Context(), "", git.TagOptions{}); !errors.Is(err, errGitDisabled) {
		t.Errorf("readTags() error = %v, want %v", err, errGitDisabled)
	}

	if _, err := env.readMessages(t.Context(), "", ""); !errors.Is(err, errGitDisabled) {
		t.Errorf("readMessages() error = %v, want %v", err, errGitDisabled)
	}

	if err := env.createTag(t.Context(), "", "v1.2.3", nil); !errors.Is(err, errGitDisabled) {
		t.Errorf("createTag() error = %v, want %v", err, errGitDisabled)
	}

	if _, err := env.listRemotes(t.Context(), ""); !errors.Is(err, errGitDisabled) {
		t.Errorf("listRemotes() error = %v, want %v", err, errGitDisabled)
	}

	if err := env.pushTag(t.Context(), "", "origin", "v1.2.3"); !errors.Is(err, errGitDisabled) {
		t.Errorf("pushTag() error = %v, want %v", err, errGitDisabled)
	}
}
