package semver

import (
	"errors"
	"fmt"
	"testing"
)

func TestAuto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		latest  string
		release string // "" means no release yet.
		part    Part
		stage   Stage
		want    string
		wantPre bool // BeforeStable
		wantPen bool // Pending
	}{
		// From a release.
		{name: "bumps a release by the part", latest: "1.2.3", part: Minor, want: "1.3.0"},
		{name: "starts a stage on a bumped release", latest: "1.2.3", part: Patch, stage: RC, want: "1.2.4-rc-001"},
		{name: "makes a major bump", latest: "1.2.3", part: Major, want: "2.0.0"},

		// Before 1.0.0.
		{name: "turns a major bump into minor before 1.0.0", latest: "0.4.2", part: Major, want: "0.5.0", wantPre: true},
		{name: "leaves a minor bump alone before 1.0.0", latest: "0.4.2", part: Minor, want: "0.5.0"},
		{name: "leaves a patch bump alone before 1.0.0", latest: "0.4.2", part: Patch, want: "0.4.3"},

		// From a pre-release whose bump covers the changes.
		{name: "continues a pre-release for a smaller bump", latest: "1.3.0-rc-002", release: "1.2.3", part: Patch, stage: RC, want: "1.3.0-rc-003", wantPen: true},
		{name: "continues a pre-release for the same bump", latest: "1.3.0-rc-002", release: "1.2.3", part: Minor, stage: RC, want: "1.3.0-rc-003", wantPen: true},
		{name: "releases a pending pre-release with no stage", latest: "1.3.0-rc-002", release: "1.2.3", part: Patch, want: "1.3.0", wantPen: true},
		{name: "moves a pending pre-release to a later stage", latest: "1.3.0-beta-004", release: "1.2.3", part: Patch, stage: RC, want: "1.3.0-rc-001", wantPen: true},
		{name: "continues a major pre-release for a feature", latest: "2.0.0-rc-001", release: "1.2.3", part: Minor, stage: RC, want: "2.0.0-rc-002", wantPen: true},
		{name: "continues a patch pre-release for a patch", latest: "1.2.4-alpha-001", release: "1.2.3", part: Patch, stage: Alpha, want: "1.2.4-alpha-002", wantPen: true},

		// From a pre-release whose bump is too small.
		{name: "bumps the release when a pre-release is too small", latest: "1.3.0-rc-002", release: "1.2.3", part: Major, stage: RC, want: "2.0.0-rc-001"},
		{name: "bumps a patch pre-release's release to minor", latest: "1.2.4-rc-001", release: "1.2.3", part: Minor, want: "1.3.0"},

		// Pre-releases before 1.0.0, and with no release yet.
		{name: "keeps a breaking change in a 0.x pre-release", latest: "0.3.0-rc-001", release: "0.2.0", part: Major, stage: RC, want: "0.3.0-rc-002", wantPre: true, wantPen: true},
		{name: "continues a first pre-release with no release", latest: "0.1.0-alpha-001", part: Minor, stage: Alpha, want: "0.1.0-alpha-002", wantPen: true},
		{name: "releases a first pre-release with no release", latest: "0.1.0-rc-003", part: Patch, want: "0.1.0", wantPen: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			release := Version{}
			if tt.release != "" {
				release = mustParse(t, tt.release)
			}

			got, err := Auto(mustParse(t, tt.latest), release, tt.part, tt.stage)
			if err != nil {
				t.Fatalf("Auto() error = %v, want nil", err)
			}

			if got.Next.String() != tt.want || got.BeforeStable != tt.wantPre || got.Pending != tt.wantPen {
				t.Errorf("Auto(%s, %s, %s, %s) = %+v, want %s (before stable %t, pending %t)",
					tt.latest, release, tt.part, tt.stage.name(), got, tt.want, tt.wantPre, tt.wantPen)
			}
		})
	}
}

func TestAutoErrors(t *testing.T) {
	t.Parallel()

	// A pending rc can't go back to alpha.
	_, err := Auto(mustParse(t, "1.3.0-rc-002"), mustParse(t, "1.2.3"), Patch, Alpha)

	var serr *StageError
	if !errors.As(err, &serr) || serr.Problem != ProblemBackwards {
		t.Errorf("Auto() error = %v, want a backwards StageError", err)
	}

	// A number can't overflow.
	_, err = Auto(Version{Major: 1, Patch: 18446744073709551615}, Version{}, Patch, Release)

	var oerr *OverflowError
	if !errors.As(err, &oerr) {
		t.Errorf("Auto() error = %v, want an OverflowError", err)
	}
}

func ExampleAuto() {
	release, _ := Parse("1.2.3")
	latest, _ := Parse("1.3.0-rc-002")

	// Only fixes since the rc: carry on with it.
	plan, _ := Auto(latest, release, Patch, RC)
	fmt.Println(plan.Next, plan.Pending)

	// A breaking change: 1.3.0 isn't enough.
	plan, _ = Auto(latest, release, Major, RC)
	fmt.Println(plan.Next, plan.Pending)
	// Output:
	// 1.3.0-rc-003 true
	// 2.0.0-rc-001 false
}
