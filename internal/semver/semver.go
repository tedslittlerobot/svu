// Package semver parses, compares and formats versions in svu's subset of
// Semantic Versioning 2.0.0.
//
// The accepted format, ordering and stage rules are defined in
// conventions/semver.md, which is the source of truth for this package.
// Section references in comments (such as "§Format") point into that document.
package semver

import (
	"cmp"
	"strconv"
)

// Stage is how far along a version is (§Stages). The zero value is Release,
// so the zero Version is the release version 0.0.0.
type Stage int

// The stages, from §Stages. Their declaration order is not their precedence:
// use Compare, which ranks Release above every pre-release stage.
const (
	// Release marks a release version, which has no suffix.
	Release Stage = iota
	// Alpha is an early, incomplete and unstable pre-release.
	Alpha
	// Beta is a feature-complete pre-release that is still being tested.
	Beta
	// RC is a release candidate.
	RC
)

// stageNames maps each pre-release stage to how it is written in a version.
var stageNames = map[Stage]string{
	Alpha: "alpha",
	Beta:  "beta",
	RC:    "rc",
}

// String returns the stage as written in a version, or "" for Release.
func (s Stage) String() string {
	return stageNames[s]
}

// name returns the stage's name, including "release" for Release, for use in
// Go error strings.
func (s Stage) name() string {
	switch {
	case s == Release:
		return "release"
	case s.valid():
		return s.String()
	default:
		return "stage " + strconv.Itoa(int(s))
	}
}

// valid reports whether s is one of the Stage constants.
func (s Stage) valid() bool {
	_, ok := stageNames[s]

	return ok || s == Release
}

// rank orders stages by precedence (§Precedence): every pre-release stage
// ranks below Release.
func (s Stage) rank() int {
	if s == Release {
		return len(stageNames) + 1
	}

	return int(s)
}

// Index bounds, from §Format.
const (
	MinIndex = 1
	MaxIndex = 999
)

// Version is a parsed svu version. A release version has Stage Release and
// Index 0; a pre-release version has a pre-release Stage and an Index from
// MinIndex to MaxIndex.
//
// Versions don't remember whether they were written with a "v" prefix: the
// prefix is ignored for precedence (§Precedence), and output decides whether
// to add one.
type Version struct {
	Major uint64
	Minor uint64
	Patch uint64
	Stage Stage
	Index int
}

// String formats the version without a "v" prefix, such as "1.2.3" or
// "1.2.3-rc-001".
func (v Version) String() string {
	s := strconv.FormatUint(v.Major, 10) + "." +
		strconv.FormatUint(v.Minor, 10) + "." +
		strconv.FormatUint(v.Patch, 10)

	if v.Stage == Release {
		return s
	}

	// §Format: the index is always exactly three digits, zero-padded.
	index := strconv.Itoa(v.Index)
	for len(index) < 3 {
		index = "0" + index
	}

	return s + "-" + v.Stage.String() + "-" + index
}

// Compare returns -1 if a precedes b, 0 if they are equal, and +1 if a
// follows b, using the order in §Precedence.
func Compare(a, b Version) int {
	return cmp.Or(
		cmp.Compare(a.Major, b.Major),
		cmp.Compare(a.Minor, b.Minor),
		cmp.Compare(a.Patch, b.Patch),
		cmp.Compare(a.Stage.rank(), b.Stage.rank()),
		cmp.Compare(a.Index, b.Index),
	)
}

// Latest parses each candidate, skips any that aren't valid versions, and
// returns the one with the highest precedence. It reports false if no
// candidate is valid.
//
// It is meant for lists such as git tag names, where only some entries are
// versions and a leading "v" is common.
func Latest(candidates []string) (Version, bool) {
	return latest(candidates, func(Version) bool { return true })
}

// LatestRelease is like Latest, but only considers release versions: it skips
// pre-releases as well as invalid candidates.
func LatestRelease(candidates []string) (Version, bool) {
	return latest(candidates, Version.IsRelease)
}

// IsRelease reports whether v is a release version, with no suffix
// (§Release versions).
func (v Version) IsRelease() bool {
	return v.Stage == Release
}

func latest(candidates []string, keep func(Version) bool) (Version, bool) {
	var (
		best  Version
		found bool
	)

	for _, c := range candidates {
		v, err := Parse(c)
		if err != nil || !keep(v) {
			continue
		}

		if !found || Compare(v, best) > 0 {
			best, found = v, true
		}
	}

	return best, found
}
