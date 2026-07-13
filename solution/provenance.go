// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// The image's governance block: what one compilation records about
// itself. The block must stay machine-independent — the same inputs
// give the same bytes on any checkout — so everything here is either
// a pure content digest or a version the module system actually
// resolved.

package solution

import (
	"crypto/sha256"
	"encoding/hex"
	"runtime/debug"
	"slices"
	"strings"

	"github.com/modern-engineering/prototype/solution/image"
)

// prototypePath is this framework's own module path: the version the
// compiling binary resolved for it becomes the image's
// prototype.version build setting.
const prototypePath = "github.com/modern-engineering/prototype"

// readBuildInfo is debug.ReadBuildInfo, swappable so tests can feed
// the version classification crafted build shapes.
var readBuildInfo = debug.ReadBuildInfo

// buildBlock assembles the image's governance block: one digest per
// unit, and the tool-chain versions that resolved to real module
// versions — the producing tool's from the config, the prototype
// library's from the compiling binary's own build info.
func buildBlock(cfg CompileConfig) *image.Build {
	return &image.Build{
		Units:    unitDigests(cfg.Units),
		Settings: buildSettings(cfg.Tool, prototypeVersion()),
	}
}

// unitDigests digests every unit's exact source text, in unit order —
// the order records keep.
func unitDigests(units []Unit) []image.UnitDigest {
	digests := make([]image.UnitDigest, 0, len(units))
	for _, u := range units {
		sum := sha256.Sum256([]byte(u.Source))
		digests = append(digests, image.UnitDigest{Name: u.Name, SHA256: hex.EncodeToString(sum[:])})
	}
	return digests
}

// buildSettings assembles the build settings, sorted by key; versions
// arrive pre-classified and empty ones record nothing.
func buildSettings(tool, prototype string) []image.Setting {
	var settings []image.Setting
	if prototype != "" {
		settings = append(settings, image.Setting{Key: "prototype.version", Value: prototype})
	}
	if tool != "" {
		settings = append(settings, image.Setting{Key: "sdl.version", Value: tool})
	}
	slices.SortFunc(settings, func(a, b image.Setting) int {
		return strings.Compare(a.Key, b.Key)
	})
	return settings
}

// prototypeVersion is the prototype module version the compiling
// binary linked against, when its build info records an ordinary one:
// the resolved dependency version of the generated compiler's usual
// shape, or the main module's own version for a binary installed from
// the prototype module itself. Locally sourced copies record nothing,
// the line cmd/sdl's skew handshake draws over go list rows — a
// version the module system did not resolve is not a pin worth
// auditing, and recording one would make images machine-dependent.
func prototypeVersion() string {
	info, ok := readBuildInfo()
	if !ok {
		return ""
	}
	if info.Main.Path == prototypePath {
		// A source-checkout build stamps a VCS-derived main version
		// (a tag or pseudo-version, possibly +dirty) that names the
		// checkout, not a module resolution; skip it with the same
		// locally-sourced classification. Test binaries and
		// -buildvcs=false builds report (devel) and land in
		// ordinaryVersion's filter.
		if vcsStamped(info) {
			return ""
		}
		return ordinaryVersion(info.Main.Version)
	}
	for _, dep := range info.Deps {
		if dep.Path != prototypePath {
			continue
		}
		m := dep
		if dep.Replace != nil {
			m = dep.Replace // the copy the build actually resolved
		}
		return ordinaryVersion(m.Version)
	}
	return ""
}

// ordinaryVersion filters a build-info version down to the ones worth
// recording. "(devel)" marks locally sourced code — an unstamped
// build, a directory replacement's target, a workspace member — and
// records nothing; so does the empty version of a module the build
// never resolved.
func ordinaryVersion(v string) string {
	if v == "(devel)" {
		return ""
	}
	return v
}

// vcsStamped reports whether the binary was built from a source
// checkout: the go tool then stamps vcs.* settings beside a
// VCS-derived main version.
func vcsStamped(info *debug.BuildInfo) bool {
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return true
		}
	}
	return false
}
