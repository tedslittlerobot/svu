package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

// reportHint asks for a bug report, for failures the user can't fix.
const reportHint = "If this keeps happening, please report it at https://github.com/tedslittlerobot/svm/issues."

// userError is an error with a message written for the user, rendered as
// described in conventions/output.md. Every error svu itself returns from a
// command is a userError; anything else comes from cobra.
type userError struct {
	// headline is one sentence saying what went wrong, without "Error: ".
	headline string
	// pointer, if set, shows where in an input the problem is.
	pointer *pointer
	// detail explains the rule that was broken. It may span several lines.
	detail string
	// hint suggests a fix, without "Hint: ".
	hint string
	// err is the underlying error, if any. It is never shown to the user.
	err error
}

// pointer marks a position in an input.
type pointer struct {
	input string
	pos   int // Byte offset into input.
}

// Error implements error. It is for logs and tests; users see render's output.
func (e *userError) Error() string {
	if e.err != nil {
		return e.headline + ": " + e.err.Error()
	}

	return e.headline
}

// Unwrap returns the underlying error.
func (e *userError) Unwrap() error {
	return e.err
}

// render writes err to w for the user to read.
func render(w io.Writer, err error) {
	var ue *userError
	if !errors.As(err, &ue) {
		// cobra's own errors, such as an unknown flag.
		_, _ = fmt.Fprintf(w, "svu: %v\n", err)

		return
	}

	var b strings.Builder

	fmt.Fprintf(&b, "Error: %s\n", ue.headline)

	if ue.pointer.printable() || ue.detail != "" {
		b.WriteString("\n")
	}

	if ue.pointer.printable() {
		p := ue.pointer
		column := utf8.RuneCountInString(p.input[:p.pos])
		fmt.Fprintf(&b, "  %s\n  %s^\n", p.input, strings.Repeat(" ", column))
	}

	if ue.detail != "" {
		for line := range strings.SplitSeq(ue.detail, "\n") {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}

	if ue.hint != "" {
		fmt.Fprintf(&b, "\nHint: %s\n", ue.hint)
	}

	_, _ = io.WriteString(w, b.String())
}

// printable reports whether the pointer can be drawn: the input must be on
// one line and contain nothing that would throw the "^" out of line.
func (p *pointer) printable() bool {
	if p == nil || p.input == "" || p.pos < 0 || p.pos > len(p.input) {
		return false
	}

	for _, r := range p.input {
		if !unicode.IsPrint(r) {
			return false
		}
	}

	return true
}
