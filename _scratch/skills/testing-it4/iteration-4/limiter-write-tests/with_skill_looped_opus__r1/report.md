# limiter tests

`outputs/limiter_test.go` is an external `limiter_test` package. It carries a runnable `Example_throttle` (the mandatory representative Allow-then-Wait call pattern), a `synctest` whole-lifecycle scenario, a `synctest` zero-burst scenario for the documented burst-0 mode, and a `New` panic table with a `mustPanic` helper.

## What changed this pass

The maintainer-critic reviewed the prior version. The one edit is to the withheld large-rate case in the panic table.

- Deleted the seven-line comment that narrated the authoring process and pointed at `report.md`. A committed `_test.go` speaks only to maintainers and only about the contract.
- Reframed the remaining note from limiter's own prose: `New` promises a panic only for a non-positive rate, so a large rate that crashes is drift from the contract. The prior note carried the exemplar's `1e9` ceiling framing, which limiter's prose never states.
- Kept the case commented (not deleted) with a tight technical reason. This follows the concurrency reference: comment a bug-triggering case out to see the work through and never drop it silently.

## Findings for the author

Large-rate crash (the withheld case). A rate above 1e9 floors `time.Second / time.Duration(rate)` to zero. `refill` then calls `time.NewTicker(0)`, which panics on the background goroutine. No caller can recover it, so the process crashes rather than `New` panicking. limiter's `New` prose promises a panic only for a non-positive rate, so per the golden hierarchy this is drift: the code violates what the prose implies. Author decision needed. Add an upper-bound guard so `New` panics cleanly (then the table pins it), or document a ceiling in the prose (then the test asserts the documented ceiling). I cannot pick for the author, and I will not change production behavior silently, so the case stays visible and commented. In a real repo this inference rides the commit body per the committing convention, not the test file.

Exported prose defect (package, not tests). `go doc` shows `New`'s doc as "The bucket starts full." for a type named `Limiter` in package `limiter`. This is copy residue in the sacred part of the contract. A go-doc-first reader trips over it. Flagged, not edited: my mandate this pass is the tests. `refill` also says "the bucket's capacity", but that comment is unexported and off go doc.

## Critique points applied vs rejected

- Applied (finding 1, HIGH): removed the `report.md` pointer and the authoring-process narration from the test comment.
- Applied (finding 2, HIGH): dropped the exemplar's `1e9`-ceiling framing; the note now reasons from limiter's prose, so it reads as surfaced drift, not a leftover from the exemplar's contract.
- Applied (finding 5, MEDIUM): surfaced the "bucket starts full" prose defect as the headline package finding above.
- Rejected (finding 4): keep the case commented rather than resolve it now. I have no author to ask and cannot add a production guard within a test-writing pass. The concurrency reference and the maintainer's own guidance bless comment-out-with-a-clear-note as the professional move when no fix is available; that is not dropping it silently.
- Rejected (finding 3): I did not rewrite `Example_throttle` and `TestThrottleLifecycle` to look less like the exemplar. limiter and the exemplar's bucket are the same API and implementation, so contract-accurate tests are near-identical by construction. Imitation is the skill's method; rewriting correct tests to manufacture difference is cargo-cult divergence and risks introducing errors. The rename to limiter is thorough and the one substantive divergence (no phantom `1e9` case, plus the prose-derived zero-burst scenario) is present.

## Test status (honest, uncached)

- `go build ./...`: clean.
- `go vet ./...`: clean.
- `go test -count=1 ./...`: pass.
- `go test -race -count=1 ./...`: pass.
- `go test -run Example -v`: `Example_throttle` PASS (Output block matches).
