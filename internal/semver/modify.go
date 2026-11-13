package semver

import (
	"fmt"
	"math"
)

// Part identifies one of the numbers in MAJOR.MINOR.PATCH.
type Part int

// The parts of a version, from §Format.
const (
	Major Part = iota + 1
	Minor
	Patch
)

var partNames = map[Part]string{
	Major: "major",
	Minor: "minor",
	Patch: "patch",
}

// String returns the part's name in lowercase, such as "major".
func (p Part) String() string {
	if name, ok := partNames[p]; ok {
		return name
	}

	return fmt.Sprintf("part %d", int(p))
}

// OverflowError reports that a part of a version is already as large as it
// can be, so it can't be bumped.
type OverflowError struct {
	Version Version
	Part    Part
}

// Error implements error.
func (e *OverflowError) Error() string {
	return fmt.Sprintf("bumping %s version of %s: number too large", e.Part, e.Version)
}

// StageProblem identifies which rule in §Stage progression a stage change
// breaks.
type StageProblem int

// The ways a stage change can break §Stage progression. The zero value is
// not used.
const (
	// ProblemAlreadyReleased: the version is a release, and the change asked
	// to release it.
	ProblemAlreadyReleased StageProblem = iota + 1
	// ProblemReleaseIsFinal: the version is a release, and the change asked
	// for a pre-release stage. "Once a version is released, it can't go back
	// into any stage."
	ProblemReleaseIsFinal
	// ProblemBackwards: the change asked for a stage earlier than the
	// version's. "A version must not go back to an earlier stage."
	ProblemBackwards
	// ProblemIndexExhausted: the change asked for the version's own stage,
	// whose index is already MaxIndex. "Going past 999 is an error."
	ProblemIndexExhausted
)

var problemDescriptions = map[StageProblem]string{
	ProblemAlreadyReleased: "already released",
	ProblemReleaseIsFinal:  "a release can't go back to a stage",
	ProblemBackwards:       "can't go back to an earlier stage",
	ProblemIndexExhausted:  "stage index exhausted",
}

// String describes the problem briefly, for use in Go error strings.
func (p StageProblem) String() string {
	if d, ok := problemDescriptions[p]; ok {
		return d
	}

	return fmt.Sprintf("problem %d", int(p))
}

// StageError reports a stage change that §Stage progression doesn't allow.
type StageError struct {
	// Version is the version the change was applied to.
	Version Version
	// To is the stage that was asked for.
	To Stage
	// Problem is the rule the change breaks.
	Problem StageProblem
}

// Error implements error.
func (e *StageError) Error() string {
	return fmt.Sprintf("moving %s to %s: %s", e.Version, e.To.name(), e.Problem)
}

// Bump increases part by one and resets the parts after it to zero. The
// result has stage at index MinIndex, or no suffix if stage is Release.
//
// Bumping always starts a new MAJOR.MINOR.PATCH, so the result may have any
// stage (§Stage progression: "the first pre-release can be any stage"),
// whatever stage v had. Any suffix v had is dropped.
//
// It returns an *OverflowError if part is already math.MaxUint64.
func (v Version) Bump(part Part, stage Stage) (Version, error) {
	if !stage.valid() {
		return Version{}, fmt.Errorf("bumping %s: unknown stage %d", v, int(stage))
	}

	next := Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch}

	var n *uint64

	switch part {
	case Major:
		n, next.Minor, next.Patch = &next.Major, 0, 0
	case Minor:
		n, next.Patch = &next.Minor, 0
	case Patch:
		n = &next.Patch
	default:
		return Version{}, fmt.Errorf("bumping %s: unknown part %d", v, int(part))
	}

	if *n == math.MaxUint64 {
		return Version{}, &OverflowError{Version: v, Part: part}
	}

	*n++

	if stage != Release {
		next.Stage, next.Index = stage, MinIndex
	}

	return next, nil
}

// Advance moves v to stage to without changing MAJOR.MINOR.PATCH, following
// §Stage progression:
//
//   - to v's own stage increments the index;
//   - to a later stage starts that stage at index MinIndex;
//   - to Release removes the suffix from a pre-release.
//
// Anything else returns a *StageError: releasing a release, moving a release
// back into a stage, moving to an earlier stage, or going past MaxIndex.
func (v Version) Advance(to Stage) (Version, error) {
	if !to.valid() {
		return Version{}, fmt.Errorf("moving %s: unknown stage %d", v, int(to))
	}

	fail := func(p StageProblem) (Version, error) {
		return Version{}, &StageError{Version: v, To: to, Problem: p}
	}

	next := v

	switch {
	case v.Stage == Release && to == Release:
		return fail(ProblemAlreadyReleased)
	case v.Stage == Release:
		return fail(ProblemReleaseIsFinal)
	case to.rank() < v.Stage.rank():
		return fail(ProblemBackwards)
	case to == Release:
		next.Stage, next.Index = Release, 0
	case to == v.Stage && v.Index >= MaxIndex:
		return fail(ProblemIndexExhausted)
	case to == v.Stage:
		next.Index++
	default:
		next.Stage, next.Index = to, MinIndex
	}

	return next, nil
}

// Regress is like Advance, except that a pre-release may also go back to an
// earlier pre-release stage. That breaks the §Stage progression rule that
// stages only move forward, as §Going back a stage allows when asked for
// explicitly. A release still can't go back into any stage.
//
// Going back, the result is at the index after the highest one that taken
// already has for v's MAJOR.MINOR.PATCH at stage to, so that it doesn't repeat
// a version that has been used. If taken has none, the index is MinIndex.
// taken is a list such as git tag names; invalid entries are skipped. For
// example, from 1.2.3-rc-001 to Alpha:
//
//   - with no taken versions, the result is 1.2.3-alpha-001;
//   - with 1.2.3-alpha-001 and 1.2.3-alpha-002 taken, it is 1.2.3-alpha-003.
//
// It returns a *StageError for the same problems as Advance, and if the
// highest taken index is already MaxIndex.
func (v Version) Regress(to Stage, taken []string) (Version, error) {
	if !to.valid() {
		return Version{}, fmt.Errorf("moving %s back: unknown stage %d", v, int(to))
	}

	if v.Stage == Release || to == Release || to.rank() >= v.Stage.rank() {
		return v.Advance(to)
	}

	used, ok := latest(taken, func(t Version) bool {
		return t.Major == v.Major && t.Minor == v.Minor && t.Patch == v.Patch && t.Stage == to
	})
	if !ok {
		next := v
		next.Stage, next.Index = to, MinIndex

		return next, nil
	}

	// Within one stage, Advance increments the index, or reports that it
	// can't.
	return used.Advance(to)
}
