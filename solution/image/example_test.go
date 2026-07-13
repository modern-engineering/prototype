// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package image_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/modern-engineering/prototype/solution/image"
)

// A value crosses the image as a tagged union — the kind plus exactly
// one value field named after it, in the spellings the type's doc
// promises. Durations render canonical (time.Duration.String), not as
// raw nanoseconds, so images stay human-readable.
func ExampleValue() {
	for _, v := range []*image.Value{
		image.String("com.acme.Echo"),
		image.Int(-1),
		image.Bool(true),
		image.Duration(1500 * time.Millisecond),
		image.Token("euCentral1"),
	} {
		out, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		fmt.Println(string(out))
	}

	// Output:
	// {"kind":"string","string":"com.acme.Echo"}
	// {"kind":"int","int":-1}
	// {"kind":"bool","bool":true}
	// {"kind":"duration","duration":"1.5s"}
	// {"kind":"token","token":"euCentral1"}
}
