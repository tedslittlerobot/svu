// Package colour decides whether semvu's stderr output uses colour, and styles
// text with ANSI escape codes when it does.
//
// The rules are in conventions/output.md, under "Colour". Only the 16 basic
// ANSI colours, bold and dim are used, so the terminal theme picks the shades,
// and every styled span ends with a reset.
package colour

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
)

// Mode is the value of the --colour flag.
type Mode int

// The colour modes. The zero value is Auto.
const (
	// Auto uses colour if the environment and stream allow it.
	Auto Mode = iota
	// Always uses colour regardless of the environment.
	Always
	// Never doesn't use colour.
	Never
)

var modeNames = map[Mode]string{
	Auto:   "auto",
	Always: "always",
	Never:  "never",
}

// ErrInvalidMode is returned by ParseMode for an unrecognised mode.
var ErrInvalidMode = errors.New("colour mode must be auto, always or never")

// ParseMode parses a mode name: "auto", "always" or "never".
func ParseMode(s string) (Mode, error) {
	for m, name := range modeNames {
		if s == name {
			return m, nil
		}
	}

	return Auto, fmt.Errorf("%w: %q", ErrInvalidMode, s)
}

// String returns the mode's name, such as "auto".
func (m Mode) String() string {
	if name, ok := modeNames[m]; ok {
		return name
	}

	return fmt.Sprintf("mode %d", int(m))
}

// Environment variables that other tools define, so they keep the American
// spelling.
const (
	envNoColour    = "NO_COLOR"    //nolint:misspell // Defined by https://no-color.org.
	envForceColour = "FORCE_COLOR" //nolint:misspell // A convention shared by many tools.
)

// Enabled reports whether output written to w should use colour. getenv looks
// up an environment variable, returning "" if it is unset. The first rule that
// applies wins:
//
//  1. mode Always or Never;
//  2. the no-colour variable (https://no-color.org) set to anything turns
//     colour off;
//  3. the force-colour variable set to anything turns colour on;
//  4. TERM=dumb turns colour off;
//  5. otherwise, colour is on only if w is a terminal.
func Enabled(mode Mode, getenv func(string) string, w io.Writer) bool {
	switch {
	case mode == Always:
		return true
	case mode == Never:
		return false
	case getenv(envNoColour) != "":
		return false
	case getenv(envForceColour) != "":
		return true
	case getenv("TERM") == "dumb":
		return false
	default:
		return isTerminal(w)
	}
}

// isTerminal reports whether w is a character device. That includes
// terminals, and also devices such as /dev/null, where colour does no harm
// because nobody reads it.
func isTerminal(w io.Writer) bool {
	f, ok := w.(interface{ Stat() (fs.FileInfo, error) })
	if !ok {
		return false
	}

	info, err := f.Stat()

	return err == nil && info.Mode()&fs.ModeCharDevice != 0
}

// Styler applies styles to text, or leaves it alone if colour is off. The zero
// Styler leaves text alone.
type Styler struct {
	on bool
}

// New returns a Styler that styles text if on is true.
func New(on bool) Styler {
	return Styler{on: on}
}

// SGR parameters for the styles semvu uses.
const (
	bold   = "1"
	dim    = "2"
	red    = "31"
	green  = "32"
	yellow = "33"
	cyan   = "36"
)

func (s Styler) apply(text string, codes string) string {
	if !s.on || text == "" {
		return text
	}

	return "\x1b[" + codes + "m" + text + "\x1b[0m"
}

// Bold makes text bold.
func (s Styler) Bold(text string) string { return s.apply(text, bold) }

// Dim makes text dim, for secondary information.
func (s Styler) Dim(text string) string { return s.apply(text, dim) }

// Red colours text red.
func (s Styler) Red(text string) string { return s.apply(text, red) }

// Green colours text green.
func (s Styler) Green(text string) string { return s.apply(text, green) }

// Yellow colours text yellow.
func (s Styler) Yellow(text string) string { return s.apply(text, yellow) }

// Cyan colours text cyan.
func (s Styler) Cyan(text string) string { return s.apply(text, cyan) }
