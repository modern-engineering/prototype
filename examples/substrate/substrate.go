// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package substrate is the sample symbol-type catalogue: the classes
// of late-bound values a solution wires through extern symbols. Where
// package ff shows the component kind of catalogue citizen, substrate
// shows the other — exported *solution.SymbolType vars that discovery
// registers so extern declarations can name them, with sensitivity
// declared where the type is.
package substrate

import "github.com/modern-engineering/prototype/solution"

// Secret is an operator-supplied credential. It is sensitive: every
// binding wired from a Secret-typed extern is tainted in the image, so
// deployment backends know what they must not log or expose.
var Secret = &solution.SymbolType{
	Doc:       "an operator-supplied credential, bound by the site and redacted wherever it surfaces",
	Sensitive: true,
}

// Endpoint is a network coordinate — an address, URL, or subject — the
// deploying site routes traffic through.
var Endpoint = &solution.SymbolType{
	Doc: "a site-bound network coordinate such as an address or subject",
}
