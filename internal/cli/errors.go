package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/tedslittlerobot/svm/internal/colour"
)

// reportHint asks for a bug report, for failures the user can't fix.
const reportHint = "If this keeps happening, please report it at https://github.com/tedslittlerobot/svm/issues."

// userError is an error with a message written for the user, rendered as
// described in conventions/output.md. Every error svu returns from a command
// is a userError, and render turns anything else into one.
//
// In a hint, wrap suggested commands, flags and values in backticks, as in
// "run `svu --help`". render shows them in green, without the backticks.
// Quoted values in a hint are green too.
type userError struct {
	// headline is one sentence saying what went wrong, without "Error: ".
	// Quote any value the user gave with %q.
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

// asUserError returns err as a userError. An error that isn't one, which can
// only come from cobra, keeps its message as the headline, written as a
// sentence.
func asUserError(err error) *userError {
	var ue *userError
	if errors.As(err, &ue) {
		return ue
	}

	headline := err.Error()
	first, size := utf8.DecodeRuneInString(headline)
	headline = string(unicode.ToUpper(first)) + headline[size:]

	if !strings.HasSuffix(headline, ".") {
		headline += "."
	}

	return &userError{headline: headline, err: err}
}

// render writes err to w for the user to read, styled with s as
// conventions/output.md describes.
func render(w io.Writer, err error, s colour.Styler) {
	ue := asUserError(err)

	var b strings.Builder

	b.WriteString(s.Bold(s.Red("Error:")) + " " + styleHeadline(s, ue.headline) + "\n")

	if ue.pointer.printable() || ue.detail != "" {
		b.WriteString("\n")
	}

	if ue.pointer.printable() {
		p := ue.pointer
		column := utf8.RuneCountInString(p.input[:p.pos])
		b.WriteString("  " + p.input[:p.pos] + s.Red(p.input[p.pos:]) + "\n")
		b.WriteString("  " + strings.Repeat(" ", column) + s.Red("^") + "\n")
	}

	if ue.detail != "" {
		for line := range strings.SplitSeq(ue.detail, "\n") {
			b.WriteString("  " + line + "\n")
		}
	}

	if ue.hint != "" {
		b.WriteString("\n" + s.Bold(s.Cyan("Hint:")) + " " + styleHint(s, ue.hint) + "\n")
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

// styleHeadline colours a headline red, with quoted user input in bold.
func styleHeadline(s colour.Styler, headline string) string {
	var b strings.Builder

	for _, t := range tokenise(headline) {
		switch t.kind {
		case tokenQuoted:
			b.WriteString(s.Bold(s.Red(t.text)))
		default:
			b.WriteString(s.Red(t.text))
		}
	}

	return b.String()
}

// styleHint colours suggested values in a hint green, and URLs dim. It drops
// the backticks that mark suggestions, with or without colour.
func styleHint(s colour.Styler, hint string) string {
	var b strings.Builder

	for _, t := range tokenise(hint) {
		switch t.kind {
		case tokenCode, tokenQuoted:
			b.WriteString(s.Green(t.text))
		case tokenURL:
			b.WriteString(s.Dim(t.text))
		default:
			b.WriteString(t.text)
		}
	}

	return b.String()
}

type tokenKind int

const (
	tokenText   tokenKind = iota
	tokenCode             // `svu --help`, without the backticks.
	tokenQuoted           // "1.2.3", with its quotes, as %q writes it.
	tokenURL              // https://…, without a sentence's final full stop.
)

type token struct {
	kind tokenKind
	text string
}

// tokenise splits a message into plain text, `code`, "quoted" values and
// URLs.
func tokenise(msg string) []token {
	var (
		tokens []token
		text   strings.Builder
	)

	emit := func(kind tokenKind, s string) {
		if text.Len() > 0 {
			tokens = append(tokens, token{kind: tokenText, text: text.String()})
			text.Reset()
		}

		tokens = append(tokens, token{kind: kind, text: s})
	}

	for i := 0; i < len(msg); {
		switch {
		case msg[i] == '`':
			if end := strings.IndexByte(msg[i+1:], '`'); end >= 0 {
				emit(tokenCode, msg[i+1:i+1+end])
				i += end + 2

				continue
			}
		case msg[i] == '"':
			if end := quotedEnd(msg, i); end > 0 {
				emit(tokenQuoted, msg[i:end])
				i = end

				continue
			}
		case strings.HasPrefix(msg[i:], "https://"):
			end := i + strings.IndexFunc(msg[i:], unicode.IsSpace)
			if end < i {
				end = len(msg)
			}

			url := strings.TrimRight(msg[i:end], ".")
			emit(tokenURL, url)
			i += len(url)

			continue
		}

		text.WriteByte(msg[i])
		i++
	}

	if text.Len() > 0 {
		tokens = append(tokens, token{kind: tokenText, text: text.String()})
	}

	return tokens
}

// quotedEnd returns the index just after the Go-quoted string that starts at
// msg[start], or -1 if it isn't closed.
func quotedEnd(msg string, start int) int {
	for i := start + 1; i < len(msg); i++ {
		switch msg[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}

	return -1
}

// valueRuleError explains, for the user, why a flag's value is invalid. It
// completes the sentence "--flag …", as in "must be auto, always or never".
type valueRuleError string

// Error implements error.
func (e valueRuleError) Error() string {
	return string(e)
}

// flagError turns pflag's errors into messages that follow
// conventions/output.md. cobra calls it for every flag that fails to parse.
func flagError(cmd *cobra.Command, err error) error {
	var (
		notExist *pflag.NotExistError
		required *pflag.ValueRequiredError
		invalid  *pflag.InvalidValueError
		syntax   *pflag.InvalidSyntaxError
	)

	path := cmd.CommandPath()

	switch {
	case errors.As(err, &notExist):
		return unknownFlagError(cmd, notExist, err)
	case errors.As(err, &required):
		f := required.GetFlag()
		placeholder, _ := pflag.UnquoteUsage(f)

		return &userError{
			headline: fmt.Sprintf("--%s needs a value.", f.Name),
			hint:     fmt.Sprintf("Give one, as in `--%s %s`.", f.Name, placeholder),
			err:      err,
		}
	case errors.As(err, &invalid):
		return invalidValueError(invalid, err)
	case errors.As(err, &syntax):
		return &userError{
			headline: fmt.Sprintf("%q is not a valid flag.", syntax.GetSpecifiedFlag()),
			hint:     fmt.Sprintf("To list the flags of %s, run `%s --help`.", path, path),
			err:      err,
		}
	default:
		return asUserError(err)
	}
}

func unknownFlagError(cmd *cobra.Command, notExist *pflag.NotExistError, err error) *userError {
	path := cmd.CommandPath()

	flag := "--" + notExist.GetSpecifiedName()
	if notExist.GetSpecifiedShortnames() != "" {
		flag = "-" + notExist.GetSpecifiedName()
	}

	ue := &userError{
		headline: fmt.Sprintf("%q is not a flag of %s.", flag, path),
		hint:     fmt.Sprintf("To list the flags of %s, run `%s --help`.", path, path),
		err:      err,
	}

	// A flag that belongs to another command, as in svu --patch.
	if owner := flagOwner(cmd.Root(), notExist.GetSpecifiedName()); owner != nil && owner != cmd {
		ue.hint = fmt.Sprintf("Did you mean `%s %s`?", owner.CommandPath(), flag)
	}

	return ue
}

// flagOwner returns the first command under root with a long flag called
// name, or nil.
func flagOwner(root *cobra.Command, name string) *cobra.Command {
	if root.LocalFlags().Lookup(name) != nil {
		return root
	}

	for _, c := range root.Commands() {
		if owner := flagOwner(c, name); owner != nil {
			return owner
		}
	}

	return nil
}

func invalidValueError(invalid *pflag.InvalidValueError, err error) *userError {
	f := invalid.GetFlag()

	ue := &userError{
		headline: fmt.Sprintf("%q is not a valid value for --%s.", invalid.GetValue(), f.Name),
		err:      err,
	}

	var rule valueRuleError

	switch {
	case errors.As(invalid, &rule):
		ue.detail = fmt.Sprintf("--%s %s.", f.Name, rule)
	case f.Value.Type() == "bool":
		ue.detail = fmt.Sprintf("--%s is a switch: it is on when given, and takes no value.", f.Name)
		ue.hint = fmt.Sprintf("Use `--%s` on its own.", f.Name)
	}

	return ue
}
