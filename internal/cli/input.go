package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/tedslittlerobot/svm/internal/git"
	"github.com/tedslittlerobot/svm/internal/semver"
)

// tagReader returns the tag names of the git repository containing dir, where
// "" means the current directory. git.Tags is the real implementation.
type tagReader func(ctx context.Context, dir string) ([]string, error)

// inputOptions are the flags that affect where svm reads its version from.
type inputOptions struct {
	// directory is the --directory flag.
	directory string
	// directorySet reports whether --directory was given at all.
	directorySet bool
}

// stdinArgument is the argument that tells svm to read the version from
// standard input.
const stdinArgument = "-"

// readVersion works out which version svm starts from:
//
//  1. the version argument, or standard input if the argument is "-";
//  2. with no argument, the latest git tag that is a valid version, in the
//     repository containing the current directory or --directory.
//
// Standard input is only read when asked for with "-". Reading it whenever it
// isn't a terminal would hang wherever stdin is a pipe that never closes (CI
// runners, IDEs, subprocesses), and would consume input meant for something
// else (git hooks, while-read loops).
func readVersion(
	ctx context.Context, args []string, stdin io.Reader, opts inputOptions, readTags tagReader,
) (semver.Version, error) {
	switch {
	case len(args) == 0:
		return latestTag(ctx, opts.directory, readTags)
	case opts.directorySet:
		return semver.Version{}, &userError{
			headline: "--directory can't be used when a version is given.",
			detail:   "svm only reads git tags when no version is given.",
			hint:     "Remove --directory, or remove the version.",
		}
	case args[0] == stdinArgument:
		input, err := readStdin(stdin)
		if err != nil {
			return semver.Version{}, err
		}

		return parseVersion(input, true)
	default:
		return parseVersion(args[0], false)
	}
}

// readStdin reads one version from standard input, with surrounding
// whitespace removed.
func readStdin(stdin io.Reader) (string, error) {
	b, err := io.ReadAll(stdin)
	if err != nil {
		return "", &userError{
			headline: "svm could not read standard input.",
			hint:     reportHint,
			err:      err,
		}
	}

	input := strings.TrimSpace(string(b))

	switch {
	case input == "":
		return "", &userError{
			headline: "Standard input is empty.",
			detail:   `With "-" as the argument, svm reads one version from standard input.`,
			hint:     "Pipe a version into svm, as in echo 1.2.3 | svm -.",
		}
	case strings.ContainsAny(input, "\r\n"):
		return "", &userError{
			headline: "Standard input has more than one line.",
			detail:   `With "-" as the argument, svm reads one version from standard input.`,
		}
	}

	return input, nil
}

// parseVersion parses a version given as an argument or on standard input.
func parseVersion(input string, fromStdin bool) (semver.Version, error) {
	v, err := semver.Parse(input)
	if err == nil {
		return v, nil
	}

	var perr *semver.ParseError
	if !errors.As(err, &perr) {
		return semver.Version{}, &userError{headline: "svm could not read the version.", hint: reportHint, err: err}
	}

	return semver.Version{}, parseError(perr, fromStdin)
}

// ruleDetails explains each version rule in terms of conventions/semver.md.
var ruleDetails = map[semver.Rule]string{
	semver.RuleEmpty:               "A version looks like 1.2.3, or 1.2.3-rc-001 for a pre-release.",
	semver.RuleUppercasePrefix:     `The "v" prefix must be lowercase.`,
	semver.RuleMissingNumber:       "A version must have three numbers, MAJOR.MINOR.PATCH, as in 1.2.3.",
	semver.RuleNumber:              "MAJOR, MINOR and PATCH must be whole numbers, as in 1.2.3.",
	semver.RuleLeadingZero:         "MAJOR, MINOR and PATCH must not have leading zeros.",
	semver.RuleNumberTooLarge:      "MAJOR, MINOR and PATCH must be no larger than 18446744073709551615.",
	semver.RuleTooManyNumbers:      "A version must have exactly three numbers, MAJOR.MINOR.PATCH, as in 1.2.3.",
	semver.RuleUnexpectedCharacter: "A version looks like 1.2.3, or 1.2.3-rc-001 for a pre-release.",
	semver.RuleBuildMetadata:       `Build metadata, the part starting with "+", isn't supported.`,
	semver.RuleStage:               "The stage must be alpha, beta or rc, in lowercase.",
	semver.RuleMissingIndex:        "The stage must be followed by a three-digit index, as in 1.2.3-rc-001.",
	semver.RuleStageSeparator:      "The stage and index must be separated by a hyphen, as in 1.2.3-rc-001.",
	semver.RuleIndex:               "The stage index must be exactly three digits, from 001 to 999.",
	semver.RuleExtraSuffix:         "A version can have only one stage and index, as in 1.2.3-rc-001.",
}

