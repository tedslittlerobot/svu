package buildinfo

import (
	"strings"
	"time"
)

// The commit and its date, filled in by git archive. .gitattributes marks this
// file export-subst, so git archive, and GitHub's source tarballs, which
// Homebrew builds from, replace each placeholder. Both are fixed for a commit,
// so a tarball's checksum never changes.
//
// The version is deliberately not filled in this way: git describe can name a
// different tag once a later tag points at the same commit, which would change
// a published tarball. Builds from a tarball set it with -ldflags instead.
//
// In a git checkout the placeholders stay as they are, and archived ignores
// them.
const (
	archiveCommit = "$Format:%H$"
	archiveDate   = "$Format:%cI$"
)

// archive is the build information that git archive filled in.
type archive struct {
	commit string
	date   string
}

// archived returns what git archive filled into commit and date, as from
// archiveCommit and archiveDate. A placeholder that wasn't filled in gives "".
// The date is normalised to UTC in RFC 3339, as the other sources give it.
func archived(commit, date string) archive {
	var a archive

	if !strings.HasPrefix(commit, "$Format:") {
		a.commit = commit
	}

	if t, err := time.Parse(time.RFC3339, date); err == nil {
		a.date = t.UTC().Format(time.RFC3339)
	}

	return a
}
