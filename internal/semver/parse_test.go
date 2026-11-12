package semver

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

func TestParseValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  Version
	}{
		// §Examples, Valid.
		{name: "accepts a release", input: "1.2.3", want: Version{Major: 1, Minor: 2, Patch: 3}},
		{name: "accepts a release with a prefix", input: "v1.2.3", want: Version{Major: 1, Minor: 2, Patch: 3}},
		{name: "accepts zero", input: "0.0.0", want: Version{}},
		{
			name:  "accepts the first alpha",
			input: "1.2.3-alpha-001",
			want:  Version{Major: 1, Minor: 2, Patch: 3, Stage: Alpha, Index: 1},
		},
		{
			name:  "accepts a prefix and suffix together",
			input: "v2.3.4-rc-987",
			want:  Version{Major: 2, Minor: 3, Patch: 4, Stage: RC, Index: 987},
		},
		{
			name:  "accepts the highest index",
			input: "10.20.30-beta-999",
			want:  Version{Major: 10, Minor: 20, Patch: 30, Stage: Beta, Index: 999},
		},
		// Further cases.
		{
			name:  "accepts the largest numbers",
			input: "18446744073709551615.0.1",
			want:  Version{Major: 18446744073709551615, Patch: 1},
		},
		{
			name:  "accepts an index with no zero padding needed",
			input: "1.0.0-beta-100",
			want:  Version{Major: 1, Stage: Beta, Index: 100},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v, want nil", tt.input, err)
			}

			if got != tt.want {
				t.Errorf("Parse(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseInvalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		wantRule       Rule
		wantPos        int
		wantSuggestion string
	}{
		// §Examples, Invalid.
		{name: "rejects a missing patch", input: "1.2", wantRule: RuleMissingNumber, wantPos: 3},
		{
			name: "rejects a leading zero", input: "01.2.3",
			wantRule: RuleLeadingZero, wantPos: 0, wantSuggestion: "1.2.3",
		},
		{
			name: "rejects an uppercase prefix", input: "V1.2.3",
			wantRule: RuleUppercasePrefix, wantPos: 0, wantSuggestion: "v1.2.3",
		},
		{
			name: "rejects a suffix with no index", input: "1.2.3-alpha",
			wantRule: RuleMissingIndex, wantPos: 11, wantSuggestion: "1.2.3-alpha-001",
		},
		{
			name: "rejects a one-digit index", input: "1.2.3-alpha-1",
			wantRule: RuleIndex, wantPos: 12, wantSuggestion: "1.2.3-alpha-001",
		},
		{
			name: "rejects a four-digit index", input: "1.2.3-alpha-0001",
			wantRule: RuleIndex, wantPos: 12, wantSuggestion: "1.2.3-alpha-001",
		},
		{name: "rejects an index of 000", input: "1.2.3-alpha-000", wantRule: RuleIndex, wantPos: 12},
		{
			name: "rejects a dot between stage and index", input: "1.2.3-alpha.001",
			wantRule: RuleStageSeparator, wantPos: 11, wantSuggestion: "1.2.3-alpha-001",
		},
		{
			name: "rejects no separator between stage and index", input: "1.2.3-alpha001",
			wantRule: RuleStageSeparator, wantPos: 11, wantSuggestion: "1.2.3-alpha-001",
		},
		{
			name: "rejects an uppercase stage", input: "1.2.3-Alpha-001",
			wantRule: RuleStage, wantPos: 6, wantSuggestion: "1.2.3-alpha-001",
		},
		{name: "rejects an unknown stage", input: "1.2.3-dev-001", wantRule: RuleStage, wantPos: 6},
		{name: "rejects ga as a stage", input: "1.2.3-ga-001", wantRule: RuleStage, wantPos: 6},
		{name: "rejects an index without a stage", input: "1.2.3-001", wantRule: RuleStage, wantPos: 6},
		{name: "rejects more than one stage and index", input: "1.2.3-rc-001-002", wantRule: RuleExtraSuffix, wantPos: 12},
		{
			name: "rejects build metadata", input: "1.2.3+build.5",
			wantRule: RuleBuildMetadata, wantPos: 5, wantSuggestion: "1.2.3",
		},
		{
			name: "rejects build metadata after a suffix", input: "1.2.3-rc-001+build.5",
			wantRule: RuleBuildMetadata, wantPos: 12, wantSuggestion: "1.2.3-rc-001",
		},
		// Further cases.
		{name: "rejects an empty string", input: "", wantRule: RuleEmpty, wantPos: 0},
		{name: "rejects a bare prefix", input: "v", wantRule: RuleMissingNumber, wantPos: 1},
		{name: "rejects a word", input: "latest", wantRule: RuleNumber, wantPos: 0},
		{name: "rejects a trailing dot", input: "1.2.", wantRule: RuleMissingNumber, wantPos: 4},
		{name: "rejects an empty number", input: "1..3", wantRule: RuleNumber, wantPos: 2},
		{name: "rejects a suffix before the patch", input: "1.2-rc-001", wantRule: RuleMissingNumber, wantPos: 3},
		{name: "rejects build metadata before the patch", input: "1.2+5", wantRule: RuleMissingNumber, wantPos: 3},
		{name: "rejects other separators", input: "1,2,3", wantRule: RuleUnexpectedCharacter, wantPos: 1},
		{name: "rejects a fourth number", input: "1.2.3.4", wantRule: RuleTooManyNumbers, wantPos: 5},
		{name: "rejects letters after the patch", input: "1.2.3a", wantRule: RuleUnexpectedCharacter, wantPos: 5},
		{
			name: "rejects surrounding whitespace", input: " 1.2.3 ",
			wantRule: RuleNumber, wantPos: 0, wantSuggestion: "1.2.3",
		},
		{name: "rejects an empty suffix", input: "1.2.3-", wantRule: RuleStage, wantPos: 6},
		{
			name: "rejects an empty index", input: "1.2.3-rc-",
			wantRule: RuleIndex, wantPos: 9, wantSuggestion: "1.2.3-rc-001",
		},
		{
			name: "rejects build metadata straight after the stage", input: "1.2.3-rc+5",
			wantRule: RuleBuildMetadata, wantPos: 8, wantSuggestion: "1.2.3-rc-001",
		},
		{name: "rejects a number too large for uint64", input: "1.18446744073709551616.0", wantRule: RuleNumberTooLarge, wantPos: 2},
		{name: "rejects a leading zero in the patch", input: "1.2.03", wantRule: RuleLeadingZero, wantPos: 4, wantSuggestion: "1.2.3"},
		{name: "rejects an index too large to fix", input: "1.2.3-rc-1000", wantRule: RuleIndex, wantPos: 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse(tt.input)
			if err == nil {
				t.Fatalf("Parse(%q) = %v, want an error", tt.input, got)
			}

			var perr *ParseError
			if !errors.As(err, &perr) {
				t.Fatalf("Parse(%q) error = %T, want *ParseError", tt.input, err)
			}

			want := ParseError{Input: tt.input, Pos: tt.wantPos, Rule: tt.wantRule, Suggestion: tt.wantSuggestion}
			if *perr != want {
				t.Errorf("Parse(%q) error = %#v, want %#v", tt.input, *perr, want)
			}
		})
	}
}

