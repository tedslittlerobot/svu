// Package buildinfo reports version metadata for the running semvu binary.
//
// Release builds inject values at link time (see .goreleaser.yaml). Builds
// made with plain `go build` or `go install` fall back to the metadata the Go
// toolchain embeds. Builds from a source tarball, which has no git metadata,
// fall back to the commit and date that git archive filled in (archive.go), so
// every binary can describe where it came from.
package buildinfo

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// Link-time overrides, set with:
//
//	-ldflags "-X github.com/tedslittlerobot/semvu/internal/buildinfo.version=1.2.3"
//
// They are unexported so that nothing else in the program can change them.
var (
	version string
	commit  string
	date    string
)

// Fallback values used when neither the linker nor the toolchain supplies one.
const (
	DevVersion = "dev"
	Unknown    = "unknown"
)

// Info describes the build of the running binary.
type Info struct {
	Version   string // Semantic version, without a leading "v".
	Commit    string // Full VCS revision.
	Date      string // Commit or build timestamp, RFC 3339.
	GoVersion string // Toolchain that compiled the binary.
	Platform  string // GOOS/GOARCH.
}

// Get returns the build information for the running binary.
func Get() Info {
	return resolve(version, commit, date, debug.ReadBuildInfo, archived(archiveCommit, archiveDate))
}

// resolve merges link-time values, toolchain metadata and what git archive
// filled in, in that order of precedence. Anything still missing is filled
// with a placeholder. The reader and archive are injected so tests can
// simulate any build.
func resolve(ver, rev, built string, read func() (*debug.BuildInfo, bool), arc archive) Info {
	info := Info{
		Version:   ver,
		Commit:    rev,
		Date:      built,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}

	if bi, ok := read(); ok && bi != nil {
		if info.Version == "" && bi.Main.Version != "(devel)" {
			info.Version = bi.Main.Version
		}

		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				info.Commit = firstNonEmpty(info.Commit, s.Value)
			case "vcs.time":
				info.Date = firstNonEmpty(info.Date, s.Value)
			}
		}
	}

	info.Version = firstNonEmpty(strings.TrimPrefix(info.Version, "v"), DevVersion)
	info.Commit = firstNonEmpty(info.Commit, arc.commit, Unknown)
	info.Date = firstNonEmpty(info.Date, arc.date, Unknown)

	return info
}

// String renders the build information on a single line.
func (i Info) String() string {
	return fmt.Sprintf("semvu %s (commit %s, built %s, %s, %s)",
		i.Version, i.Commit, i.Date, i.GoVersion, i.Platform)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}
