package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Rule identifies which rule in conventions/semver.md an invalid version
// breaks.
type Rule int

// The rules a version can break. The zero Rule is not used.
const (
	// RuleEmpty: the version is the empty string.
	RuleEmpty Rule = iota + 1
	// RuleUppercasePrefix: the prefix is "V", not "v" (§Format).
	RuleUppercasePrefix
	// RuleMissingNumber: the version ends, or its suffix starts, before
	// MAJOR, MINOR and PATCH are all present (§Format).
	RuleMissingNumber
	// RuleNumber: something other than a digit is where MAJOR, MINOR or PATCH
	// should be (§Format).
	RuleNumber
	// RuleLeadingZero: MAJOR, MINOR or PATCH has a leading zero (§Format,
	// SemVer §2).
	RuleLeadingZero
	// RuleNumberTooLarge: MAJOR, MINOR or PATCH doesn't fit in a uint64.
	RuleNumberTooLarge
	// RuleTooManyNumbers: there is a fourth dot-separated number.
	RuleTooManyNumbers
	// RuleUnexpectedCharacter: a character that can't appear at this point in
	// any valid version.
	RuleUnexpectedCharacter
	// RuleBuildMetadata: the version contains "+" (§Format, SemVer §10).
	RuleBuildMetadata
	// RuleStage: the suffix doesn't start with alpha, beta or rc (§Stages).
	RuleStage
	// RuleMissingIndex: the suffix has a stage but no index (§Format).
	RuleMissingIndex
	// RuleStageSeparator: the stage and index aren't separated by "-"
	// (§Format).
	RuleStageSeparator
	// RuleIndex: the index isn't exactly three digits from 001 to 999
	// (§Format).
	RuleIndex
	// RuleExtraSuffix: something follows the index (§Format).
	RuleExtraSuffix
)

var ruleDescriptions = map[Rule]string{
	RuleEmpty:               "empty version",
	RuleUppercasePrefix:     "uppercase prefix",
	RuleMissingNumber:       "missing number",
	RuleNumber:              "expected a number",
	RuleLeadingZero:         "leading zero",
	RuleNumberTooLarge:      "number too large",
	RuleTooManyNumbers:      "more than three numbers",
	RuleUnexpectedCharacter: "unexpected character",
	RuleBuildMetadata:       "build metadata not supported",
	RuleStage:               "unknown stage",
	RuleMissingIndex:        "missing index",
	RuleStageSeparator:      "stage and index not separated by a hyphen",
	RuleIndex:               "index not three digits from 001 to 999",
	RuleExtraSuffix:         "more than one stage and index",
}

// String describes the rule briefly, for use in Go error strings.
func (r Rule) String() string {
	if d, ok := ruleDescriptions[r]; ok {
		return d
	}

	return "rule " + strconv.Itoa(int(r))
}

// ParseError reports why a string is not a valid svm version.
type ParseError struct {
	// Input is the string that failed to parse, exactly as given.
	Input string
	// Pos is the byte offset in Input where the problem starts. It equals
	// len(Input) when the problem is that Input ends too soon.
	Pos int
	// Rule is the first rule Input breaks.
	Rule Rule
	// Suggestion is a valid version that Input was probably meant to be, or
	// "" if there isn't an obvious one.
	Suggestion string
}

// Error implements error.
func (e *ParseError) Error() string {
	return fmt.Sprintf("parsing version %q: %s at byte %d", e.Input, e.Rule, e.Pos)
}

var stagesByName = map[string]Stage{
	"alpha": Alpha,
	"beta":  Beta,
	"rc":    RC,
}

// Parse parses s as an svm version (§Format). A leading "v" is accepted and
// discarded. If s is invalid, the error is a *ParseError describing the first
// problem found.
func Parse(s string) (Version, error) {
	v, err := parse(s)
	if err != nil {
		err.Suggestion = suggest(s)

		return Version{}, err
	}

	return v, nil
}