func TestParseErrorMessage(t *testing.T) {
	t.Parallel()

	_, err := Parse("1.2.3-alpha-1")

	want := `parsing version "1.2.3-alpha-1": index not three digits from 001 to 999 at byte 12`
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestRuleString(t *testing.T) {
	t.Parallel()

	for r := RuleEmpty; r <= RuleExtraSuffix; r++ {
		if strings.HasPrefix(r.String(), "rule ") {
			t.Errorf("Rule(%d).String() = %q, want a description", r, r.String())
		}
	}

	if got, want := Rule(0).String(), "rule 0"; got != want {
		t.Errorf("Rule(0).String() = %q, want %q", got, want)
	}
}

// specPattern is the whole-version pattern from §Format.
var specPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(alpha|beta|rc)-([0-9]{3}))?$`)

func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"1.2.3", "v1.2.3", "0.0.0", "1.2.3-alpha-001", "v2.3.4-rc-987", "10.20.30-beta-999",
		"1.2", "01.2.3", "V1.2.3", "1.2.3-alpha", "1.2.3-alpha-1", "1.2.3-alpha-000",
		"1.2.3-alpha.001", "1.2.3-Alpha-001", "1.2.3-001", "1.2.3+build.5", "",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		v, err := Parse(s)

		// The pattern can't express that 000 is not a valid index (§Format).
		wantValid := specPattern.MatchString(s) && !strings.HasSuffix(s, "-000")

		if err != nil {
			// The pattern doesn't limit the size of numbers; semvu limits them
			// to uint64.
			var perr *ParseError

			tooLarge := errors.As(err, &perr) && perr.Rule == RuleNumberTooLarge
			if wantValid && !tooLarge {
				t.Fatalf("Parse(%q) error = %v, want nil", s, err)
			}

			checkParseError(t, s, err)

			return
		}

		if !wantValid {
			t.Fatalf("Parse(%q) = %v, want an error", s, v)
		}

		if got, want := v.String(), strings.TrimPrefix(s, "v"); got != want {
			t.Errorf("Parse(%q).String() = %q, want %q", s, got, want)
		}
	})
}

func checkParseError(t *testing.T, s string, err error) {
	t.Helper()

	var perr *ParseError
	if !errors.As(err, &perr) {
		t.Fatalf("Parse(%q) error = %T, want *ParseError", s, err)
	}

	if perr.Pos < 0 || perr.Pos > len(s) {
		t.Errorf("Parse(%q) error Pos = %d, want 0 to %d", s, perr.Pos, len(s))
	}

	if perr.Suggestion != "" {
		if _, err := Parse(perr.Suggestion); err != nil {
			t.Errorf("Parse(%q) suggested %q, which is invalid: %v", s, perr.Suggestion, err)
		}
	}
}
