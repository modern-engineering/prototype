# Maintainer critique — truncate-drift / with_skill_looped_sonnet__r2

Scope: `outputs/truncate_test.go` and `report.md`, reviewed against
`skill-v4/SKILL.md` + its exemplar and the intent corpus (INTENT.md,
feedback-3.json, TESTS.md). `truncate.go` was not opened; the exported
surface came from `go doc -all` on the copied module. I built, vetted,
linted, and ran the suite to check every claim below (`go build ./...`,
`go vet ./...`, `golangci-lint run ./...`, `gofmt -l .` all clean; `go
test ./...` fails exactly as `report.md` describes).

Overall this is a strong submission — the merged Head/Tail table, the
head-only diagnosis of the drift, the example-then-typical-then-edge-case
file order, and the named struct-literal fields are all textbook
executions of standing guidance. The findings below are what stops me
from merging as-is, ranked by severity.

## 1. `truncate_test.go`: `Example_preview` has zero in-function comments (refuse to merge)

```go
func Example_preview() {
	name := "café society"
	if short := truncate.Head(name, 4); short != name {
		fmt.Println(short + "…")
	}
	// Output:
	// café…
}
```

The doc comment above the function is fine, but the body carries not one
comment. This is the single most repeated note across every prior
`with_skill` review in the corpus (limiter-fable, limiter-opus,
limiter-sonnet, config-fable all dinged the exact same gap): in-function
example comments are "prime real estate" and are *mandatory*, because
they're the only place non-Go contract parts get repeated at the call
site for a pkgsite reader. Here there's a genuinely non-obvious
construct — `if short := truncate.Head(name, 4); short != name` — a
length check that exists purely to decide whether to print the ellipsis,
and is silently always-true for this input. Nothing tells the reader why
the guard is there, why `4`, or what it means for the branch not to fire.
That's exactly the situation the doctrine calls out: this needed a
comment and doesn't have one.

## 2. `truncate_test.go`: the suite's own design hides `Example_preview` from every ordinary `go test` run until the bug is fixed

I confirmed this by running it:

```
$ go test ./... -v
=== RUN   TestTruncate
--- PASS: TestTruncate (0.00s)
=== RUN   TestNegativeNIsTreatedAsZero
--- FAIL: TestNegativeNIsTreatedAsZero (0.00s)
panic: runtime error: slice bounds out of range [:0] ...
FAIL	example.invalid/truncate	0.325s
```

`Example_preview` never even gets an `=== RUN` line — Go's test binary
always runs Tests to completion before it starts Examples, and
`TestNegativeNIsTreatedAsZero`'s unrecovered panic kills the process
first. Leaving that panic in is the right call (the corpus explicitly
praises exactly this: "I appreciate very much keeping the test failing.
It would have been a mistake to bless the panic"), so I'm not asking for
a recover or a `t.Skip` here — either would paper over a real bug, which
the skill is explicit about not doing. But the consequence deserves more
than a footnote: as delivered, the package's one *mandatory* runnable
example has no standing verification in CI for as long as this bug is
open. `report.md` mentions a one-off manual check
(`go test -run Example_preview`), which I confirm passes, but a manual
check done once at authoring time is not a safeguard for whoever touches
this package next. At minimum this deserves a line in the test file or
report calling out that `Example_preview` is currently unverified by the
default `go test ./...` invocation.

## 3. `truncate_test.go`: `TestNegativeNIsTreatedAsZero`'s Head assertion is dead code today

```go
if got := truncate.Head("hello", -1); got != "" {
	t.Errorf("Head(%q, %d) = %q, want %q per doc", "hello", -1, got, "")
}
```

The panic happens *inside* `Head`, on the call itself — confirmed from
the crash's own stack trace (`truncate.Head(...)` at `truncate.go:23`,
called from `truncate_test.go:58`, before the comparison). That means the
`!= ""` check and the crafted `t.Errorf` message never run today; the
test currently fails purely via an unhandled runtime panic, not via the
comparison it's written to look like it's making. That's fine as
forward-looking code — it becomes the real assertion the day `Head` gets
its guard — but as committed right now it reads as an active check when
it's actually inert. Worth a one-line acknowledgment in the comment
(something like "once Head is fixed, this comparison becomes live";
right now the panic is the entire test).

## 4. `truncate_test.go`: the test's name asserts a property the test itself disproves for `Head`

`TestNegativeNIsTreatedAsZero` states a package-wide claim. The whole
point of the function, though, is that this claim is false for `Head`
today — that's the drift being surfaced. Anyone scanning a bare
`go test -list` or a CI failure list sees `FAIL:
TestNegativeNIsTreatedAsZero` with no signal that this is the one test
expected to stay red pending a known, already-diagnosed bug, versus a
fresh regression. The in-source comment explains this once you open the
file, but the name itself oversells the property it's naming.

## 5. Delivered artifact: no commit trail carries the drift finding's rationale

As delivered, `outputs/` is a flat pair of files with no commit history.
The corpus is explicit that a non-trivial finding's author-context
inference (oversight vs. deliberate contract change, "promise it or
refuse to harden") belongs in a commit body, and that non-trivial tests
land as fine-grained commits carrying that context — not only in a
side-channel `report.md` that a future `git log` reader will never see.
The reasoning itself is good (I'd sign off on the "oversight, not
deliberate" call); it's the artifact shape that's missing. Flagged with
the caveat that I can't tell from what was handed to me whether this
reflects the run's actual process or just how the outputs were
collected.
