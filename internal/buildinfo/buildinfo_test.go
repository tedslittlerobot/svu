package buildinfo

import (
	"runtime"
	"runtime/debug"
	"testing"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	toolchain := func(mainVersion string, settings ...debug.BuildSetting) func() (*debug.BuildInfo, bool) {
		return func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				Main:     debug.Module{Version: mainVersion},
				Settings: settings,
			}, true
		}
	}

	noBuildInfo := func() (*debug.BuildInfo, bool) { return nil, false }

	vcs := []debug.BuildSetting{
		{Key: "vcs.revision", Value: "toolchain-sha"},
		{Key: "vcs.time", Value: "2026-01-01T00:00:00Z"},
	}

	tests := []struct {
		name    string
		version string
		commit  string
		date    string
		read    func() (*debug.BuildInfo, bool)
		want    Info
	}{
		{
			name:    "link-time values take precedence over toolchain metadata",
			version: "1.2.3",
			commit:  "linker-sha",
			date:    "2026-02-02T00:00:00Z",
			read:    toolchain("v9.9.9", vcs...),
			want:    Info{Version: "1.2.3", Commit: "linker-sha", Date: "2026-02-02T00:00:00Z"},
		},
		{
			name: "toolchain metadata fills in when nothing was linked",
			read: toolchain("v1.4.0", vcs...),
			want: Info{Version: "1.4.0", Commit: "toolchain-sha", Date: "2026-01-01T00:00:00Z"},
		},
		{
			name:    "partial link-time values are completed from the toolchain",
			version: "2.0.0",
			read:    toolchain("v9.9.9", vcs...),
			want:    Info{Version: "2.0.0", Commit: "toolchain-sha", Date: "2026-01-01T00:00:00Z"},
		},
		{
			name:    "leading v is stripped from linked versions",
			version: "v3.1.4",
			read:    noBuildInfo,
			want:    Info{Version: "3.1.4", Commit: Unknown, Date: Unknown},
		},
		{
			name: "devel module version is treated as missing",
			read: toolchain("(devel)"),
			want: Info{Version: DevVersion, Commit: Unknown, Date: Unknown},
		},
		{
			name: "placeholders are used when no metadata exists",
			read: noBuildInfo,
			want: Info{Version: DevVersion, Commit: Unknown, Date: Unknown},
		},
		{
			name: "nil build info is tolerated",
			read: func() (*debug.BuildInfo, bool) { return nil, true },
			want: Info{Version: DevVersion, Commit: Unknown, Date: Unknown},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := resolve(tt.version, tt.commit, tt.date, tt.read)

			want := tt.want
			want.GoVersion = runtime.Version()
			want.Platform = runtime.GOOS + "/" + runtime.GOARCH

			if got != want {
				t.Errorf("resolve() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestInfoString(t *testing.T) {
	t.Parallel()

	info := Info{
		Version:   "1.2.3",
		Commit:    "abc123",
		Date:      "2026-01-01T00:00:00Z",
		GoVersion: "go1.26.4",
		Platform:  "linux/amd64",
	}

	const want = "semvu 1.2.3 (commit abc123, built 2026-01-01T00:00:00Z, go1.26.4, linux/amd64)"

	if got := info.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	got := Get()

	if got.Version == "" || got.Commit == "" || got.Date == "" {
		t.Errorf("Get() returned empty fields: %+v", got)
	}
}