func parseError(perr *semver.ParseError, fromStdin bool) *userError {
	if perr.Rule == semver.RuleEmpty {
		return &userError{headline: "The version is empty.", detail: ruleDetails[perr.Rule], err: perr}
	}

	where := ""
	if fromStdin {
		where = " on standard input"
	}

	ue := &userError{
		headline: fmt.Sprintf("%q%s is not a valid version.", perr.Input, where),
		pointer:  &pointer{input: perr.Input, pos: perr.Pos},
		detail:   ruleDetails[perr.Rule],
		err:      perr,
	}

	switch {
	case perr.Suggestion != "":
		ue.hint = fmt.Sprintf("Did you mean %q?", perr.Suggestion)
	case !fromStdin && looksLikeCommand(perr.Input):
		ue.hint = "To list svm's commands, run svm --help."
	}

	return ue
}

// looksLikeCommand reports whether s could be a mistyped command name.
func looksLikeCommand(s string) bool {
	return strings.Trim(s, "abcdefghijklmnopqrstuvwxyz-") == ""
}

// latestTag returns the latest git tag in dir that is a valid version.
func latestTag(ctx context.Context, dir string, readTags tagReader) (semver.Version, error) {
	tags, err := readTags(ctx, dir)
	if err != nil {
		return semver.Version{}, tagsError(err, dir)
	}

	if v, ok := semver.Latest(tags); ok {
		return v, nil
	}

	headline := "The git repository has no tags that are valid versions."
	if dir != "" {
		headline = fmt.Sprintf("The git repository at %q has no tags that are valid versions.", dir)
	}

	detail := "None of its tags is a valid version, such as v1.2.3 or v1.2.3-rc-001."
	if len(tags) == 0 {
		detail = "It has no tags."
	}

	return semver.Version{}, &userError{
		headline: headline,
		detail:   detail,
		hint:     "Give a version, as in svm 0.1.0, or tag a release, as in git tag v0.1.0.",
	}
}

func tagsError(err error, dir string) *userError {
	place := "the current directory"
	if dir != "" {
		place = fmt.Sprintf("%q", dir)
	}

	var cmdErr *git.CommandError

	switch {
	case errors.Is(err, git.ErrNotRepository):
		hint := "Give a version, as in svm 1.2.3, or run svm inside a git repository."
		if dir != "" {
			hint = "Give a version, as in svm 1.2.3, or set --directory to a git repository."
		}

		return &userError{
			headline: "svm has no valid input.",
			detail: "No version was given, so svm reads the latest version from git tags,\n" +
				"but " + place + " is not in a git repository.",
			hint: hint,
			err:  err,
		}
	case errors.Is(err, git.ErrDirectoryNotFound):
		return &userError{
			headline: place + " does not exist.",
			detail:   "--directory must be a directory in a git repository.",
			err:      err,
		}
	case errors.Is(err, git.ErrNotDirectory):
		return &userError{
			headline: place + " is not a directory.",
			detail:   "--directory must be a directory in a git repository.",
			err:      err,
		}
	case errors.Is(err, git.ErrNotInstalled):
		return &userError{
			headline: "svm could not find git.",
			detail:   "No version was given, so svm needs git to read the latest version from the tags.",
			hint:     "Install git, or give a version, as in svm 1.2.3.",
			err:      err,
		}
	case errors.As(err, &cmdErr) && cmdErr.Stderr != "":
		return &userError{
			headline: "git could not list the tags in " + place + ".",
			detail:   cmdErr.Stderr,
			err:      err,
		}
	default:
		return &userError{headline: "svm could not read the git tags in " + place + ".", hint: reportHint, err: err}
	}
}
