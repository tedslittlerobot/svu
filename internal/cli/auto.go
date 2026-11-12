package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"

	"github.com/tedslittlerobot/svu/internal/commits"
	"github.com/tedslittlerobot/svu/internal/git"
	"github.com/tedslittlerobot/svu/internal/semver"
)

// messageReader returns the commit messages since the tag since, or every
// commit's message if since is "". git.Messages is the real implementation.
type messageReader func(ctx context.Context, dir, since string) ([]string, error)

// autoFlag is the version modifier that works out the bump from commits.
const autoFlag = "auto"

// autoVersion works out the next version for svu modify --auto: it reads the
// latest tag that q matches, and bumps it as the Conventional Commits since
// then need. If explain isn't nil, it writes how it decided there.
func autoVersion(
	ctx context.Context, env environment, q tagQuery, stage semver.Stage, explain io.Writer,
) (semver.Version, error) {
	latest, tag, err := latestNamedTag(ctx, env.readTags, q)
	if err != nil {
		return semver.Version{}, err
	}

	messages, err := env.readMessages(ctx, q.dir, tag)
	if err != nil {
		return semver.Version{}, historyError(err, q)
	}

	summary := commits.Summarise(messages)

	since := "in the repository"
	if tag != "" {
		since = "since " + tag
	}

	if summary.Bump == commits.None {
		return semver.Version{}, nothingToReleaseError(summary, since)
	}

	// A pre-release is a bump over the last release, so that's needed too.
	var release semver.Version

	if !latest.IsRelease() {
		tags, err := env.readTags(ctx, q.dir, q.options())
		if err != nil {
			return semver.Version{}, tagsError(err, q)
		}

		release, _ = semver.LatestRelease(tags)
	}

	plan, err := semver.Auto(latest, release, bumpPart(summary.Bump), stage)
	if err != nil {
		return semver.Version{}, fmt.Errorf("working out the next version: %w", err)
	}

	if explain != nil {
		_, _ = fmt.Fprintf(explain, "%s %s: %s.\n%s\n",
			count(summary.Commits, "commit"), since, breakdown(summary), planReason(latest, release, plan))
	}

	return plan.Next, nil
}

// bumpPart turns a commit bump into the part of a version it bumps.
func bumpPart(b commits.Bump) semver.Part {
	switch b {
	case commits.Major:
		return semver.Major
	case commits.Minor:
		return semver.Minor
	default:
		return semver.Patch
	}
}

// planReason says, in one sentence, how the next version was chosen.
func planReason(latest, release semver.Version, plan semver.AutoPlan) string {
	switch {
	case plan.Pending && plan.Next.IsRelease():
		return fmt.Sprintf("%s already covers a %s bump, so it's released, as %s.", latest, plan.Part, plan.Next)
	case plan.Pending:
		return fmt.Sprintf("%s already covers a %s bump, so it carries on, to %s.", latest, plan.Part, plan.Next)
	case plan.BeforeStable:
		from := latest
		if !latest.IsRelease() {
			from = release
		}

		return fmt.Sprintf("Before 1.0.0, a breaking change is a minor bump, from %s to %s.", from, plan.Next)
	case latest.IsRelease():
		return fmt.Sprintf("That's a %s bump, from %s to %s.", plan.Part, latest, plan.Next)
	default:
		return fmt.Sprintf("%s doesn't cover a %s bump, so it's a %s bump from the last release, %s, to %s.",
			latest, plan.Part, plan.Part, release, plan.Next)
	}
}

// breakdown lists what a summary counted, biggest bumps first, as in
// "1 breaking change, 2 feat and 1 fix".
func breakdown(s commits.Summary) string {
	var parts []string

	if s.Breaking > 0 {
		parts = append(parts, count(s.Breaking, "breaking change"))
	}

	// The types that bump come first, then the rest alphabetically.
	types := make([]string, 0, len(s.Types))
	for t := range s.Types {
		types = append(types, t)
	}

	rank := map[string]int{"feat": 0, "fix": 1, "perf": 2}

	slices.SortFunc(types, func(a, b string) int {
		ra, okA := rank[a]
		rb, okB := rank[b]

		switch {
		case okA && okB:
			return ra - rb
		case okA:
			return -1
		case okB:
			return 1
		default:
			return cmp.Compare(a, b)
		}
	})

	for _, t := range types {
		parts = append(parts, strconv.Itoa(s.Types[t])+" "+t)
	}

	if s.Unconventional > 0 {
		parts = append(parts, fmt.Sprintf("%d not following Conventional Commits", s.Unconventional))
	}

	return join(parts, "and")
}

// count writes n and a noun, pluralised with "s", as in "1 commit".
func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}

	return strconv.Itoa(n) + " " + noun + "s"
}

// nothingToReleaseError reports commits that don't need a new version.
func nothingToReleaseError(s commits.Summary, since string) *userError {
	ue := &userError{
		code: ExitNothingToRelease,
		hint: "To choose the bump yourself, use `--patch`, `--minor` or `--major`.",
	}

	switch {
	case s.Commits == 0:
		ue.headline = fmt.Sprintf("There are no commits %s.", since)
		ue.detail = "--auto works out the next version from the commits since the latest tag."
	case s.Unconventional == s.Commits:
		ue.headline = fmt.Sprintf("None of the %s %s follows Conventional Commits.", count(s.Commits, "commit"), since)
		if s.Commits == 1 {
			ue.headline = fmt.Sprintf("The commit %s doesn't follow Conventional Commits.", since)
		}

		ue.detail = `--auto reads messages such as "feat: add a flag" or "fix: handle empty input".` +
			"\nSee https://www.conventionalcommits.org."
	default:
		ue.headline = fmt.Sprintf("None of the %s %s needs a new version.", count(s.Commits, "commit"), since)
		if s.Commits == 1 {
			ue.headline = fmt.Sprintf("The commit %s doesn't need a new version.", since)
		}

		ue.detail = "Only breaking changes, feat, fix and perf commits change the version.\n" +
			"These are " + breakdown(s) + "."
	}

	return ue
}

// historyError reports a failure to read the commit history.
func historyError(err error, q tagQuery) *userError {
	var cmdErr *git.CommandError
	if errors.As(err, &cmdErr) && cmdErr.Stderr != "" {
		return &userError{
			code:     ExitGit,
			headline: "git could not read the commits in " + q.place() + ".",
			detail:   cmdErr.Stderr,
			err:      err,
		}
	}

	return tagsError(err, q)
}
