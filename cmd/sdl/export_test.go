// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package main

// Main hands the whole command — signal wiring, dispatch, process
// exit — to the external test package, whose script harness
// re-executes this very test binary as sdl (the cmd/go precedent for
// driving a program through its own test binary).
var Main = main
