// Package commits reads commit messages written to the Conventional Commits
// 1.0.0 specification (https://www.conventionalcommits.org/en/v1.0.0/), and
// works out how far they move a version.
//
// The bump rules are:
//
//   - a breaking change, marked with "!" before the colon or a
//     "BREAKING CHANGE:" footer, needs a major bump, whatever its type;
//   - "feat" needs a minor bump;
//   - "fix" and "perf" need a patch bump;
//   - any other type, such as "docs" or "chore", needs no bump on its own.
//
// Messages that don't follow the specification are counted, but don't bump.
package commits

import (
	"strings"
)

// Bump is how far a change moves a version. Larger values are bigger bumps.
type Bump int

// The bumps, smallest first.
const (
	// None means no new version is needed.
	None Bump = iota
	// Patch means PATCH goes up.
	Patch
	// Minor means MINOR goes up.
	Minor
	// Major means MAJOR goes up.
	Major
)

var bumpNames = map[Bump]string{
	None:  "no",
	Patch: "patch",
	Minor: "minor",
	Major: "major",
}

// String returns the bump's name, such as "minor".
func (b Bump) String() string {
	if name, ok := bumpNames[b]; ok {
		return name
	}

	return "unknown"
}

// Message is one commit message, as Parse reads it.
type Message struct {
	// Conventional is true if the message follows the specification. The
	// other fields are only set when it is.
	Conventional bool
	// Type is the commit type, in lowercase, such as "feat".
	Type string
	// Scope is the optional scope, such as "parser" in "fix(parser): …".
	Scope string
	// Breaking is true for a breaking change.
	Breaking bool
	// Description is the summary after the colon.
	Description string
}

// Parse reads a commit message. A message that doesn't follow Conventional
// Commits gives a Message with Conventional false.
func Parse(message string) Message {
	header, body, _ := strings.Cut(strings.TrimSpace(message), "\n")

	// type(scope)!: description
	prefix, description, ok := strings.Cut(header, ": ")
	if !ok || strings.TrimSpace(description) == "" {
		return Message{}
	}

	m := Message{Description: strings.TrimSpace(description)}

	if rest, bang := strings.CutSuffix(prefix, "!"); bang {
		m.Breaking = true
		prefix = rest
	}

	if open := strings.IndexByte(prefix, '('); open >= 0 {
		scope, closed := strings.CutSuffix(prefix[open+1:], ")")
		if !closed || scope == "" || strings.ContainsAny(scope, "()") {
			return Message{}
		}

		m.Scope = scope
		prefix = prefix[:open]
	}

	if !isType(prefix) {
		return Message{}
	}

	m.Conventional = true
	m.Type = strings.ToLower(prefix)
	m.Breaking = m.Breaking || hasBreakingFooter(body)

	return m
}

// isType reports whether s is a valid commit type: one or more letters.
func isType(s string) bool {
	if s == "" {
		return false
	}

	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
			return false
		}
	}

	return true
}

// hasBreakingFooter reports whether the body has a breaking change footer.
// The token must be in uppercase (specification, rule 16).
func hasBreakingFooter(body string) bool {
	for line := range strings.SplitSeq(body, "\n") {
		if strings.HasPrefix(line, "BREAKING CHANGE: ") || strings.HasPrefix(line, "BREAKING-CHANGE: ") {
			return true
		}
	}

	return false
}

// Bump returns the bump the message needs on its own.
func (m Message) Bump() Bump {
	switch {
	case !m.Conventional:
		return None
	case m.Breaking:
		return Major
	case m.Type == "feat":
		return Minor
	case m.Type == "fix" || m.Type == "perf":
		return Patch
	default:
		return None
	}
}

// Summary counts a set of commit messages. Each message is counted once:
// as a breaking change, by its type, or as unconventional.
type Summary struct {
	// Commits is how many messages there were.
	Commits int
	// Breaking is how many were breaking changes, of any type.
	Breaking int
	// Types counts the conventional, non-breaking messages by type.
	Types map[string]int
	// Unconventional is how many didn't follow the specification.
	Unconventional int
	// Bump is the biggest bump any of them needs.
	Bump Bump
}

// Summarise parses and counts messages.
func Summarise(messages []string) Summary {
	s := Summary{Commits: len(messages), Types: map[string]int{}}

	for _, text := range messages {
		m := Parse(text)

		switch {
		case !m.Conventional:
			s.Unconventional++
		case m.Breaking:
			s.Breaking++
		default:
			s.Types[m.Type]++
		}

		s.Bump = max(s.Bump, m.Bump())
	}

	return s
}
