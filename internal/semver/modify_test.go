package semver

import (
	"errors"
	"fmt"
	"testing"
)

func TestBump(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		from  string
		part  Part
		stage Stage
		want  string
	}{
		{name: "bumps major and resets minor and patch", from: "1.2.3", part: Major, want: "2.0.0"},
		{name: "bumps minor and resets patch", from: "1.2.3", part: Minor, want: "1.3.0"},
		{name: "bumps patch only", from: "1.2.3", part: Patch, want: "1.2.4"},
		{name: "removes a suffix when bumping major", from: "1.2.3-rc-004", part: Major, want: "2.0.0"},
		{name: "removes a suffix when bumping minor", from: "1.2.3-beta-002", part: Minor, want: "1.3.0"},
		{name: "removes a suffix when bumping patch", from: "1.2.3-alpha-099", part: Patch, want: "1.2.4"},
		{name: "bumps from zero", from: "0.0.0", part: Patch, want: "0.0.1"},
		{name: "starts a stage at 001", from: "1.2.3", part: Minor, stage: Alpha, want: "1.3.0-alpha-001"},
		{name: "starts any stage first", from: "1.2.3", part: Major, stage: RC, want: "2.0.0-rc-001"},
		{
			name: "starts an earlier stage on a new version", from: "1.2.3-rc-005",
			part: Patch, stage: Alpha, want: "1.2.4-alpha-001",
		},
		{
			name: "starts the same stage again at 001 on a new version", from: "1.2.3-beta-007",
			part: Patch, stage: Beta, want: "1.2.4-beta-001",
		},
		{
			name: "bumps only the part asked for when the others are at their limit",
			from: "1.18446744073709551615.18446744073709551615", part: Major, want: "2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := mustParse(t, tt.from).Bump(tt.part, tt.stage)
			if err != nil {
				t.Fatalf("Bump(%s, %s) error = %v, want nil", tt.part, tt.stage.name(), err)
			}

			if got.String() != tt.want {
				t.Errorf("%s.Bump(%s, %s) = %s, want %s", tt.from, tt.part, tt.stage.name(), got, tt.want)
			}
		})
	}
}

func TestBumpOverflow(t *testing.T) {
	t.Parallel()

	for _, part := range []Part{Major, Minor, Patch} {
		t.Run(fmt.Sprintf("refuses to bump %s past the largest number", part), func(t *testing.T) {
			t.Parallel()

			v := Version{Major: 18446744073709551615, Minor: 18446744073709551615, Patch: 18446744073709551615}

			_, err := v.Bump(part, Release)

			var oerr *OverflowError
			if !errors.As(err, &oerr) {
				t.Fatalf("Bump(%s) error = %v, want *OverflowError", part, err)
			}

			if want := (OverflowError{Version: v, Part: part}); *oerr != want {
				t.Errorf("Bump(%s) error = %#v, want %#v", part, *oerr, want)
			}
		})
	}
}

func TestBumpRejectsUnknownValues(t *testing.T) {
	t.Parallel()

	v := Version{Major: 1}

	if _, err := v.Bump(Part(0), Release); err == nil {
		t.Error("Bump(Part(0)) error = nil, want an error")
	}

	if _, err := v.Bump(Major, Stage(99)); err == nil {
		t.Error("Bump(Major, Stage(99)) error = nil, want an error")
	}
}

func TestAdvance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from string
		to   Stage
		want string
	}{
		{name: "releases a release candidate", from: "1.2.3-rc-001", to: Release, want: "1.2.3"},
		{name: "releases an alpha", from: "1.2.3-alpha-042", to: Release, want: "1.2.3"},
		{name: "increments the index within a stage", from: "1.2.3-rc-001", to: RC, want: "1.2.3-rc-002"},
		{name: "increments the index into three digits", from: "1.2.3-beta-099", to: Beta, want: "1.2.3-beta-100"},
		{name: "increments the index to the last one", from: "1.2.3-alpha-998", to: Alpha, want: "1.2.3-alpha-999"},
		{name: "moves to a later stage at 001", from: "1.2.3-alpha-099", to: RC, want: "1.2.3-rc-001"},
		{name: "moves to the next stage at 001", from: "1.2.3-alpha-003", to: Beta, want: "1.2.3-beta-001"},
		{name: "moves to a later stage from the last index", from: "1.2.3-beta-999", to: RC, want: "1.2.3-rc-001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := mustParse(t, tt.from).Advance(tt.to)
			if err != nil {
				t.Fatalf("%s.Advance(%s) error = %v, want nil", tt.from, tt.to.name(), err)
			}

			if got.String() != tt.want {
				t.Errorf("%s.Advance(%s) = %s, want %s", tt.from, tt.to.name(), got, tt.want)
			}
		})
	}
}

func TestAdvanceErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from string
		to   Stage
		want StageProblem
	}{
		{name: "refuses to release a release", from: "1.2.3", to: Release, want: ProblemAlreadyReleased},
		{name: "refuses to take a release back to alpha", from: "1.2.3", to: Alpha, want: ProblemReleaseIsFinal},
		{name: "refuses to take a release back to rc", from: "1.2.3", to: RC, want: ProblemReleaseIsFinal},
		{name: "refuses to go from rc back to alpha", from: "1.2.3-rc-001", to: Alpha, want: ProblemBackwards},
		{name: "refuses to go from beta back to alpha", from: "1.2.3-beta-005", to: Alpha, want: ProblemBackwards},
		{name: "refuses to go from rc back to beta", from: "1.2.3-rc-002", to: Beta, want: ProblemBackwards},
		{name: "refuses to go past the last index", from: "1.2.3-rc-999", to: RC, want: ProblemIndexExhausted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			from := mustParse(t, tt.from)

			got, err := from.Advance(tt.to)

			var serr *StageError
			if !errors.As(err, &serr) {
				t.Fatalf("%s.Advance(%s) = %v, %v, want a *StageError", tt.from, tt.to.name(), got, err)
			}

			if want := (StageError{Version: from, To: tt.to, Problem: tt.want}); *serr != want {
				t.Errorf("%s.Advance(%s) error = %#v, want %#v", tt.from, tt.to.name(), *serr, want)
			}
		})
	}
}

func TestAdvanceRejectsUnknownStages(t *testing.T) {
	t.Parallel()

	if _, err := mustParse(t, "1.2.3-rc-001").Advance(Stage(99)); err == nil {
		t.Error("Advance(Stage(99)) error = nil, want an error")
	}
}

func TestModifyErrorMessages(t *testing.T) {
	t.Parallel()

	v := Version{Major: 1, Minor: 2, Patch: 3}

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "describes an overflow",
			err:  &OverflowError{Version: v, Part: Minor},
			want: "bumping minor version of 1.2.3: number too large",
		},
		{
			name: "describes a stage problem",
			err:  &StageError{Version: v, To: Alpha, Problem: ProblemReleaseIsFinal},
			want: "moving 1.2.3 to alpha: a release can't go back to a stage",
		},
		{
			name: "describes a release problem",
			err:  &StageError{Version: v, To: Release, Problem: ProblemAlreadyReleased},
			want: "moving 1.2.3 to release: already released",
		},
		{
			name: "describes unknown values without panicking",
			err:  &StageError{Version: v, To: Stage(99), Problem: StageProblem(0)},
			want: "moving 1.2.3 to stage 99: problem 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}

	if got, want := Part(0).String(), "part 0"; got != want {
		t.Errorf("Part(0).String() = %q, want %q", got, want)
	}
}

func ExampleVersion_Bump() {
	v, _ := Parse("1.2.3")

	minor, _ := v.Bump(Minor, Release)
	alpha, _ := v.Bump(Minor, Alpha)

	fmt.Println(minor)
	fmt.Println(alpha)
	// Output:
	// 1.3.0
	// 1.3.0-alpha-001
}

func ExampleVersion_Advance() {
	v, _ := Parse("1.2.3-alpha-099")

	for _, to := range []Stage{Alpha, RC, Release} {
		next, err := v.Advance(to)
		fmt.Println(next, err)
	}

	_, err := v.Advance(Alpha)
	fmt.Println(err == nil)

	rc, _ := v.Advance(RC)
	_, err = rc.Advance(Alpha)
	fmt.Println(err)
	// Output:
	// 1.2.3-alpha-100 <nil>
	// 1.2.3-rc-001 <nil>
	// 1.2.3 <nil>
	// true
	// moving 1.2.3-rc-001 to alpha: can't go back to an earlier stage
}
