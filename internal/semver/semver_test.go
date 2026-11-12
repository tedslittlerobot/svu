package semver

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestVersionString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version Version
		want    string
	}{
		{name: "formats the zero version", version: Version{}, want: "0.0.0"},
		{name: "formats a release", version: Version{Major: 1, Minor: 2, Patch: 3}, want: "1.2.3"},
		{
			name:    "pads a one-digit index",
			version: Version{Major: 1, Minor: 2, Patch: 3, Stage: Alpha, Index: 1},
			want:    "1.2.3-alpha-001",
		},
		{
			name:    "pads a two-digit index",
			version: Version{Major: 1, Stage: Beta, Index: 42},
			want:    "1.0.0-beta-042",
		},
		{
			name:    "formats a three-digit index",
			version: Version{Major: 10, Minor: 20, Patch: 30, Stage: RC, Index: 999},
			want:    "10.20.30-rc-999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.version.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCompare(t *testing.T) {
	t.Parallel()

	// §Precedence: each version precedes the next.
	ordered := []string{
		"0.0.0",
		"0.0.1",
		"0.1.0",
		"1.0.0-alpha-001",
		"1.2.3-alpha-001",
		"1.2.3-alpha-002",
		"1.2.3-alpha-010",
		"1.2.3-beta-001",
		"1.2.3-rc-001",
		"1.2.3",
		"1.2.4-alpha-001",
		"1.2.10",
		"1.10.0",
		"2.0.0",
		"10.0.0",
	}

	for i, a := range ordered {
		for j, b := range ordered {
			want := cmp.Compare(i, j)
			if got := Compare(mustParse(t, a), mustParse(t, b)); got != want {
				t.Errorf("Compare(%s, %s) = %d, want %d", a, b, got, want)
			}
		}
	}
}

func TestCompareIgnoresPrefix(t *testing.T) {
	t.Parallel()

	if got := Compare(mustParse(t, "v1.2.3"), mustParse(t, "1.2.3")); got != 0 {
		t.Errorf("Compare(v1.2.3, 1.2.3) = %d, want 0", got)
	}
}

// TestCompareMatchesSemVer checks the invariant in §Precedence: for any two
// svu versions, svu's ordering is the same as plain SemVer 2.0.0 ordering.
func TestCompareMatchesSemVer(t *testing.T) {
	t.Parallel()

	var versions []Version

	for _, major := range []uint64{0, 1, 10} {
		for _, minor := range []uint64{0, 2, 10} {
			for _, patch := range []uint64{0, 9, 10} {
				base := Version{Major: major, Minor: minor, Patch: patch}
				versions = append(versions, base)

				for _, stage := range []Stage{Alpha, Beta, RC} {
					for _, index := range []int{1, 2, 9, 10, 99, 100, 999} {
						v := base
						v.Stage, v.Index = stage, index
						versions = append(versions, v)
					}
				}
			}
		}
	}

	for _, a := range versions {
		for _, b := range versions {
			if got, want := Compare(a, b), semverCompare(a.String(), b.String()); got != want {
				t.Fatalf("Compare(%s, %s) = %d, but SemVer orders them %d", a, b, got, want)
			}
		}
	}
}

// semverCompare is a reference implementation of SemVer 2.0.0 §11 precedence
// for valid SemVer strings without build metadata. It knows nothing about
// svu's stages, so it is an independent check on Compare.
func semverCompare(a, b string) int {
	aCore, aPre, _ := strings.Cut(a, "-")
	bCore, bPre, _ := strings.Cut(b, "-")

	// §11.2: compare major, minor and patch numerically.
	if c := slices.CompareFunc(strings.Split(aCore, "."), strings.Split(bCore, "."), compareNumeric); c != 0 {
		return c
	}

	// §11.3: a version without a pre-release has higher precedence.
	switch {
	case aPre == bPre:
		return 0
	case aPre == "":
		return 1
	case bPre == "":
		return -1
	}

	// §11.4: compare dot-separated identifiers from left to right; a larger
	// set of identifiers wins if all the preceding ones are equal.
	return slices.CompareFunc(strings.Split(aPre, "."), strings.Split(bPre, "."), compareIdentifier)
}

func compareNumeric(a, b string) int {
	x, _ := strconv.ParseUint(a, 10, 64)
	y, _ := strconv.ParseUint(b, 10, 64)

	return cmp.Compare(x, y)
}

// compareIdentifier implements SemVer §11.4.1 to §11.4.3.
func compareIdentifier(a, b string) int {
	_, errA := strconv.ParseUint(a, 10, 64)
	_, errB := strconv.ParseUint(b, 10, 64)

	switch {
	case errA == nil && errB == nil:
		return compareNumeric(a, b)
	case errA == nil:
		return -1
	case errB == nil:
		return 1
	default:
		return strings.Compare(a, b) // ASCII order.
	}
}

func TestSemverCompareReference(t *testing.T) {
	t.Parallel()

	// The example ordering from SemVer §11.4, to check the reference itself.
	ordered := []string{
		"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta",
		"1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0",
	}

	for i, a := range ordered {
		for j, b := range ordered {
			if got, want := semverCompare(a, b), cmp.Compare(i, j); got != want {
				t.Errorf("semverCompare(%s, %s) = %d, want %d", a, b, got, want)
			}
		}
	}
}

func TestLatest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		candidates []string
		want       string
		wantFound  bool
	}{
		{name: "finds nothing in an empty list", candidates: nil, wantFound: false},
		{
			name:       "finds nothing when no candidate is valid",
			candidates: []string{"latest", "release-1", "1.2", "V1.2.3"},
			wantFound:  false,
		},
		{
			name:       "picks the only valid candidate",
			candidates: []string{"nightly", "v0.1.0", "1.2.3-alpha-1"},
			want:       "0.1.0",
			wantFound:  true,
		},
		{
			name:       "picks the highest version, not the last or longest",
			candidates: []string{"v1.10.0", "v1.9.0", "v1.2.0", "v1.10.0-rc-002"},
			want:       "1.10.0",
			wantFound:  true,
		},
		{
			name:       "ranks a release above its pre-releases",
			candidates: []string{"1.2.3-rc-005", "1.2.3", "1.2.3-beta-009"},
			want:       "1.2.3",
			wantFound:  true,
		},
		{
			name:       "ranks a pre-release of a later version above an earlier release",
			candidates: []string{"v1.2.3", "v1.3.0-alpha-001"},
			want:       "1.3.0-alpha-001",
			wantFound:  true,
		},
		{
			name:       "treats prefixed and unprefixed tags alike",
			candidates: []string{"v2.0.0", "1.9.9", "2.0.0"},
			want:       "2.0.0",
			wantFound:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, found := Latest(tt.candidates)
			if found != tt.wantFound {
				t.Fatalf("Latest(%q) found = %t, want %t", tt.candidates, found, tt.wantFound)
			}

			if found && got.String() != tt.want {
				t.Errorf("Latest(%q) = %s, want %s", tt.candidates, got, tt.want)
			}
		})
	}
}

func mustParse(t *testing.T, s string) Version {
	t.Helper()

	v, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse(%q) error = %v", s, err)
	}

	return v
}

func ExampleParse() {
	v, err := Parse("v1.2.3-rc-002")
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Println(v.Major, v.Minor, v.Patch, v.Stage, v.Index)
	fmt.Println(v)
	// Output:
	// 1 2 3 rc 2
	// 1.2.3-rc-002
}

func ExampleParse_invalid() {
	_, err := Parse("1.2.3-alpha-1")
	fmt.Println(err)
	// Output:
	// parsing version "1.2.3-alpha-1": index not three digits from 001 to 999 at byte 12
}

func ExampleLatest() {
	tags := []string{"v1.9.0", "nightly", "v1.10.0-rc-001", "v1.10.0"}

	v, ok := Latest(tags)
	fmt.Println(v, ok)
	// Output:
	// 1.10.0 true
}
