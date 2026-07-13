// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package token_test

import (
	"fmt"

	"github.com/modern-engineering/prototype/sdl/token"
)

// Grammar-derived tooling — syntax highlighters, completion — takes the
// whole keyword vocabulary from Keywords instead of asking Lookup one
// word at a time.
func ExampleKeywords() {
	for kw := range token.Keywords() {
		fmt.Println(kw)
	}
	// Output:
	// solution
	// import
	// extern
	// var
	// default
	// deploy
	// provision
	// as
	// true
	// false
}
