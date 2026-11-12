package cli

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/pflag"

	"github.com/tedslittlerobot/semvu/internal/semver"
)

// modifier is one flag that changes the version, such as --minor or --rc.
type modifier[T any] struct {
	flag  string // Without the leading "--".
	usage string
	value T
	set   bool
}

// The modifier flags, in the order they appear in help and messages. The
// stage modifiers are in §Stages order, which hints rely on.
func newVersionModifiers() []*modifier[semver.Part] {
	return []*modifier[semver.Part]{
		{flag: "major", value: semver.Major, usage: "Bump MAJOR, and reset MINOR and PATCH to 0"},
		{flag: "minor", value: semver.Minor, usage: "Bump MINOR, and reset PATCH to 0"},
		{flag: "patch", value: semver.Patch, usage: "Bump PATCH"},
		{flag: hotfixFlag, value: semver.Patch, usage: "Bump PATCH of the latest release, skipping pre-releases"},
		{flag: autoFlag, usage: "Work out the bump from the Conventional Commits since the latest tag"},
	}
}

// hotfixFlag is the version modifier that patches the latest release. From
// git tags, semvu modify then reads the latest release rather than the latest
// version, so a release candidate in progress doesn't get in the way.
const hotfixFlag = "hotfix"

func newStageModifiers() []*modifier[semver.Stage] {
	return []*modifier[semver.Stage]{
		{flag: "alpha", value: semver.Alpha, usage: "Move to alpha-001, or to the next alpha"},
		{flag: "beta", value: semver.Beta, usage: "Move to beta-001, or to the next beta"},
		{flag: "rc", value: semver.RC, usage: "Move to rc-001, or to the next rc"},
		{flag: "release", value: semver.Release, usage: "Remove the stage from a pre-release"},
	}
}

// modifierNames returns the names of all the modifier flags, without "--".
func modifierNames() []string {
	var names []string

	for _, m := range newVersionModifiers() {
		names = append(names, m.flag)
	}

	for _, m := range newStageModifiers() {
		names = append(names, m.flag)
	}

	return names
}

// modifiers holds the modifier flags for one run of semvu.
type modifiers struct {
	versions []*modifier[semver.Part]
	stages   []*modifier[semver.Stage]
}

func newModifiers(flags *pflag.FlagSet) *modifiers {
	m := &modifiers{versions: newVersionModifiers(), stages: newStageModifiers()}

	for _, v := range m.versions {
		flags.BoolVar(&v.set, v.flag, false, v.usage)
		setHelpGroup(flags, v.flag, groupVersion)
	}

	for _, s := range m.stages {
		flags.BoolVar(&s.set, s.flag, false, s.usage)
		setHelpGroup(flags, s.flag, groupStage)
	}

	return m
}

// validate checks that at most one modifier from each group was given.
func (m *modifiers) validate() error {
	if err := atMostOne("version", m.versions); err != nil {
		return err
	}

	return atMostOne("stage", m.stages)
}

func atMostOne[T any](group string, mods []*modifier[T]) error {
	var all, set []string

	for _, mod := range mods {
		all = append(all, "--"+mod.flag)

		if mod.set {
			set = append(set, "--"+mod.flag)
		}
	}

	if len(set) <= 1 {
		return nil
	}

	return &userError{
		code:     ExitUsage,
		headline: join(set, "and") + " can't be used together.",
		detail:   fmt.Sprintf("Give at most one %s modifier: %s.", group, join(all, "or")),
	}
}

// chosen returns the value of the modifier that was set, if any.
func chosen[T any](mods []*modifier[T]) (T, bool) {
	for _, mod := range mods {
		if mod.set {
			return mod.value, true
		}
	}

	var zero T

	return zero, false
}

// hotfix reports whether --hotfix was given.
func (m *modifiers) hotfix() bool {
	return slices.ContainsFunc(m.versions, func(mod *modifier[semver.Part]) bool {
		return mod.flag == hotfixFlag && mod.set
	})
}

// auto reports whether --auto was given.
func (m *modifiers) auto() bool {
	return slices.ContainsFunc(m.versions, func(mod *modifier[semver.Part]) bool {
		return mod.flag == autoFlag && mod.set
	})
}

// stage returns the stage modifier given, or Release if none was.
func (m *modifiers) stage() semver.Stage {
	stage, _ := chosen(m.stages)

	return stage
}

