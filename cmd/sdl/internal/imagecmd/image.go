// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package imagecmd implements the sdl image command group: the
// plumbing verbs that operate on desired-state image files directly,
// the way go mod edit amends and queries go.mod. Where the porcelain
// (build, fmt) works the authoring surface and echo re-renders a whole
// image as source, these verbs amend and inspect THE IMAGE FILE itself
// — mechanical operations for pipelines and tooling, no authoring
// surface involved.
package imagecmd

import "github.com/modern-engineering/prototype/cmd/sdl/internal/base"

// CmdImage is the sdl image command group.
var CmdImage = &base.Command{
	UsageLine: "sdl image <command> [arguments]",
	Short:     "amend and query desired-state image files",
	Long: `Image groups the plumbing verbs that operate on desired-state image
files: edit amends an image in place, info prints its identity and
provenance, and records and symbols print its sections as aligned
tables. The group serves pipelines and tooling; the human-facing
rendering of a whole image is sdl echo.`,
	Commands: []*base.Command{
		cmdEdit,
		cmdInfo,
		cmdRecords,
		cmdSymbols,
	},
}
