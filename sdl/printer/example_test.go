// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package printer_test

import (
	"os"

	"github.com/modern-engineering/prototype/sdl/parser"
	"github.com/modern-engineering/prototype/sdl/printer"
)

// Formatting normalizes layout, never lexemes: the ragged source below
// prints with canonical spacing and blank-line folding while the
// author's "1h30m" spelling survives (a synthesized tree, carrying no
// lexeme, would render the same value as "1h30m0s").
func ExampleFprint() {
	const ragged = `solution demo



import   ff   "example.com/ff"

deploy ff.Ping as P1 {

	interval:     1h30m
	count:1

}
`
	f, err := parser.ParseFile("demo.sdl", []byte(ragged))
	if err != nil {
		panic(err)
	}
	if err := printer.Fprint(os.Stdout, f); err != nil {
		panic(err)
	}
	// Output:
	// solution demo
	//
	// import ff "example.com/ff"
	//
	// deploy ff.Ping as P1 {
	//	interval: 1h30m
	//	count: 1
	// }
}
