# strings — one runner for a function family; Examples carry the trivial cases

Source: `strings/strings_test.go` and `strings/example_test.go` (Go development
tree, 1.27 dev, 2026-06). Verbatim; BSD-3-Clause, © The Go Authors.

strings_test.go lines 60-74 (the table runs to line 220; elided):

```go
type IndexTest struct {
	s   string
	sep string
	out int
}

var indexTests = []IndexTest{
	{"", "", 0},
	{"", "a", -1},
	{"", "foo", -1},
	{"fo", "foo", -1},
	{"foo", "foo", 0},
	{"oofofoofooo", "f", 2},
	{"oofofoofooo", "foo", 4},
	{"barfoobarfoo", "foo", 3},
	// [... ~140 further cases elided; clusters carry comments like
	//  "cases with one byte strings - test special case in Index()" ...]
```

strings_test.go lines 222-238:

```go
// Execute f on each test case.  funcName should be the name of f; it's used
// in failure reports.
func runIndexTests(t *testing.T, f func(s, sep string) int, funcName string, testCases []IndexTest) {
	for _, test := range testCases {
		actual := f(test.s, test.sep)
		if actual != test.out {
			t.Errorf("%s(%q,%q) = %v; want %v", funcName, test.s, test.sep, actual, test.out)
		}
	}
}

func TestIndex(t *testing.T)     { runIndexTests(t, Index, "Index", indexTests) }
func TestLastIndex(t *testing.T) { runIndexTests(t, LastIndex, "LastIndex", lastIndexTests) }
func TestIndexAny(t *testing.T)  { runIndexTests(t, IndexAny, "IndexAny", indexAnyTests) }
func TestLastIndexAny(t *testing.T) {
	runIndexTests(t, LastIndexAny, "LastIndexAny", lastIndexAnyTests)
}
```

A pure-function bag is the one API shape where tests may name functions — and
even then the whole family shares one runner and one case shape, so each
`TestIndex`-style function is a one-liner binding a function to its table. The
case-ID (`funcName` plus inputs) appears in every failure message, so a plain
loop beats `t.Run` ceremony. Do not write a fresh loop body per function.

example_test.go lines 94-100:

```go
func ExampleCount() {
	fmt.Println(strings.Count("cheese", "e"))
	fmt.Println(strings.Count("five", "")) // before & after each rune
	// Output:
	// 3
	// 5
}
```

The empty-separator edge case lives in a runnable Example, not a test: it
asserts (via Output) AND teaches on pkgsite. The in-function comment
`// before & after each rune` speaks to USERS about Count's documented contract
— it explains a non-obvious result, never the example's own mechanics. These
examples carry no doc comments; when you add one, it must not open with the
function's own name (see the encoding/json exemplar; bufio's
`ExampleWriter_ReadFrom` doc comment is the violation NOT to copy).

strings_test.go lines 1541-1551:

```go
func BenchmarkReplace(b *testing.B) {
	for _, tt := range ReplaceTests {
		desc := fmt.Sprintf("%q %q %q %d", tt.in, tt.old, tt.new, tt.n)
		b.Run(desc, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				Replace(tt.in, tt.old, tt.new, tt.n)
			}
		})
	}
}
```

The modern benchmark shape: reuse the correctness table, name sub-benchmarks by
their inputs, `b.ReportAllocs`, and `for b.Loop()` — no `b.N`, no timer resets,
no assertions smuggled in.
