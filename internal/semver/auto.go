package semver

// AutoPlan describes the version Auto chose, and why.
type AutoPlan struct {
	// Next is the new version.
	Next Version
	// Part is the bump that was applied, after the rule for versions before
	// 1.0.0.
	Part Part
	// BeforeStable is true when a major bump became a minor one, because the
	// version is before 1.0.0.
	BeforeStable bool
	// Pending is true when the latest version was a pre-release whose own
	// bump already covered Part, so Next carries on from it rather than
	// bumping again.
	Pending bool
}

// Auto works out the next version when a set of changes needs part bumped,
// such as from Conventional Commits. latest is the latest version, and release
// is the latest release before it (the zero Version if there is none). The
// result has stage at index MinIndex, or no suffix for Release, as with Bump.
//
// Before 1.0.0, a major bump becomes a minor one: SemVer §4 says anything may
// change in 0.y.z, and going to 1.0.0 should be a deliberate choice.
//
// When latest is a pre-release, its version is already a bump over release.
// If that bump is at least as big as part, the changes fit in it, so the
// result carries on from latest: the next index of stage, or the release
// itself for Release. Otherwise release is bumped by part. For example, with
// release 1.2.3 and latest 1.3.0-rc-002:
//
//   - a fix, with stage RC, gives 1.3.0-rc-003;
//   - a fix, with no stage, gives 1.3.0;
//   - a breaking change, with stage RC, gives 2.0.0-rc-001.
func Auto(latest, release Version, part Part, stage Stage) (AutoPlan, error) {
	base := latest
	if !latest.IsRelease() {
		base = release
	}

	plan := AutoPlan{Part: part}

	if base.Major == 0 && part == Major {
		plan.Part, plan.BeforeStable = Minor, true
	}

	var err error

	switch {
	case latest.IsRelease():
		plan.Next, err = latest.Bump(plan.Part, stage)
	case covers(pendingPart(latest, release), plan.Part):
		// Advance(Release) releases the pre-release.
		plan.Pending = true
		plan.Next, err = latest.Advance(stage)
	default:
		plan.Next, err = release.Bump(plan.Part, stage)
	}

	if err != nil {
		return AutoPlan{}, err
	}

	return plan, nil
}

// pendingPart returns the part that the pre-release v bumps over release.
func pendingPart(v, release Version) Part {
	switch {
	case v.Major != release.Major:
		return Major
	case v.Minor != release.Minor:
		return Minor
	default:
		return Patch
	}
}

// covers reports whether a bump of have is at least as big as a bump of want.
// Major is the biggest bump and Patch the smallest.
func covers(have, want Part) bool {
	return have <= want
}
