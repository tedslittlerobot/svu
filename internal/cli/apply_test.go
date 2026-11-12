package cli

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tedslittlerobot/svu/internal/git"
	"github.com/tedslittlerobot/svu/internal/git/gittest"
)

// tagCall records one call to make a tag.
type tagCall struct {
	dir       string
	name      string
	annotated bool
}

// recordTags returns a tagCreator that records its calls in calls, and
// returns err.
func recordTags(calls *[]tagCall, err error) tagCreator {
	return func(_ context.Context, dir, name string, editor *git.Editor) error {
		*calls = append(*calls, tagCall{dir: dir, name: name, annotated: editor != nil})

		return err
	}
}

func TestApply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		tags     []string
		messages []string
		want     string
		wantTag  tagCall
	}{
		{
			name:    "tags the new version like the latest tag",
			args:    []string{"modify", "--patch", "--apply"},
			tags:    []string{"v1.2.3"},
			want:    "1.2.4\n",
			wantTag: tagCall{name: "v1.2.4"},
		},
		{
			name:    "tags without a v when the latest tag has none",
			args:    []string{"modify", "--minor", "--apply"},
			tags:    []string{"1.2.3"},
			want:    "1.3.0\n",
			wantTag: tagCall{name: "1.3.0"},
		},
		{
			name:    "adds a v when asked, whatever the latest tag",
			args:    []string{"modify", "--minor", "--apply", "--vp"},
			tags:    []string{"1.2.3"},
			want:    "v1.3.0\n",
			wantTag: tagCall{name: "v1.3.0"},
		},
		{
			name:    "follows --with-v-prefix in a repository with no tags",
			args:    []string{"modify", "--initial", "0.0.0", "--minor", "--apply"},
			want:    "0.1.0\n",
			wantTag: tagCall{name: "0.1.0"},
		},
		{
			name:    "tags a version given as an argument like the latest tag",
			args:    []string{"modify", "2.0.0", "--apply"},
			tags:    []string{"v1.2.3"},
			want:    "2.0.0\n",
			wantTag: tagCall{name: "v2.0.0"},
		},
		{
			name:    "makes an annotated tag with --apply-annotate",
			args:    []string{"modify", "--rc", "--apply-annotate"},
			tags:    []string{"v1.3.0-rc-001"},
			want:    "1.3.0-rc-002\n",
			wantTag: tagCall{name: "v1.3.0-rc-002", annotated: true},
		},
		{
			name:    "puts the prefix in the tag name",
			args:    []string{"modify", "--tag-prefix", "api/", "--minor", "--apply"},
			tags:    []string{"api/v0.4.0", "v9.0.0"},
			want:    "0.5.0\n",
			wantTag: tagCall{name: "api/v0.5.0"},
		},
		{
			name:    "tags in the --directory repository",
			args:    []string{"modify", "--directory", "elsewhere", "--patch", "--apply"},
			tags:    []string{"v1.0.0"},
			want:    "1.0.1\n",
			wantTag: tagCall{dir: "elsewhere", name: "v1.0.1"},
		},
		{
			name:     "tags the version --auto works out",
			args:     []string{"modify", "--auto", "--apply"},
			tags:     []string{"v1.2.3"},
			messages: []string{"feat: a"},
			want:     "1.3.0\n",
			wantTag:  tagCall{name: "v1.3.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var calls []tagCall

			got := invoke(t, invocation{
				args:      tt.args,
				tags:      scopedTags(t, tt.wantTag.dir, tt.tags, nil),
				messages:  func(context.Context, string, string) ([]string, error) { return tt.messages, nil },
				createTag: recordTags(&calls, nil),
			})

			assertResult(t, got, result{stdout: tt.want})

			if len(calls) != 1 || calls[0] != tt.wantTag {
				t.Errorf("tags made = %+v, want just %+v", calls, tt.wantTag)
			}
		})
	}
}

func TestApplyGivesTheEditorTheTerminal(t *testing.T) {
	t.Parallel()

	var editor *git.Editor

	got := invoke(t, invocation{
		args:  []string{"modify", "--patch", "--apply-annotate"},
		stdin: strings.NewReader("typed"),
		tags:  fixedTags(t, "", "v1.0.0"),
		createTag: func(_ context.Context, _, _ string, e *git.Editor) error {
			editor = e
			_, _ = e.Out.Write([]byte("editor screen\n"))

			return nil
		},
	})

	// What the editor writes goes to stderr, never into the version on stdout.
	assertResult(t, got, result{stdout: "1.0.1\n", stderr: "editor screen\n"})

	if editor == nil || editor.In == nil {
		t.Fatal("svu made the tag without giving git the terminal")
	}
}

func TestApplyErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		tags     tagReader
		tagError error
		code     int
		want     string
	}{
		{
			name: "refuses a version that's already tagged",
			args: []string{"modify", "1.2.4", "--apply"},
			tags: fixedTags(t, "", "v1.2.4"),
			code: ExitTagExists,
			want: "Error: 1.2.4 is already tagged, as \"v1.2.4\".\n\n" +
				"  --apply only makes new tags, so the version wasn't tagged or printed.\n\n" +
				"Hint: To see where it points, run git show v1.2.4.\n",
		},
		{
			name:     "reports a tag that appeared in the meantime",
			args:     []string{"modify", "--patch", "--apply"},
			tags:     fixedTags(t, "", "v1.2.3"),
			tagError: git.ErrTagExists,
			code:     ExitTagExists,
			want: "Error: The tag \"v1.2.4\" already exists.\n\n" +
				"  --apply only makes new tags, so the version wasn't tagged or printed.\n\n" +
				"Hint: To see where it points, run git show v1.2.4.\n",
		},
		{
			name: "reports a version given outside a repository",
			args: []string{"modify", "1.2.3", "--apply"},
			tags: failingTags(git.ErrNotRepository),
			code: ExitRepository,
			want: "Error: The current directory is not in a git repository.\n\n" +
				"  --apply tags the new version, so svu needs a git repository.\n\n" +
				"Hint: Run svu inside a git repository, or leave out --apply.\n",
		},
		{
			name: "reports git missing",
			args: []string{"modify", "1.2.3", "--apply"},
			tags: failingTags(git.ErrNotInstalled),
			code: ExitGit,
			want: "Error: svu could not find git.\n\n" +
				"  --apply needs git to tag the new version.\n\n" +
				"Hint: Install git, or leave out --apply.\n",
		},
		{
			name: "reports a failure to read the tags",
			args: []string{"modify", "1.2.3", "--apply"},
			tags: failingTags(&git.CommandError{Stderr: "fatal: broken", Err: errors.New("exit status 128")}),
			code: ExitGit,
			want: "Error: git could not list the tags in the current directory.\n\n  fatal: broken\n",
		},
		{
			name:     "passes on git's explanation for a lightweight tag",
			args:     []string{"modify", "--patch", "--apply"},
			tags:     fixedTags(t, "", "v1.2.3"),
			tagError: &git.CommandError{Stderr: "fatal: cannot lock ref", Err: errors.New("exit status 128")},
			code:     ExitGit,
			want:     "Error: git could not make the tag \"v1.2.4\".\n\n  fatal: cannot lock ref\n",
		},
		{
			name:     "points at the editor settings for an annotated tag",
			args:     []string{"modify", "--patch", "--apply-annotate"},
			tags:     fixedTags(t, "", "v1.2.3"),
			tagError: &git.CommandError{Err: errors.New("exit status 1")},
			code:     ExitGit,
			want: "Error: git could not make the tag \"v1.2.4\".\n\n" +
				"Hint: If your editor didn't open, or the message was empty, check git's core.editor setting, " +
				"or the GIT_EDITOR, VISUAL and EDITOR variables.\n",
		},
		{
			name:     "reports the repository disappearing",
			args:     []string{"modify", "--patch", "--apply"},
			tags:     fixedTags(t, "", "v1.2.3"),
			tagError: git.ErrNotRepository,
			code:     ExitRepository,
			want: "Error: The current directory is not in a git repository.\n\n" +
				"  --apply tags the new version, so svu needs a git repository.\n\n" +
				"Hint: Run svu inside a git repository, or leave out --apply.\n",
		},
		{
			name:     "reports an unexpected failure",
			args:     []string{"modify", "--patch", "--apply"},
			tags:     fixedTags(t, "", "v1.2.3"),
			tagError: errors.New("boom"),
			code:     ExitInternal,
			want:     "Error: svu could not read the git tags in the current directory.\n\n" + reportLine,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var calls []tagCall

			got := invoke(t, invocation{args: tt.args, tags: tt.tags, createTag: recordTags(&calls, tt.tagError)})
			assertResult(t, got, result{code: tt.code, stderr: tt.want})
		})
	}
}

func TestApplyInRealGitRepository(t *testing.T) {
	t.Parallel()

	dir := gittest.NewRepository(t, "v1.2.3")
	gittest.Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "fix: a")

	env := invocation{tags: git.Tags, createTag: git.CreateTag}

	env.args = []string{"modify", "--patch", "--apply", "--directory", dir}
	assertResult(t, invoke(t, env), result{stdout: "1.2.4\n"})

	// The new tag is the latest, on HEAD, so a re-run sees it's tagged.
	env.args = []string{"latest", "--at-head", "--vp", "--directory", dir}
	assertResult(t, invoke(t, env), result{stdout: "v1.2.4\n"})
}
