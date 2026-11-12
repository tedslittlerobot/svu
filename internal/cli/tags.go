package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tedslittlerobot/svu/internal/git"
	"github.com/tedslittlerobot/svu/internal/semver"
)

// tagReader returns the tag names of the git repository containing dir, where
// "" means the current directory. git.Tags is the real implementation.
type tagReader func(ctx context.Context, dir string, opts git.TagOptions) ([]string, error)

// tagFlags are the flags shared by every command that reads git tags.
type tagFlags struct {
	directory string
	global    bool
	prefix    string
	initial   string
}

// tagFlagNames are the tag flags, in the order help shows them.
var tagFlagNames = []string{"directory", "global", "tag-prefix", "initial"}

// add adds the tag flags to cmd.
func (f *tagFlags) add(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.StringVar(&f.directory, "directory", "", "Read tags from the git repository containing `path`")
	_ = cmd.MarkFlagDirname("directory") // Only fails if the flag doesn't exist.

	flags.BoolVar(&f.global, "global", false,
		"Read every tag in the repository, not only those on the checked-out commit's history")
	flags.StringVar(&f.prefix, "tag-prefix", "",
		"Only read tags starting with `prefix`, such as api/ for api/v1.2.3")
	flags.StringVar(&f.initial, "initial", "", "Use `version` if no tag matches, as in a new repository")

	for _, name := range tagFlagNames {
		setHelpGroup(flags, name, groupInput)
	}
}

// changed returns the first tag flag given on the command line, as typed
// (such as "--directory"), or "" if none was.
func (f *tagFlags) changed(cmd *cobra.Command) string {
	for _, name := range tagFlagNames {
		if cmd.Flags().Changed(name) {
			return "--" + name
		}
	}

	return ""
}

// query builds the tag query the flags describe. releaseFlag is the flag that
// asks for releases only, such as "--released", or "" if none did.
func (f *tagFlags) query(releaseFlag string, fallback bool) (tagQuery, error) {
	q := tagQuery{
		dir:          f.directory,
		all:          f.global,
		prefix:       f.prefix,
		releasesOnly: releaseFlag != "",
		releaseFlag:  releaseFlag,
		fallback:     fallback,
	}

	if f.initial == "" {
		return q, nil
	}

	initial, err := parseVersion(f.initial, " for --initial", false)
	if err != nil {
		return tagQuery{}, err
	}

	if q.releasesOnly && !initial.IsRelease() {
		return tagQuery{}, &userError{
			code:     ExitUsage,
			headline: fmt.Sprintf("%q can't be the --initial version with %s.", f.initial, releaseFlag),
			detail:   releaseFlag + " only gives release versions, so --initial must be one too.",
			hint:     "Give a release, as in `--initial 0.1.0`.",
		}
	}

	q.initial = &initial
	q.initialText = f.initial

	return q, nil
}

// gitFlagWithVersionError reports a tag flag given alongside a version, where
// it would have no effect.
func gitFlagWithVersionError(flag string) *userError {
	return &userError{
		code:     ExitUsage,
		headline: flag + " can't be used when a version is given.",
		detail:   "svu modify only reads git tags when no version is given.",
		hint:     "Remove " + flag + ", or remove the version.",
	}
}

// tagQuery describes which version to find in a repository's git tags, and
// why, so that errors can be worded for the command that asked.
type tagQuery struct {
	// dir is the --directory flag, or "" for the current directory.
	dir string
	// all reads every tag, not only those reachable from HEAD.
	all bool
	// atHead reads only the tags on HEAD itself.
	atHead bool
	// prefix keeps only tags starting with it, and removes it.
	prefix string
	// releasesOnly skips pre-releases.
	releasesOnly bool
	// releaseFlag is the flag that set releasesOnly, for messages.
	releaseFlag string
	// initial, if set, is the version to use when no tag matches.
	initial *semver.Version
	// initialText is --initial as typed, for its "v".
	initialText string
	// fallback is true when tags are read only because no version was given,
	// as svu modify does. Errors then suggest giving a version instead.
	fallback bool
}

func (q tagQuery) options() git.TagOptions {
	return git.TagOptions{All: q.all, AtHead: q.atHead, Prefix: q.prefix}
}

// place names the directory the query reads, for messages.
func (q tagQuery) place() string {
	if q.dir == "" {
		return "the current directory"
	}

	return fmt.Sprintf("%q", q.dir)
}

// repository names the repository the query reads, for messages.
func (q tagQuery) repository() string {
	if q.dir == "" {
		return "The git repository"
	}

	return fmt.Sprintf("The git repository at %q", q.dir)
}

// pick returns the latest version among tags that the query accepts.
func (q tagQuery) pick(tags []string) (semver.Version, bool) {
	if q.releasesOnly {
		return semver.LatestRelease(tags)
	}

	return semver.Latest(tags)
}

// source returns the text a version read for q came from, given the name of
// its tag: the name without the prefix, or --initial as typed if there was no
// tag. A version is printed with a "v" if this has one.
func (q tagQuery) source(tag string) string {
	if tag == "" {
		return q.initialText
	}

	return strings.TrimPrefix(tag, q.prefix)
}

