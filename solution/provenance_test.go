// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution

import (
	"runtime/debug"
	"testing"

	"github.com/modern-engineering/prototype/solution/image"
)

// stubBuildInfo swaps the build-info read for one crafted shape and
// restores it when the test ends.
func stubBuildInfo(t *testing.T, info *debug.BuildInfo, ok bool) {
	t.Helper()
	restore := readBuildInfo
	readBuildInfo = func() (*debug.BuildInfo, bool) { return info, ok }
	t.Cleanup(func() { readBuildInfo = restore })
}

// The version classification holds over every build shape the
// compiling binary shows up in: only versions the module system
// resolved get recorded; every locally sourced shape — test binaries,
// unstamped builds, directory replacements, VCS-stamped checkout
// builds — records nothing, keeping images machine-independent.
func TestPrototypeVersion(t *testing.T) {
	tests := []struct {
		name string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{"no build info", nil, false, ""},
		{"ordinary dependency", &debug.BuildInfo{
			Main: debug.Module{Path: "example.test/sol", Version: "(devel)"},
			Deps: []*debug.Module{{Path: prototypePath, Version: "v0.4.2"}},
		}, true, "v0.4.2"},
		{"pseudo-version dependency", &debug.BuildInfo{
			Main: debug.Module{Path: "sdl.invalid/solmain", Version: "(devel)"},
			Deps: []*debug.Module{{Path: prototypePath, Version: "v0.0.0-20260101000000-0123456789ab"}},
		}, true, "v0.0.0-20260101000000-0123456789ab"},
		{"directory-replaced dependency", &debug.BuildInfo{
			Main: debug.Module{Path: "sdl.invalid/solmain", Version: "(devel)"},
			Deps: []*debug.Module{{
				Path:    prototypePath,
				Version: "v0.0.0-solution",
				Replace: &debug.Module{Path: "/home/dev/prototype", Version: "(devel)"},
			}},
		}, true, ""},
		{"module-replaced dependency", &debug.BuildInfo{
			Main: debug.Module{Path: "example.test/sol", Version: "(devel)"},
			Deps: []*debug.Module{{
				Path:    prototypePath,
				Version: "v0.4.0",
				Replace: &debug.Module{Path: "example.test/fork", Version: "v0.5.1"},
			}},
		}, true, "v0.5.1"},
		{"absent dependency", &debug.BuildInfo{
			Main: debug.Module{Path: "example.test/sol", Version: "(devel)"},
			Deps: []*debug.Module{{Path: "example.test/other", Version: "v1.0.0"}},
		}, true, ""},
		{"installed prototype main", &debug.BuildInfo{
			Main: debug.Module{Path: prototypePath, Version: "v0.3.0"},
		}, true, "v0.3.0"},
		{"devel prototype main", &debug.BuildInfo{
			Main: debug.Module{Path: prototypePath, Version: "(devel)"},
		}, true, ""},
		{"vcs-stamped prototype main", &debug.BuildInfo{
			Main: debug.Module{Path: prototypePath, Version: "v0.0.0-20260101000000-0123456789ab+dirty"},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "0123456789ab"},
				{Key: "vcs.modified", Value: "true"},
			},
		}, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubBuildInfo(t, tt.info, tt.ok)
			if got := prototypeVersion(); got != tt.want {
				t.Errorf("prototypeVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}

// The governance block assembles as documented: digests in unit
// order with the lowercase-hex SHA-256 rendering, settings sorted by
// key, present only when their version is.
func TestBuildBlock(t *testing.T) {
	stubBuildInfo(t, &debug.BuildInfo{
		Main: debug.Module{Path: "example.test/sol", Version: "(devel)"},
		Deps: []*debug.Module{{Path: prototypePath, Version: "v0.4.2"}},
	}, true)
	cfg := CompileConfig{
		Tool: "v0.4.9",
		Units: []Unit{
			{Name: "b.sdl", Source: "solution hi\n"},
			{Name: "a.sdl", Source: ""},
		},
	}
	got := buildBlock(cfg)
	wantUnits := []image.UnitDigest{
		// shasum -a 256 over the exact source text.
		{Name: "b.sdl", SHA256: "a91f3dadf6ec3b94d20c207b15934f7f26ed0d60fb078dabf4a37a63007caaab"},
		{Name: "a.sdl", SHA256: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
	}
	if len(got.Units) != 2 || got.Units[0] != wantUnits[0] || got.Units[1] != wantUnits[1] {
		t.Errorf("Units = %+v, want %+v (unit order, lowercase hex)", got.Units, wantUnits)
	}
	wantSettings := []image.Setting{
		{Key: "prototype.version", Value: "v0.4.2"},
		{Key: "sdl.version", Value: "v0.4.9"},
	}
	if len(got.Settings) != 2 || got.Settings[0] != wantSettings[0] || got.Settings[1] != wantSettings[1] {
		t.Errorf("Settings = %+v, want %+v (sorted by key)", got.Settings, wantSettings)
	}

	stubBuildInfo(t, nil, false)
	bare := buildBlock(CompileConfig{Units: cfg.Units})
	if len(bare.Settings) != 0 {
		t.Errorf("Settings = %+v, want none when no version resolves and no tool is named", bare.Settings)
	}
}
