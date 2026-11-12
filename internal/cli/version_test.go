package cli

import "testing"

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		args       []string
		wantCode   int
		wantStdout string
		wantStderr string
	}{
		{
			name:       "prints full build information",
			args:       []string{"version"},
			wantCode:   ExitOK,
			wantStdout: "svu 1.2.3 (commit abc123, built 2026-01-01T00:00:00Z, go1.26.4, linux/amd64)\n",
		},
		{
			name:       "rejects positional arguments",
			args:       []string{"version", "extra"},
			wantCode:   ExitFailure,
			wantStderr: `svu: unknown command "extra" for "svu version"` + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := run(t, tt.args...)

			if got.code != tt.wantCode {
				t.Errorf("exit code = %d, want %d", got.code, tt.wantCode)
			}

			if got.stdout != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", got.stdout, tt.wantStdout)
			}

			if got.stderr != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", got.stderr, tt.wantStderr)
			}
		})
	}
}
