package cli

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/tedslittlerobot/semvu/internal/git"
	"github.com/tedslittlerobot/semvu/internal/git/gittest"
)

// pushCall records one call to push a tag.
type pushCall struct {
	dir    string
	remote string
	name   string
}

// recordPushes returns a tagPusher that records its calls in calls, and
// returns err.
func recordPushes(calls *[]pushCall, err error) tagPusher {
	return func(_ context.Context, dir, remote, name string) error {
		*calls = append(*calls, pushCall{dir: dir, remote: remote, name: name})

		return err
	}
}

// fixedRemotes returns a remoteLister for a repository with remotes.
func fixedRemotes(remotes ...string) remoteLister {
	return func(context.Context, string) ([]string, error) {
		return remotes, nil
	}
}

func TestPush(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		remotes  []string
		want     string
		wantTag  tagCall
		wantPush pushCall
	}{
		{
			name:     "pushes to the only remote",
			args:     []string{"modify", "--patch", "--apply", "--push"},
			remotes:  []string{"upstream"},
			want:     "v1.2.4\n",
			wantTag:  tagCall{name: "v1.2.4"},
			wantPush: pushCall{remote: "upstream", name: "v1.2.4"},
		},
		{
			name:     "pushes to origin when there are several remotes",
			args:     []string{"m", "--minor", "--apply", "--push"},
			remotes:  []string{"fork", "origin", "upstream"},
			want:     "v1.3.0\n",
			wantTag:  tagCall{name: "v1.3.0"},
			wantPush: pushCall{remote: "origin", name: "v1.3.0"},
		},
		{
			name:     "pushes to the remote named by --remote",
			args:     []string{"modify", "--patch", "--apply", "--push", "--remote", "upstream"},
			remotes:  []string{"origin", "upstream"},
			want:     "v1.2.4\n",
			wantTag:  tagCall{name: "v1.2.4"},
			wantPush: pushCall{remote: "upstream", name: "v1.2.4"},
		},
		{
			name:     "pushes an annotated tag",
			args:     []string{"modify", "--patch", "--apply-annotate", "--push"},
			remotes:  []string{"origin"},
			want:     "v1.2.4\n",
			wantTag:  tagCall{name: "v1.2.4", annotated: true},
			wantPush: pushCall{remote: "origin", name: "v1.2.4"},
		},
		{
			name:     "pushes the tag with its prefix, from --directory",
			args:     []string{"modify", "--directory", "elsewhere", "--tag-prefix", "api/", "--patch", "--apply", "--push"},
			remotes:  []string{"origin"},
			want:     "v1.2.4\n",
			wantTag:  tagCall{dir: "elsewhere", name: "api/v1.2.4"},
			wantPush: pushCall{dir: "elsewhere", remote: "origin", name: "api/v1.2.4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var (
				tags   []tagCall
				pushes []pushCall
			)

			got := invoke(t, invocation{
				args:      tt.args,
				tags:      scopedTags(t, tt.wantTag.dir, []string{"v1.2.3", "api/v1.2.3"}, nil),
				createTag: recordTags(&tags, nil),
				remotes:   fixedRemotes(tt.remotes...),
				push:      recordPushes(&pushes, nil),
			})

			assertResult(t, got, result{stdout: tt.want})

			if len(tags) != 1 || tags[0] != tt.wantTag {
				t.Errorf("tags made = %+v, want just %+v", tags, tt.wantTag)
			}

			if len(pushes) != 1 || pushes[0] != tt.wantPush {
				t.Errorf("pushes = %+v, want just %+v", pushes, tt.wantPush)
			}
		})
	}
}

func TestPushFlagErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "needs --apply",
			args: []string{"modify", "1.2.3", "--patch", "--push"},
			want: "Error: --push only works with --apply or --apply-annotate.\n\n" +
				"  --push pushes the tag that --apply makes.\n\n" +
				"Hint: Add --apply, or remove --push.\n",
		},
		{
			name: "needs --push for --remote",
			args: []string{"modify", "1.2.3", "--apply", "--remote", "origin"},
			want: "Error: --remote only works with --push.\n\n" +
				"  --remote chooses the remote that --push sends the new tag to.\n\n" +
				"Hint: Add --push, or remove --remote.\n",
		},
		{
			name: "rejects an empty --remote",
			args: []string{"modify", "1.2.3", "--apply", "--push", "--remote="},
			want: "Error: --remote can't be empty.\n\n" +
				"  --remote names the remote that --push sends the new tag to.\n\n" +
				"Hint: Give a remote's name, as in --remote origin.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assertResult(t, run(t, tt.args...), result{code: ExitUsage, stderr: tt.want})
		})
	}
}

func TestPushRemoteErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		remotes remoteLister
		code    int
		want    string
	}{
		{
			name:    "refuses a repository with no remotes",
			args:    []string{"--push"},
			remotes: fixedRemotes(),
			code:    ExitNoRemote,
			want: "Error: The git repository has no remotes.\n\n" +
				"  --push sends the new tag to a remote, so semvu made no tag.\n\n" +
				"Hint: Add one, as in git remote add origin <url>, or leave out --push.\n",
		},
		{
			name:    "refuses several remotes without origin",
			args:    []string{"--push"},
			remotes: fixedRemotes("upstream", "fork"),
			code:    ExitNoRemote,
			want: "Error: The git repository has 2 remotes, and none is called \"origin\".\n\n" +
				"  Its remotes are \"fork\" and \"upstream\". With several, --push uses \"origin\",\n" +
				"  so semvu made no tag.\n\n" +
				"Hint: Choose one, as in --remote fork.\n",
		},
		{
			name:    "refuses a remote that doesn't exist, suggesting origin",
			args:    []string{"--push", "--remote", "mirror"},
			remotes: fixedRemotes("upstream", "origin"),
			code:    ExitNoRemote,
			want: "Error: The git repository has no remote called \"mirror\".\n\n" +
				"  Its remotes are \"origin\" and \"upstream\", so semvu made no tag.\n\n" +
				"Hint: Choose one of its remotes, as in --remote origin.\n",
		},
		{
			name:    "refuses a remote that doesn't exist, suggesting the only one",
			args:    []string{"--push", "--remote", "origin", "--directory", "elsewhere"},
			remotes: fixedRemotes("upstream"),
			code:    ExitNoRemote,
			want: "Error: The git repository at \"elsewhere\" has no remote called \"origin\".\n\n" +
				"  Its only remote is \"upstream\", so semvu made no tag.\n\n" +
				"Hint: Choose one of its remotes, as in --remote upstream.\n",
		},
		{
			name:    "refuses a remote asked for when there are none",
			args:    []string{"--push", "--remote", "upstream"},
			remotes: fixedRemotes(),
			code:    ExitNoRemote,
			want: "Error: The git repository has no remotes.\n\n" +
				"  --push sends the new tag to a remote, so semvu made no tag.\n\n" +
				"Hint: Add one, as in git remote add upstream <url>, or leave out --push.\n",
		},
		{
			name: "reports a failure to list the remotes",
			args: []string{"--push"},
			remotes: func(context.Context, string) ([]string, error) {
				return nil, &git.CommandError{Stderr: "fatal: bad config", Err: errors.New("exit status 128")}
			},
			code: ExitGit,
			want: "Error: git could not list the remotes in the current directory.\n\n  fatal: bad config\n",
		},
		{
			name: "reports git missing when listing the remotes",
			args: []string{"--push"},
			remotes: func(context.Context, string) ([]string, error) {
				return nil, git.ErrNotInstalled
			},
			code: ExitGit,
			want: "Error: semvu could not find git.\n\n" +
				"  --apply needs git to tag the new version.\n\n" +
				"Hint: Install git, or leave out --apply.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// No createTag: the remote is checked before anything is tagged.
			dir := ""
			if slices.Contains(tt.args, "--directory") {
				dir = "elsewhere"
			}

			got := invoke(t, invocation{
				args:    append([]string{"modify", "--patch", "--apply"}, tt.args...),
				tags:    scopedTags(t, dir, []string{"v1.2.3"}, nil),
				remotes: tt.remotes,
			})
			assertResult(t, got, result{code: tt.code, stderr: tt.want})
		})
	}
}

func TestPushFailure(t *testing.T) {
	t.Parallel()

	rejected := &git.CommandError{Stderr: "! [rejected] v1.2.4 -> v1.2.4 (already exists)", Err: errors.New("exit status 1")}

	tests := []struct {
		name    string
		args    []string
		pushErr error
		want    string
	}{
		{
			name:    "says the tag was made, and how to push it",
			args:    []string{"modify", "--patch", "--apply", "--push"},
			pushErr: rejected,
			want: "Error: semvu made the tag \"v1.2.4\", but git could not push it to \"origin\".\n\n" +
				"  ! [rejected] v1.2.4 -> v1.2.4 (already exists)\n\n" +
				"Hint: To push it, run git push origin v1.2.4.\n",
		},
		{
			name:    "points the push command at --directory",
			args:    []string{"modify", "--patch", "--apply", "--push", "--directory", "my repo"},
			pushErr: errors.New("boom"),
			want: "Error: semvu made the tag \"v1.2.4\", but git could not push it to \"origin\".\n\n" +
				"Hint: To push it, run git -C \"my repo\" push origin v1.2.4.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := ""
			if slices.Contains(tt.args, "--directory") {
				dir = "my repo"
			}

			var (
				tags   []tagCall
				pushes []pushCall
			)

			got := invoke(t, invocation{
				args:      tt.args,
				tags:      scopedTags(t, dir, []string{"v1.2.3"}, nil),
				createTag: recordTags(&tags, nil),
				remotes:   fixedRemotes("origin"),
				push:      recordPushes(&pushes, tt.pushErr),
			})

			assertResult(t, got, result{code: ExitGit, stderr: tt.want})

			if len(tags) != 1 {
				t.Errorf("tags made = %+v, want one", tags)
			}
		})
	}
}

func TestPushInRealGitRepository(t *testing.T) {
	t.Parallel()

	remote := t.TempDir()
	gittest.Run(t, remote, "init", "--quiet", "--bare")

	dir := gittest.NewRepository(t, "v1.2.3")
	gittest.Run(t, dir, "commit", "--quiet", "--allow-empty", "--message", "fix: a")
	gittest.Run(t, dir, "remote", "add", "upstream", remote)

	env := invocation{tags: git.Tags, createTag: git.CreateTag, remotes: git.Remotes, push: git.PushTag}

	env.args = []string{"modify", "--patch", "--apply", "--push", "--directory", dir}
	assertResult(t, invoke(t, env), result{stdout: "v1.2.4\n"})

	pushed, err := git.Tags(t.Context(), remote, git.TagOptions{All: true})
	if err != nil || !slices.Equal(pushed, []string{"v1.2.4"}) {
		t.Errorf("tags on the remote = %q, %v, want just v1.2.4", pushed, err)
	}
}
