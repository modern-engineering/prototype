# go/parser — expectations at line granularity, inside the fixture

Source: `go/parser/error_test.go` and `go/parser/testdata/commas.src` (Go
development tree, 1.27 dev, 2026-06). Verbatim; BSD-3-Clause, © The Go Authors.

error_test.go lines 5-21 (the file-header comment):

```go
// This file implements a parser test harness. The files in the testdata
// directory are parsed and the errors reported are compared against the
// error messages expected in the test files. The test files must end in
// .src rather than .go so that they are not disturbed by gofmt runs.
//
// Expected errors are indicated in the test files by putting a comment
// of the form /* ERROR "rx" */ immediately following an offending token.
// The harness will verify that an error matching the regular expression
// rx is reported at that source position.
//
// For instance, the following test file indicates that a "not declared"
// error should be reported for the undeclared variable x:
//
//	package p
//	func f() {
//		_ = x /* ERROR "not declared" */ + 1
//	}
```

A bespoke few-hundred-line harness, written once, that turns the parser's user
contract into a corpus: the ERROR comment's own source position IS the expected
diagnostic position, so nobody hand-counts line:col at helper-call distance. The
header comment teaches the whole convention, with a worked example, before any
code.

error_test.go lines 61-67:

```go
// ERROR comments must be of the form /* ERROR "rx" */ and rx is
// a regular expression that matches the expected error message.
// The special form /* ERROR HERE "rx" */ must be used for error
// messages that appear immediately after a token, rather than at
// a token's position, and ERROR AFTER means after the comment
// (e.g. at end of line).
var errRx = regexp.MustCompile(`^/\* *ERROR *(HERE|AFTER)? *"([^"]*)" *\*/$`)
```

The expectation language is one regular expression with two documented
refinements (`HERE`, `AFTER`) for positions a plain comment cannot express. Grow
markers only when a real fixture needs them.

testdata/commas.src (complete file):

```go
// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Test case for error messages/parser synchronization
// after missing commas.

package p

var _ = []int{
	0/* ERROR AFTER "missing ','" */
}

var _ = []int{
	0,
	1,
	2,
	3/* ERROR AFTER "missing ','" */
}
```

The fixture states its purpose in two lines, then the offending token and its
expected error sit on the SAME line. Sealing is bidirectional in such harnesses:
an unmatched expectation fails, and so does an unexpected diagnostic — the
corpus is the contract, in both directions.
