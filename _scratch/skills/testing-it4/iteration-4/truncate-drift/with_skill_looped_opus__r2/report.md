# truncate tests

Wrote external `truncate_test.go` from the doc comments first, then revised it against a maintainer critique bound to skill-v4.

Structure: a composed `Example` (breadcrumb head+…+tail), per-symbol `ExampleHead`/`ExampleTail`, and one merged Head/Tail table proving runes-not-bytes, over-limit-unchanged, and the exact/empty/zero/negative edges.

Drift found: both docs promise "a negative n is treated as zero: returns the empty string," but Head lacks Tail's `n <= 0` guard and hits a negative slice bound, so `Head(s, n<0)` panics. I sided with the prose (Head/Tail symmetry says the guard was forgotten): the negative row holds Head to its promise and fails today, with a Head-specific `headRecovered` reporting the panic so the rest stays green. Production code left untouched for the owner.

## Critique points applied

- (HIGH) Prose sub-test names: dropped `t.Run` and the `name` field entirely; one continuous loop. The `Head(%q,%d)`/`Tail(%q,%d)` failure messages name the inputs themselves.
- (HIGH) Generic `wantEmpty` masking the function: deleted. Replaced with a Head-specific `headRecovered` that calls `truncate.Head` directly (no func-value indirection, no hand-wired identity string) and recovers only where Head actually panics. Tail, which cannot panic, is now a plain compare.
- (MEDIUM) Standalone negative test: folded into the merged table as a single row `{"café", -5, "", ""}`. Dropped the redundant ASCII negative input — the `n <= 0` guard is byte-content-blind, so one input discriminates. Deleted `TestNegativeLimitYieldsEmptyString`.
- (MEDIUM) Example cheaped out: restored per-symbol `ExampleHead`/`ExampleTail` for pkgsite attribution; kept the composed `Example` as the representative call pattern. `ExampleHead` now shows the doc's `héllo`→`hé` and the over-generous-returns-unchanged promise.
- (LOW-MED) Unexplained call-site literals: the composed `Example` now explains why `4`/`5` (they land on the "café" and "señor" segments).

## Critique point partially rejected, with reason

- Critique #4 also wanted an example demonstrating the negative→empty promise. Not added as an example: Head panics on negative n (the drift), so a runnable example asserting that behavior would panic and fail. Examples can only assert passing behavior, so the negative promise is held by the table row instead. The over-generous and exact-length promises are shown in examples as asked.

Status: `gofmt` clean, `go build ./...` OK, `go vet` clean, all three examples pass; `go test ./...` FAIL — exactly one assertion, the negative-limit Head row, by design.
