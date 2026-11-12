package colour

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestParseMode(t *testing.T) {
	t.Parallel()

	for _, want := range []Mode{Auto, Always, Never} {
		got, err := ParseMode(want.String())
		if err != nil || got != want {
			t.Errorf("ParseMode(%q) = %v, %v, want %v, nil", want.String(), got, err, want)
		}
	}

	if _, err := ParseMode("sometimes"); !errors.Is(err, ErrInvalidMode) {
		t.Errorf("ParseMode(%q) error = %v, want ErrInvalidMode", "sometimes", err)
	}

	if got, want := Mode(9).String(), "mode 9"; got != want {
		t.Errorf("Mode(9).String() = %q, want %q", got, want)
	}
}

func TestEnabled(t *testing.T) {
	t.Parallel()

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = devNull.Close() })

	regular, err := os.Create(filepath.Join(t.TempDir(), "stderr"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = regular.Close() })

	tests := []struct {
		name string
		mode Mode
		env  map[string]string
		w    io.Writer
		want bool
	}{
		{name: "always wins over the no-colour variable", mode: Always, env: map[string]string{envNoColour: "1"}, w: &bytes.Buffer{}, want: true},
		{name: "never wins over the force-colour variable", mode: Never, env: map[string]string{envForceColour: "1"}, w: devNull, want: false},
		{name: "the no-colour variable turns colour off", env: map[string]string{envNoColour: "1", envForceColour: "1"}, w: devNull, want: false},
		{name: "the force-colour variable turns colour on", env: map[string]string{envForceColour: "1", "TERM": "dumb"}, w: &bytes.Buffer{}, want: true},
		{name: "TERM=dumb turns colour off", env: map[string]string{"TERM": "dumb"}, w: devNull, want: false},
		{name: "uses colour on a character device", w: devNull, want: true},
		{name: "doesn't use colour in a regular file", w: regular, want: false},
		{name: "doesn't use colour in a buffer", w: &bytes.Buffer{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			getenv := func(key string) string { return tt.env[key] }

			if got := Enabled(tt.mode, getenv, tt.w); got != tt.want {
				t.Errorf("Enabled() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestEnabledIgnoresUnreadableFiles(t *testing.T) {
	t.Parallel()

	f, err := os.Create(filepath.Join(t.TempDir(), "closed"))
	if err != nil {
		t.Fatal(err)
	}

	_ = f.Close()

	if Enabled(Auto, func(string) string { return "" }, f) {
		t.Error("Enabled() = true for a closed file, want false")
	}
}

func TestStyler(t *testing.T) {
	t.Parallel()

	on, off := New(true), New(false)

	tests := []struct {
		name    string
		style   func(Styler, string) string
		wantOn  string
		wantOff string
	}{
		{name: "bold", style: Styler.Bold, wantOn: "\x1b[1mx\x1b[0m", wantOff: "x"},
		{name: "dim", style: Styler.Dim, wantOn: "\x1b[2mx\x1b[0m", wantOff: "x"},
		{name: "red", style: Styler.Red, wantOn: "\x1b[31mx\x1b[0m", wantOff: "x"},
		{name: "green", style: Styler.Green, wantOn: "\x1b[32mx\x1b[0m", wantOff: "x"},
		{name: "yellow", style: Styler.Yellow, wantOn: "\x1b[33mx\x1b[0m", wantOff: "x"},
		{name: "cyan", style: Styler.Cyan, wantOn: "\x1b[36mx\x1b[0m", wantOff: "x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.style(on, "x"); got != tt.wantOn {
				t.Errorf("%s with colour = %q, want %q", tt.name, got, tt.wantOn)
			}

			if got := tt.style(off, "x"); got != tt.wantOff {
				t.Errorf("%s without colour = %q, want %q", tt.name, got, tt.wantOff)
			}
		})
	}

	if got := on.Bold(""); got != "" {
		t.Errorf("Bold(\"\") = %q, want it left empty", got)
	}

	if got := (Styler{}).Cyan("x"); got != "x" {
		t.Errorf("zero Styler Cyan() = %q, want %q", got, "x")
	}
}
