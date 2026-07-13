// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution_test

import (
	"context"
	"fmt"
	"os"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/solution"
)

// ExampleMainCompile compiles a tiny solution through the Go API — the
// call a generated main makes with its embedded units and imported
// catalogue packages — and lets the image land on stdout.
func ExampleMainCompile() {
	hello := &application.Descriptor{
		Name: "hello",
		Doc:  "print a greeting",
		Make: func() application.Service {
			return application.Main(func(context.Context) error { return nil })
		},
	}

	code := solution.MainCompile(solution.CompileConfig{
		Solution: "hi",
		Units: []solution.Unit{{
			Name: "hi.sdl",
			Source: "solution hi\n" +
				"import greet \"example.com/greet\"\n" +
				"deploy greet.Hello as hi\n",
		}},
		Catalogue: []solution.Package{{
			Path:     "example.com/greet",
			Name:     "greet",
			Elements: []solution.Element{solution.App("Hello", hello)},
		}},
		Output: os.Stdout,
	})
	fmt.Println("exit", code)

	// Output:
	// {
	//   "format": "solution-image/1",
	//   "solution": "hi",
	//   "generation": 1,
	//   "build": {
	//     "units": [
	//       {
	//         "name": "hi.sdl",
	//         "sha256": "bde072d6dd1bdb052257fb4d7dc8c8c19d2549d8cf40c99bc3c569222b989e05"
	//       }
	//     ]
	//   },
	//   "catalogue": [
	//     {
	//       "path": "example.com/greet",
	//       "name": "greet",
	//       "elements": [
	//         {
	//           "name": "Hello",
	//           "kind": "component",
	//           "doc": "print a greeting"
	//         }
	//       ]
	//     }
	//   ],
	//   "symbols": [],
	//   "records": [
	//     {
	//       "verb": "deploy",
	//       "element": {
	//         "package": "example.com/greet",
	//         "name": "Hello"
	//       },
	//       "name": "hi"
	//     }
	//   ]
	// }
	// exit 0
}
