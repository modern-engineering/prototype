// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package work

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// A Context is the module context the go CLI would resolve imports in
// for one solution directory: the go.mod and go.work governing it, as
// 'go env GOMOD GOWORK' reports them there.
type Context struct {
	gomod  string // path of the governing go.mod; "" without one
	gowork string // path of the governing go.work; "" when absent or off
}

// A Mode is one of the go CLI's module resolution modes; the detected
// Context selects it for the whole build.
type Mode int

const (
	// ModeModule resolves imports through the enclosing module's
	// requirements and replaces.
	ModeModule Mode = iota

	// ModeWorkspace resolves imports through the union of an active
	// go.work's modules.
	ModeWorkspace

	// ModeNone has no enclosing module context at all; import
	// resolution belongs to the generated program's own module.
	ModeNone
)

// Detect reads the module context governing dir with one
// 'go env GOMOD GOWORK' run — the same lookup the go CLI performs — so
// every later resolution decision rests on what the go tool itself
// sees in the solution directory.
func Detect(ctx context.Context, dir string) (*Context, error) {
	out, err := goOutput(ctx, dir, append(os.Environ(), "GOFLAGS="), "env", "GOMOD", "GOWORK")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 2 {
		return nil, fmt.Errorf("go env GOMOD GOWORK: unexpected output %q", out)
	}
	c := &Context{gomod: lines[0], gowork: lines[1]}
	if c.gomod == os.DevNull {
		c.gomod = "" // the go tool's "no module" spelling
	}
	if c.gowork == "off" {
		c.gowork = "" // an explicit GOWORK=off opts out of workspace mode
	}
	return c, nil
}

// Mode reports the resolution mode the context selects: an active
// workspace over an enclosing module over nothing, the go CLI's own
// precedence.
func (c *Context) Mode() Mode {
	switch {
	case c.gowork != "":
		return ModeWorkspace
	case c.gomod != "":
		return ModeModule
	}
	return ModeNone
}

// Env is the environment for child go processes resolving in the
// solution directory: the caller's environment with GOFLAGS emptied, so
// stray -mod=mod or -modfile flags cannot change how the synthesized
// context resolves. Outside workspace mode GOWORK is also forced off,
// pinning the detected verdict for every later invocation; workspace
// mode keeps the ambient GOWORK, because the workspace is the
// resolution context.
func (c *Context) Env() []string {
	if c.Mode() == ModeWorkspace {
		return append(os.Environ(), "GOFLAGS=")
	}
	return append(os.Environ(), "GOWORK=off", "GOFLAGS=")
}
