package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/tedslittlerobot/svu/internal/semver"
)

// stdinArgument is the argument that tells svu to read the version from
// standard input.
const stdinArgument = "-"

// readVersion works out which version svu starts from:
//
//  1. the version argument, or standard input if the argument is "-";
//  2. with no argument, the latest git tag that q matches.
//
// Standard input is only read when asked for with "-". Reading it whenever it
// isn't a terminal would hang wherever stdin is a pipe that never closes (CI
// runners, IDEs, subprocesses), and would consume input meant for something
// else (git hooks, while-read loops).
func readVersion(
	ctx context.Context, args []string, stdin io.Reader, q tagQuery, readTags tagReader,
) (semver.Version, error) {
	switch {
	case len(args) == 0:
		return latestTag(ctx, readTags, q)
	case args[0] == stdinArgument:
		input, err := readStdin(stdin)
		if err != nil {
			return semver.Version{}, err
		}

		return parseVersion(input, " on standard input", false)
	default:
		return parseVersion(args[0], "", true)
	}
}

// readStdin reads one version from standard input, with surrounding
// whitespace removed.
func readStdin(stdin io.Reader) (string, error) {
	b, err := io.ReadAll(stdin)
	if err != nil {
		return "", &userError{
			code:     ExitInternal,
			headline: "svu could not read standard input.",
			hint:     reportHint,
			err:      err,
		}
	}

	input := strings.TrimSpace(string(b))

	switch {
	case input == "":
		return "", &userError{
			code:     ExitInvalidVersion,
			headline: "Standard input is empty.",
			detail:   `With "-" as the argument, svu reads one version from standard input.`,
			hint:     "Pipe a version into svu, as in `echo 1.2.3 | svu modify -`.",
		}
	case strings.ContainsAny(input, "\r\n"):
		return "", &userError{
			code:     ExitInvalidVersion,
			headline: "Standard input has more than one line.",
			detail:   `With "-" as the argument, svu reads one version from standard input.`,
		}
	}

	return input, nil
}

// parseVersion parses a version the user gave. where says where it came
// from, for messages: "" for the version argument, or a phrase such as
// " on standard input". modifyHints is true for svu modify's version
// argument, where an invalid word is probably a mistyped modifier.
func parseVersion(input, where string, modifyHints bool) (semver.Version, error) {
	v, err := semver.Parse(input)
	if err == nil {
		return v, nil
	}

	var perr *semver.ParseError
	if !errors.As(err, &perr) {
		return semver.Version{}, &userError{code: ExitInternal, headline: "svu could not read the version.", hint: reportHint, err: err}
	}

	return semver.Version{}, parseError(perr, where, modifyHints)
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

func parseError(perr *semver.ParseError, where string, modifyHints bool) *userError {
	if perr.Rule == semver.RuleEmpty {
		return &userError{
			code:     ExitInvalidVersion,
			headline: "The version" + where + " is empty.",
			detail:   ruleDetails[perr.Rule],
			err:      perr,
		}
	}

	ue := &userError{
		code:     ExitInvalidVersion,
		headline: fmt.Sprintf("%q%s is not a valid version.", perr.Input, where),
		pointer:  &pointer{input: perr.Input, pos: perr.Pos},
		detail:   ruleDetails[perr.Rule],
		err:      perr,
	}

	switch {
	case perr.Suggestion != "":
		ue.hint = fmt.Sprintf("Did you mean %q?", perr.Suggestion)
	case !modifyHints:
		// No hint: the input isn't svu modify's version argument.
	case slices.Contains(modifierNames(), perr.Input):
		ue.hint = fmt.Sprintf("Did you mean `--%s`?", perr.Input)
	case isWord(perr.Input):
		ue.hint = "To see what svu modify accepts, run `svu modify --help`."
	}

	return ue
}

// isWord reports whether s is made only of lowercase letters and hyphens, as a
// mistyped command or flag would be.
func isWord(s string) bool {
	return strings.Trim(s, "abcdefghijklmnopqrstuvwxyz-") == ""
}
