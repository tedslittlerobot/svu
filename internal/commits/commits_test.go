package commits

import (
	"fmt"
	"maps"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		message  string
		want     Message
		wantBump Bump
	}{
		// The examples in the Conventional Commits 1.0.0 specification.
		{
			name: "reads a breaking change footer",
			message: "feat: allow provided config object to extend other configs\n\n" +
				"BREAKING CHANGE: `extends` key in config file is now used for extending other config files",
			want: Message{
				Conventional: true, Type: "feat", Breaking: true,
				Description: "allow provided config object to extend other configs",
			},
			wantBump: Major,
		},
		{
			name:     "reads ! as a breaking change",
			message:  "feat!: send an email to the customer when a product is shipped",
			want:     Message{Conventional: true, Type: "feat", Breaking: true, Description: "send an email to the customer when a product is shipped"},
			wantBump: Major,
		},
		{
			name:     "reads a scope with !",
			message:  "feat(api)!: send an email to the customer when a product is shipped",
			want:     Message{Conventional: true, Type: "feat", Scope: "api", Breaking: true, Description: "send an email to the customer when a product is shipped"},
			wantBump: Major,
		},
		{
			name:     "makes any type with a breaking change major",
			message:  "chore!: drop support for Node 6\n\nBREAKING CHANGE: use JavaScript features not available in Node 6.",
			want:     Message{Conventional: true, Type: "chore", Breaking: true, Description: "drop support for Node 6"},
			wantBump: Major,
		},
		{
			name:     "gives docs no bump",
			message:  "docs: correct spelling of CHANGELOG",
			want:     Message{Conventional: true, Type: "docs", Description: "correct spelling of CHANGELOG"},
			wantBump: None,
		},
		{
			name:     "gives feat a minor bump",
			message:  "feat(lang): add Polish language",
			want:     Message{Conventional: true, Type: "feat", Scope: "lang", Description: "add Polish language"},
			wantBump: Minor,
		},
		{
			name: "gives fix a patch bump, with a body and footers",
			message: "fix: prevent racing of requests\n\n" +
				"Introduce a request id and a reference to latest request. Dismiss\n" +
				"incoming responses other than from latest request.\n\n" +
				"Reviewed-by: Z\nRefs: #123",
			want:     Message{Conventional: true, Type: "fix", Description: "prevent racing of requests"},
			wantBump: Patch,
		},

		// Further cases.
		{name: "gives perf a patch bump", message: "perf: cache tags", want: Message{Conventional: true, Type: "perf", Description: "cache tags"}, wantBump: Patch},
		{name: "ignores the type's case", message: "Fix: handle empty input", want: Message{Conventional: true, Type: "fix", Description: "handle empty input"}, wantBump: Patch},
		{name: "reads BREAKING-CHANGE as a footer", message: "fix: x\n\nBREAKING-CHANGE: y", want: Message{Conventional: true, Type: "fix", Breaking: true, Description: "x"}, wantBump: Major},
		{name: "needs the footer in uppercase", message: "fix: x\n\nbreaking change: y", want: Message{Conventional: true, Type: "fix", Description: "x"}, wantBump: Patch},
		{name: "ignores surrounding whitespace", message: "\n  feat: x  \n", want: Message{Conventional: true, Type: "feat", Description: "x"}, wantBump: Minor},
		{name: "rejects a merge commit", message: "Merge branch 'main' into feature"},
		{name: "rejects a revert's default message", message: "Revert \"feat: add a flag\""},
		{name: "needs a space after the colon", message: "feat:add a flag"},
		{name: "needs a description", message: "feat: "},
		{name: "rejects an empty scope", message: "feat(): add a flag"},
		{name: "rejects an unclosed scope", message: "feat(api: add a flag"},
		{name: "rejects two scopes", message: "feat(a)(b): add a flag"},
		{name: "rejects ! before the scope", message: "feat!(api): add a flag"},
		{name: "rejects a type with spaces", message: "new feature: add a flag"},
		{name: "rejects a type with digits", message: "v2: add a flag"},
		{name: "rejects an empty message", message: ""},
		{name: "rejects a missing type", message: ": add a flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Parse(tt.message)
			if got != tt.want {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.message, got, tt.want)
			}

			if b := got.Bump(); b != tt.wantBump {
				t.Errorf("Parse(%q).Bump() = %s, want %s", tt.message, b, tt.wantBump)
			}
		})
	}
}

func TestSummarise(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		messages []string
		want     Summary
	}{
		{name: "summarises nothing", messages: nil, want: Summary{Types: map[string]int{}}},
		{
			name:     "takes the biggest bump",
			messages: []string{"fix: a", "feat: b", "fix: c", "docs: d"},
			want:     Summary{Commits: 4, Types: map[string]int{"fix": 2, "feat": 1, "docs": 1}, Bump: Minor},
		},
		{
			name:     "counts a breaking change once, not by its type",
			messages: []string{"feat!: a", "feat: b"},
			want:     Summary{Commits: 2, Breaking: 1, Types: map[string]int{"feat": 1}, Bump: Major},
		},
		{
			name:     "counts unconventional messages without bumping",
			messages: []string{"Update README", "chore: tidy"},
			want:     Summary{Commits: 2, Types: map[string]int{"chore": 1}, Unconventional: 1, Bump: None},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Summarise(tt.messages)

			sameCounts := got.Commits == tt.want.Commits && got.Breaking == tt.want.Breaking &&
				got.Unconventional == tt.want.Unconventional && got.Bump == tt.want.Bump
			if !sameCounts || !maps.Equal(got.Types, tt.want.Types) {
				t.Errorf("Summarise() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBumpString(t *testing.T) {
	t.Parallel()

	for b, want := range map[Bump]string{None: "no", Patch: "patch", Minor: "minor", Major: "major", Bump(9): "unknown"} {
		if got := b.String(); got != want {
			t.Errorf("Bump(%d).String() = %q, want %q", int(b), got, want)
		}
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"feat: x", "fix(api)!: y", "chore: z\n\nBREAKING CHANGE: w", "Merge branch 'a'", "feat(: x", "",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, message string) {
		m := Parse(message)

		if !m.Conventional {
			if m != (Message{}) || m.Bump() != None {
				t.Fatalf("Parse(%q) = %+v, want the zero Message for an unconventional message", message, m)
			}

			return
		}

		if m.Type == "" || m.Type != strings.ToLower(m.Type) || m.Description == "" {
			t.Fatalf("Parse(%q) = %+v, want a lowercase type and a description", message, m)
		}

		if m.Breaking != (m.Bump() == Major) {
			t.Fatalf("Parse(%q) = %+v with bump %s, want breaking exactly when major", message, m, m.Bump())
		}
	})
}

func ExampleSummarise() {
	s := Summarise([]string{
		"feat(parser): accept a leading v",
		"fix: handle empty input",
		"docs: explain stages",
	})

	fmt.Println(s.Commits, s.Bump)
	// Output:
	// 3 minor
}