// latestTag returns the latest git tag that is a valid version: the same
// filtering and ordering for every command that reads tags. It also returns
// the tag's name, such as "v1.2.3" or "api/v1.2.3", or "" when the version is
// --initial.
func latestTag(ctx context.Context, readTags tagReader, q tagQuery) (semver.Version, string, error) {
	tags, err := readTags(ctx, q.dir, q.options())
	if err != nil {
		return semver.Version{}, "", tagsError(err, q)
	}

	if v, ok := q.pick(tags); ok {
		return v, q.prefix + tagName(tags, v), nil
	}

	if q.initial != nil {
		return *q.initial, "", nil
	}

	var ue *userError

	if latest, ok := semver.Latest(tags); ok {
		ue = &userError{
			code:     ExitNoMatchingTag,
			headline: q.repository() + " has no released versions.",
			detail:   fmt.Sprintf("Its latest version, %q, is a pre-release.", latest.String()),
			hint:     fmt.Sprintf("Leave out `%s` to use pre-releases too.", q.releaseFlag),
		}

		if q.releaseFlag == hotfixReleaseFlag {
			ue.hint = "A hotfix patches the latest release, and there isn't one yet. Use `--patch` instead."
		}
	} else {
		ue = noVersionTagsError(q, len(tags))
	}

	// Tags on other branches don't count by default. Say so if they would
	// have given an answer.
	if !q.all && !q.atHead {
		everything := q
		everything.all = true

		if all, err := readTags(ctx, q.dir, everything.options()); err == nil {
			if _, ok := q.pick(all); ok {
				ue.hint = "Tags that aren't on the checked-out commit's history have versions. " +
					"To include them, use `--global`."
			}
		}
	}

	return semver.Version{}, "", ue
}

// tagName returns the first of tags that is version v.
func tagName(tags []string, v semver.Version) string {
	for _, tag := range tags {
		if t, err := semver.Parse(tag); err == nil && semver.Compare(t, v) == 0 {
			return tag
		}
	}

	return ""
}

func noVersionTagsError(q tagQuery, count int) *userError {
	which := "tags"
	if q.prefix != "" {
		which = fmt.Sprintf("tags starting with %q", q.prefix)
	}

	where := ""

	switch {
	case q.atHead:
		where = " on the checked-out commit"
	case !q.all:
		where = " on the checked-out commit's history"
	}

	detail := fmt.Sprintf("None of its %s%s is a valid version, such as v1.2.3 or v1.2.3-rc-001.", which, where)
	if count == 0 {
		detail = fmt.Sprintf("It has no %s%s.", which, where)
	}

	example := fmt.Sprintf("`git tag %sv0.1.0`", q.prefix)

	hint := fmt.Sprintf("Tag a release, as in %s, or use `--initial 0.0.0` to start from scratch.", example)
	switch {
	case q.atHead:
		hint = "The checked-out commit has no version tag yet. Tag it, or leave out `--at-head`."
	case q.fallback:
		hint = fmt.Sprintf("Give a version, as in `svu modify 0.1.0`, tag a release, as in %s, "+
			"or use `--initial 0.0.0`.", example)
	}

	return &userError{
		code:     ExitNoMatchingTag,
		headline: q.repository() + " has no tags that are valid versions.",
		detail:   detail,
		hint:     hint,
	}
}

func tagsError(err error, q tagQuery) *userError {
	var cmdErr *git.CommandError

	switch {
	case errors.Is(err, git.ErrNotRepository):
		return notRepositoryError(err, q)
	case errors.Is(err, git.ErrDirectoryNotFound):
		return &userError{
			code:     ExitRepository,
			headline: q.place() + " does not exist.",
			detail:   "--directory must be a directory in a git repository.",
			err:      err,
		}
	case errors.Is(err, git.ErrNotDirectory):
		return &userError{
			code:     ExitRepository,
			headline: q.place() + " is not a directory.",
			detail:   "--directory must be a directory in a git repository.",
			err:      err,
		}
	case errors.Is(err, git.ErrNotInstalled):
		ue := &userError{
			code:     ExitGit,
			headline: "svu could not find git.",
			detail:   "svu needs git to read the latest version from the tags.",
			hint:     "Install git.",
			err:      err,
		}

		if q.fallback {
			ue.detail = "No version was given, so svu needs git to read the latest version from the tags."
			ue.hint = "Install git, or give a version, as in `svu modify 1.2.3`."
		}

		return ue
	case errors.As(err, &cmdErr) && cmdErr.Stderr != "":
		return &userError{
			code:     ExitGit,
			headline: "git could not list the tags in " + q.place() + ".",
			detail:   cmdErr.Stderr,
			err:      err,
		}
	default:
		return &userError{
			code:     ExitInternal,
			headline: "svu could not read the git tags in " + q.place() + ".",
			hint:     reportHint,
			err:      err,
		}
	}
}

func notRepositoryError(err error, q tagQuery) *userError {
	if q.fallback {
		hint := "Give a version, as in `svu modify 1.2.3`, or run svu inside a git repository."
		if q.dir != "" {
			hint = "Give a version, as in `svu modify 1.2.3`, or set `--directory` to a git repository."
		}

		return &userError{
			code:     ExitRepository,
			headline: "svu has no valid input.",
			detail: "No version was given, so svu reads the latest version from git tags,\n" +
				"but " + q.place() + " is not in a git repository.",
			hint: hint,
			err:  err,
		}
	}

	headline := fmt.Sprintf("%q is not in a git repository.", q.dir)
	hint := "Set `--directory` to a directory in a git repository."

	if q.dir == "" {
		headline = "The current directory is not in a git repository."
		hint = "Run svu inside a git repository, or set `--directory` to one."
	}

	return &userError{
		code:     ExitRepository,
		headline: headline,
		detail:   "svu reads the latest version from the repository's git tags.",
		hint:     hint,
		err:      err,
	}
}
