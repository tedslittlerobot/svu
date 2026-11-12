package cli

import (
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/tedslittlerobot/svu/internal/colour"
)

// helpGroupAnnotation is the flag annotation that names a flag's group in
// help.
const helpGroupAnnotation = "svu_help_group"

// Flag groups, in the order help shows them. Flags without a group are shown
// last.
const (
	groupInput   = "Input"
	groupFilter  = "Filter"
	groupVersion = "Version modifiers (use at most one)"
	groupStage   = "Stage modifiers (use at most one)"
	groupOutput  = "Output"
	groupTagging = "Tagging"
)

var helpGroups = []string{groupInput, groupFilter, groupVersion, groupStage, groupOutput, groupTagging}

// setHelpGroup puts a flag in a help group. The flag must exist.
func setHelpGroup(flags *pflag.FlagSet, name, group string) {
	_ = flags.SetAnnotation(name, helpGroupAnnotation, []string{group}) // Only fails if the flag doesn't exist.
}

// renderHelp writes help for cmd to w, styled with s. Help always goes to
// stderr (conventions/output.md), so it never mixes with a result.
func renderHelp(w io.Writer, cmd *cobra.Command, s colour.Styler) {
	var b strings.Builder

	description := cmd.Long
	if description == "" {
		description = cmd.Short
	}

	b.WriteString(styleFlags(s, strings.TrimSpace(description)) + "\n")

	writeUsage(&b, cmd, s)
	writeExamples(&b, cmd, s)
	writeCommands(&b, cmd, s)
	writeFlags(&b, cmd, s)

	if cmd.HasAvailableSubCommands() {
		b.WriteString("\nRun " + s.Cyan(cmd.Root().Name()+" help <command>") + " for help with a command.\n")
	}

	_, _ = io.WriteString(w, b.String())
}

func heading(b *strings.Builder, s colour.Styler, title string) {
	b.WriteString("\n" + s.Bold(title+":") + "\n")
}

func writeUsage(b *strings.Builder, cmd *cobra.Command, s colour.Styler) {
	heading(b, s, "Usage")

	// A command with subcommands is only used through them, even if it runs
	// something on its own.
	if cmd.HasAvailableSubCommands() {
		b.WriteString("  " + s.Cyan(cmd.CommandPath()) + " " + s.Dim("<command>") + "\n")
	} else {
		rest := strings.TrimPrefix(cmd.UseLine(), cmd.CommandPath()+" ")
		b.WriteString("  " + s.Cyan(cmd.CommandPath()) + " " + s.Dim(rest) + "\n")
	}

	if len(cmd.Aliases) > 0 {
		heading(b, s, "Aliases")
		b.WriteString("  " + s.Cyan(strings.Join(append([]string{cmd.Name()}, cmd.Aliases...), ", ")) + "\n")
	}
}

// writeExamples shows cmd.Example, dimming any "# comment" on each line.
func writeExamples(b *strings.Builder, cmd *cobra.Command, s colour.Styler) {
	if cmd.Example == "" {
		return
	}

	heading(b, s, "Examples")

	for line := range strings.SplitSeq(cmd.Example, "\n") {
		code, comment, hasComment := strings.Cut(line, "#")
		if hasComment {
			line = code + s.Dim("#"+comment)
		}

		b.WriteString(line + "\n")
	}
}

func writeCommands(b *strings.Builder, cmd *cobra.Command, s colour.Styler) {
	var commands []*cobra.Command

	for _, c := range cmd.Commands() {
		// cobra counts the help command as unavailable, but it's one of svu's
		// commands like any other.
		if c.IsAvailableCommand() || (c.Name() == "help" && !c.Hidden) {
			commands = append(commands, c)
		}
	}

	if len(commands) == 0 {
		return
	}

	names := func(c *cobra.Command) string {
		return strings.Join(append([]string{c.Name()}, c.Aliases...), ", ")
	}

	width := 0
	for _, c := range commands {
		width = max(width, len(names(c)))
	}

	heading(b, s, "Commands")

	for _, c := range commands {
		b.WriteString("  " + s.Cyan(names(c)) + strings.Repeat(" ", width-len(names(c))) + "  " + styleFlags(s, c.Short) + "\n")
	}
}

// flagLine is one flag as help shows it.
type flagLine struct {
	names       string // Such as "-h, --help".
	placeholder string // Such as "path", or "" for a boolean flag.
	usage       string
}

