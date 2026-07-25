# Engineer's note: truncate tests

Wrote `truncate_test.go` go-doc-first (external `truncate_test` package): a runnable display example, one table driving Head and Tail with hard-coded rune-count expectations, and a dedicated negative-count test that recovers an expected panic so it reads as a failure, not an aborted run.

Finding (drift): the docs promise "a negative n is treated as zero" for both functions; Tail guards `n <= 0`, but Head is missing the same guard and panics (`r[:n]`, truncate.go:23). The asymmetry looks like an accidental omission rather than a deliberate contract change, so the test sides with the prose. Owner's call: add Tail's guard to Head (one line) or amend Head's doc, then this test pins whichever contract you choose. The finding and this inference are now also recorded in the doc comment of TestNegativeCountActsAsZero, so the red suite explains itself in-repo; whoever lands this should carry the same rationale into the commit body.

Status: `gofmt -l` clean; `go vet ./...` clean; `go test ./...` fails exactly one sub-test, deliberately: TestNegativeCountActsAsZero/Head reports `Head("héllo", -1) panicked: runtime error: slice bounds out of range [:0] with capacity 32, want ""`. Example_display, TestTruncate, and TestNegativeCountActsAsZero/Tail pass.

## Critique response (round 2)

All five findings survived judgment; each traces to the skill or its references, and none conflicts with the package prose.

1. Drift not carried by the committed artifact — applied. TestNegativeCountActsAsZero's doc comment now states that Tail keeps the promise while Head panics today, that this reads as accidental omission, and that the test fails until the owner guards Head or amends the doc. The commit-body half applies at landing time; the paragraph above is the body's content.
2. Example body empty of comments — applied. Each call site now carries a user-facing comment: the rune-not-byte budget on the Head cut, the keep-the-end scenario on the Tail cut, and the generous-limit affordance on the new third line.
3. Func-valued table + name-string wiring — applied, choosing the sub-test remedy over keeping the helper: "Head" and "Tail" read like Go symbol names, and inlining the recover puts `truncate.Head(...)` visibly in the test body instead of behind a `cut` parameter and a `name` string. The eight repeated lines are the cost tables-and-subtests.md explicitly accepts. The emptyOnNegative helper is gone.
4. Example skips the documented generous-limit pattern — applied. A third Println shows a short value passing through a limit of 40 unchanged, with its own comment; the prose headlines this affordance twice, so the example was underselling it.
5. TestTruncate doc comment narrates structure — applied. "One table drives both cuts" and the hard-coded-expectations sentence are gone; only the rationale remains (byte-index slicing fails the multi-byte cases outright instead of silently mangling the cut).
