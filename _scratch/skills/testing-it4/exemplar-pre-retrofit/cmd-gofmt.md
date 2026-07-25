# cmd/gofmt — golden beside fixture, -update, idempotence

Source: `cmd/gofmt/gofmt_test.go` and `cmd/gofmt/testdata/rewrite1.input` (Go
development tree, 1.27 dev, 2026-06). Verbatim; BSD-3-Clause, © The Go Authors.
Note: `package main`, in-process — the gofmt binary is never executed.

gofmt_test.go lines 112-136 (the tail of `runTest(t, in, out string)`; flag
parsing and processing elided):

```go
	expected, err := os.ReadFile(out)
	if err != nil {
		t.Error(err)
		return
	}

	if got := buf.Bytes(); !bytes.Equal(got, expected) {
		if *update {
			if in != out {
				if err := os.WriteFile(out, got, 0666); err != nil {
					t.Error(err)
				}
				return
			}
			// in == out: don't accidentally destroy input
			t.Errorf("WARNING: -update did not rewrite input file %s", in)
		}

		t.Errorf("(gofmt %s) != %s (see %s.gofmt)\n%s", in, out, in,
			diff.Diff("expected", expected, "got", got))
		if err := os.WriteFile(in+".gofmt", got, 0666); err != nil {
			t.Error(err)
		}
	}
}
```

The golden discipline in one screen: `-update` rewrites the golden (but refuses
to destroy an input), failures print a unified diff — never `%q` dumps — and the
got-bytes land beside the fixture as `.gofmt` debris for inspection. Changing
the formatter MUST edit a golden; that friction is the feature.

gofmt_test.go lines 138-169:

```go
// TestRewrite processes testdata/*.input files and compares them to the
// corresponding testdata/*.golden files. The gofmt flags used to process
// a file must be provided via a comment of the form
//
//	//gofmt flags
//
// in the processed file within the first 20 lines, if any.
func TestRewrite(t *testing.T) {
	// determine input files
	match, err := filepath.Glob("testdata/*.input")
	if err != nil {
		t.Fatal(err)
	}

	// add larger examples
	match = append(match, "gofmt.go", "gofmt_test.go")

	for _, in := range match {
		name := filepath.Base(in)
		t.Run(name, func(t *testing.T) {
			out := in // for files where input and output are identical
			if strings.HasSuffix(in, ".input") {
				out = in[:len(in)-len(".input")] + ".golden"
			}
			runTest(t, in, out)
			if in != out && !t.Failed() {
				// Check idempotence.
				runTest(t, out, out)
			}
		})
	}
}
```

Expectations live beside fixtures: each `testdata/*.input` pairs with a
`*.golden` sibling, per-fixture flags ride INSIDE the fixture as a `//gofmt`
comment, and the package's own sources join the corpus as large examples. The
cheapest strong property a formatter can pin closes the loop: re-run the
formatter on its own golden and demand a fixed point (`runTest(t, out, out)`).

testdata/rewrite1.input (lines 1-9; the paired rewrite1.golden is identical with
`Foo` rewritten to `Bar`):

```go
//gofmt -r=Foo->Bar

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

type Foo int
```

One case is one reviewable pair of files; the flag that produced the golden sits
on line 1 of the input. Nothing about the case lives at helper-call distance in
Go code.