func (l flagLine) width() int {
	if l.placeholder == "" {
		return len(l.names)
	}

	return len(l.names) + 1 + len(l.placeholder)
}

// writeFlags shows cmd's flags in their groups, then any flags it inherits
// from its parents.
func writeFlags(b *strings.Builder, cmd *cobra.Command, s colour.Styler) {
	grouped := map[string][]flagLine{}

	var ungrouped, inherited []flagLine

	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if f.Hidden {
			return
		}

		if group := f.Annotations[helpGroupAnnotation]; len(group) == 1 {
			grouped[group[0]] = append(grouped[group[0]], newFlagLine(f))

			return
		}

		ungrouped = append(ungrouped, newFlagLine(f))
	})

	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Hidden {
			inherited = append(inherited, newFlagLine(f))
		}
	})

	// One width for every group, so all the descriptions line up.
	width := 0
	for _, lines := range append(slices.Collect(maps.Values(grouped)), ungrouped, inherited) {
		for _, l := range lines {
			width = max(width, l.width())
		}
	}

	for _, group := range helpGroups {
		writeFlagGroup(b, s, group, grouped[group], width)
	}

	other := "Flags"
	if len(grouped) > 0 {
		other = "Other"
	}

	writeFlagGroup(b, s, other, ungrouped, width)
	writeFlagGroup(b, s, "Global flags", inherited, width)
}

func writeFlagGroup(b *strings.Builder, s colour.Styler, title string, lines []flagLine, width int) {
	if len(lines) == 0 {
		return
	}

	heading(b, s, title)

	for _, l := range lines {
		names := strings.TrimLeft(l.names, " ")
		b.WriteString("  " + strings.Repeat(" ", len(l.names)-len(names)) + s.Cyan(names))

		if l.placeholder != "" {
			b.WriteString(" " + s.Dim(l.placeholder))
		}

		b.WriteString(strings.Repeat(" ", width-l.width()) + "  " + styleFlags(s, l.usage) + "\n")
	}
}

func newFlagLine(f *pflag.Flag) flagLine {
	placeholder, usage := pflag.UnquoteUsage(f)

	// cobra adds --help to each command with its own wording.
	if f.Name == "help" {
		usage = "Show this help"
	}

	names := "    --" + f.Name
	if f.Shorthand != "" {
		names = "-" + f.Shorthand + ", --" + f.Name
	}

	return flagLine{names: names, placeholder: placeholder, usage: usage}
}

// newHelpCommand returns svu's help command. Unlike --help, it writes help to
// stdout, because help is its main output. show renders help for a command
// to a stream.
func newHelpCommand(show func(w io.Writer, cmd *cobra.Command)) *cobra.Command {
	return &cobra.Command{
		Use:   "help [command]",
		Short: "Show help for svu or one of its commands",
		Args:  cobra.ArbitraryArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
			var names []cobra.Completion

			if len(args) == 0 {
				for _, c := range cmd.Root().Commands() {
					if c.IsAvailableCommand() {
						names = append(names, cobra.CompletionWithDesc(c.Name(), c.Short))
					}
				}
			}

			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			target, rest, err := cmd.Root().Find(args)
			if err != nil || len(rest) > 0 {
				return unknownCommandError(cmd.Root(), strings.Join(args, " "))
			}

			target.InitDefaultHelpFlag() // So that help lists --help.
			show(cmd.OutOrStdout(), target)

			return nil
		},
	}
}

// styleFlags colours any flag named in prose, such as "--directory" or
// "(alias --vp)", the way help colours flags. A flag starts a word.
func styleFlags(s colour.Styler, text string) string {
	var b strings.Builder

	for i := 0; i < len(text); {
		if !startsFlag(text, i) {
			b.WriteByte(text[i])
			i++

			continue
		}

		end := i
		for end < len(text) && (text[end] == '-' || isLower(text[end])) {
			end++
		}

		b.WriteString(s.Cyan(text[i:end]))
		i = end
	}

	return b.String()
}

// startsFlag reports whether a flag such as "--minor" starts at text[i].
func startsFlag(text string, i int) bool {
	if i > 0 && !strings.ContainsRune(" (\n", rune(text[i-1])) {
		return false
	}

	return strings.HasPrefix(text[i:], "--") && i+2 < len(text) && isLower(text[i+2])
}

func isLower(c byte) bool {
	return c >= 'a' && c <= 'z'
}