// apply changes v as the modifiers ask. It must only be called after
// validate has passed.
func (m *modifiers) apply(v semver.Version) (semver.Version, error) {
	part, bump := chosen(m.versions)
	stage, move := chosen(m.stages)

	if m.hotfix() && !v.IsRelease() {
		return semver.Version{}, &userError{
			code:     ExitModification,
			headline: fmt.Sprintf("%q is a pre-release, so it can't be hotfixed.", v.String()),
			detail:   "--hotfix patches a released version.",
			hint:     "Give the release to fix, as in `semvu modify 1.2.3 --hotfix`, or use `--patch`.",
		}
	}

	var (
		next semver.Version
		err  error
	)

	switch {
	case bump:
		// With no stage modifier, stage is Release: any suffix is removed.
		next, err = v.Bump(part, stage)
	case move:
		next, err = v.Advance(stage)
	default:
		return v, nil
	}

	if err != nil {
		return semver.Version{}, m.modifyError(err)
	}

	return next, nil
}

func (m *modifiers) modifyError(err error) *userError {
	var (
		serr *semver.StageError
		oerr *semver.OverflowError
	)

	switch {
	case errors.As(err, &serr):
		return m.stageError(serr)
	case errors.As(err, &oerr):
		return &userError{
			code:     ExitModification,
			headline: fmt.Sprintf("The %s version of %q can't be bumped.", oerr.Part, oerr.Version),
			detail:   ruleDetails[semver.RuleNumberTooLarge],
			err:      err,
		}
	default:
		return &userError{code: ExitInternal, headline: "semvu could not modify the version.", hint: reportHint, err: err}
	}
}

const newVersionHint = "add `--major`, `--minor` or `--patch` to start a new version"

// hotfixReleaseFlag names --hotfix where tag errors mention the flag that
// asked for releases only.
const hotfixReleaseFlag = "--" + hotfixFlag

func (m *modifiers) stageError(err *semver.StageError) *userError {
	v := err.Version.String()

	switch err.Problem {
	case semver.ProblemAlreadyReleased:
		return &userError{
			code:     ExitModification,
			headline: fmt.Sprintf("%q has already been released.", v),
			detail:   "--release removes the stage from a pre-release, such as 1.2.3-rc-001.",
			hint:     "To start a new version, use `--major`, `--minor` or `--patch`.",
			err:      err,
		}
	case semver.ProblemReleaseIsFinal:
		return &userError{
			code:     ExitModification,
			headline: fmt.Sprintf("%q has been released, so it can't go back to %s.", v, err.To),
			detail:   "Once a version is released, it can't go back into any stage.",
			hint:     fmt.Sprintf("Add `--major`, `--minor` or `--patch` to start a new version at %s, as in `--patch --%s`.", err.To, err.To),
			err:      err,
		}
	case semver.ProblemBackwards:
		return &userError{
			code:     ExitModification,
			headline: fmt.Sprintf("%q can't go back from %s to %s.", v, err.Version.Stage, err.To),
			detail:   "Stages only move forward: alpha, then beta, then rc, then the release.",
			hint:     fmt.Sprintf("Use %s, or %s.", join(m.stageFlags(err.Version.Stage, true), "or"), newVersionHint),
			err:      err,
		}
	case semver.ProblemIndexExhausted:
		return &userError{
			code:     ExitModification,
			headline: fmt.Sprintf("%q is the last %s allowed.", v, err.To),
			detail:   fmt.Sprintf("A stage allows at most %d iterations.", semver.MaxIndex),
			hint:     fmt.Sprintf("Use %s, or %s.", join(m.stageFlags(err.Version.Stage, false), "or"), newVersionHint),
			err:      err,
		}
	default:
		return &userError{code: ExitInternal, headline: "semvu could not modify the version.", hint: reportHint, err: err}
	}
}

// stageFlags returns the stage modifier flags after from, and from's own flag
// too if inclusive is true.
func (m *modifiers) stageFlags(from semver.Stage, inclusive bool) []string {
	var flags []string

	found := false

	for _, s := range m.stages {
		if found || (inclusive && s.value == from) {
			flags = append(flags, "`--"+s.flag+"`")
		}

		found = found || s.value == from
	}

	return flags
}

// join lists items in English, as in "a, b and c".
func join(items []string, conjunction string) string {
	if len(items) <= 1 {
		return strings.Join(items, "")
	}

	return strings.Join(items[:len(items)-1], ", ") + " " + conjunction + " " + items[len(items)-1]
}
