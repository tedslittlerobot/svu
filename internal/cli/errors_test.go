package cli

import (
	"bytes"
	"errors"
	"testing"
)

func TestUserErrorKeepsTheCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("disk on fire")

	tests := []struct {
		name string
		err  *userError
		want string
	}{
		{name: "describes itself by its headline", err: &userError{headline: "Something failed."}, want: "Something failed."},
		{
			name: "includes the cause in its description",
			err:  &userError{headline: "Something failed.", err: cause},
			want: "Something failed.: disk on fire",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}

			if got, want := errors.Unwrap(tt.err), tt.err.err; !errors.Is(got, want) {
				t.Errorf("Unwrap() = %v, want %v", got, want)
			}
		})
	}
}

func TestRenderNeverShowsTheCause(t *testing.T) {
	t.Parallel()

	var b bytes.Buffer

	render(&b, &userError{headline: "Something failed.", err: errors.New("disk on fire")})

	if got, want := b.String(), "Error: Something failed.\n"; got != want {
		t.Errorf("render() wrote %q, want %q", got, want)
	}
}