// parse does the work of Parse without looking for a suggestion.
func parse(s string) (Version, *ParseError) {
	fail := func(pos int, rule Rule) (Version, *ParseError) {
		return Version{}, &ParseError{Input: s, Pos: pos, Rule: rule}
	}

	if s == "" {
		return fail(0, RuleEmpty)
	}

	pos := 0

	switch s[0] {
	case 'v':
		pos++
	case 'V':
		return fail(0, RuleUppercasePrefix)
	}

	var numbers [3]uint64

	for i := range numbers {
		if i > 0 {
			switch {
			case pos == len(s) || s[pos] == '-' || s[pos] == '+':
				return fail(pos, RuleMissingNumber)
			case s[pos] != '.':
				return fail(pos, RuleUnexpectedCharacter)
			}

			pos++
		}

		n, end, rule := number(s, pos)
		if rule != 0 {
			return fail(pos, rule)
		}

		numbers[i], pos = n, end
	}

	v := Version{Major: numbers[0], Minor: numbers[1], Patch: numbers[2]}

	if pos == len(s) {
		return v, nil
	}

	switch s[pos] {
	case '-':
		// The suffix follows.
	case '+':
		return fail(pos, RuleBuildMetadata)
	case '.':
		return fail(pos, RuleTooManyNumbers)
	default:
		return fail(pos, RuleUnexpectedCharacter)
	}

	pos++

	// Stage (§Stages).
	start := pos
	for pos < len(s) && isLetter(s[pos]) {
		pos++
	}

	stage, ok := stagesByName[s[start:pos]]
	if !ok {
		return fail(start, RuleStage)
	}

	switch {
	case pos == len(s):
		return fail(pos, RuleMissingIndex)
	case s[pos] == '+':
		return fail(pos, RuleBuildMetadata)
	case s[pos] != '-':
		return fail(pos, RuleStageSeparator)
	}

	pos++

	// Index: exactly three digits, 001 to 999 (§Format).
	start = pos
	for pos < len(s) && isDigit(s[pos]) {
		pos++
	}

	index, err := strconv.Atoi(s[start:pos])
	if pos-start != 3 || err != nil || index < MinIndex {
		return fail(start, RuleIndex)
	}

	if pos < len(s) {
		if s[pos] == '+' {
			return fail(pos, RuleBuildMetadata)
		}

		return fail(pos, RuleExtraSuffix)
	}

	v.Stage, v.Index = stage, index

	return v, nil
}

// number reads MAJOR, MINOR or PATCH starting at pos. It returns the value,
// the position just after it, and the rule broken, or 0 if none.
func number(s string, pos int) (n uint64, end int, rule Rule) {
	end = pos
	for end < len(s) && isDigit(s[end]) {
		end++
	}

	switch {
	case end == pos && pos == len(s):
		return 0, pos, RuleMissingNumber
	case end == pos:
		return 0, pos, RuleNumber
	case s[pos] == '0' && end-pos > 1:
		return 0, pos, RuleLeadingZero
	}

	n, err := strconv.ParseUint(s[pos:end], 10, 64)
	if err != nil {
		return 0, pos, RuleNumberTooLarge
	}

	return n, end, 0
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

var (
	// leadingZeros matches a run of zeros at the start of a number.
	leadingZeros = regexp.MustCompile(`(^v?|\.)0+([0-9])`)
	// looseSuffix matches a suffix with a recognisable stage and an index
	// that is loosely written or missing.
	looseSuffix = regexp.MustCompile(`^(alpha|beta|rc)[-.]?([0-9]{0,4})$`)
)

// suggest fixes common mistakes in s, such as the wrong case, a short index
// or build metadata. It returns the fixed version if that is valid and
// different from s, or "" otherwise.
func suggest(s string) string {
	fixed := strings.ToLower(strings.TrimSpace(s))
	fixed, _, _ = strings.Cut(fixed, "+")

	core, suffix, hasSuffix := strings.Cut(fixed, "-")
	fixed = leadingZeros.ReplaceAllString(core, "$1$2")

	if hasSuffix {
		m := looseSuffix.FindStringSubmatch(suffix)
		if m == nil {
			return ""
		}

		index := MinIndex
		if m[2] != "" {
			index, _ = strconv.Atoi(m[2]) // The pattern guarantees digits.
		}

		fixed += fmt.Sprintf("-%s-%03d", m[1], index)
	}

	if fixed == s {
		return ""
	}

	if _, err := parse(fixed); err != nil {
		return ""
	}

	return fixed
}
