# Evidence — what was measured, so it is not re-litigated

Paths are relative to `_scratch/` on branch `worktree-master-tests-orig` (see
README); `it4/` is `skills/testing-it4/`, `ws/` is `skills/testing-workspace/`.

## The four rounds

Each round ran the evals against the skill and a no-skill baseline of the same
model. Assertion sets GREW as rulings were absorbed, so ratios compare within a
round, never across. Best with-skill cell vs baseline, from `grading.json` under
`ws/iteration-1..3/` and `it4/iteration-4/`:

| round | skill | limiter       | config review | units      | truncate   | cache               |
| ----- | ----- | ------------- | ------------- | ---------- | ---------- | ------------------- |
| 1     | v1    | 5/6 vs 4/6    | 7/8 vs 5/8    | 4/5 vs 4/5 | —          | —                   |
| 2     | v2    | 8/9 vs 5/9    | 9/9 vs 4/9    | 5/5 vs 4/5 | —          | —                   |
| 3     | v3    | 9/10 vs 3/10  | 9/9 vs 4/9    | 4/5 vs 3/5 | 6/7 vs 4/7 | —                   |
| 4     | v4    | 14/14 vs 2/14 | 11/11 vs 5/11 | 6/6 vs 5/6 | 9/9 vs 5/9 | 15/15 (no baseline) |

The cache fixture is round-4-new, so nothing prior contaminates it. The
maintainer's prose moved further: round 1 "MUCH WORSE! ... I had no will power to
continue reading" (`ws/feedback.json`), round 3 "convoluted `call` helper -
terrible. I immediately don't want to continue reading" (`ws/feedback-3.json`),
round 4's best run only refinements — bubble timing, unrolled panic cases, 16
goroutines maybe not contending (`ws/feedback-4-best.json`).

## A/B verdict: the review loop wins 8 of 9 cells

Round 4 ran a grid: 3 evals (cache, limiter, truncate) x 3 models (fable, opus,
sonnet) x 3 replicates x 2 arms — plain writer versus writer plus the skill's own
review pass — rated 1-10 by the maintainer surrogate
(`it4/iteration-4/surrogate-review*.json`). Cell means: looped 8.2 against plain
7.4, at or above plain in eight of nine cells. The loss is limiter x sonnet, 6.67
to 6.33, caused by one run rated 4: the reviser EDITED THE PACKAGE, adding a
guard to `New` and rewriting the exported doc, justified as "the fixture itself
is the strongest evidence of the maintainer-intended fix" — the incident behind
SKILL.md's owner-only line. Exactly one of 28 re-runs reports `SOURCE: MODIFIED`
(`it4/verify-supplement/limiter-write-tests-with_skill_looped_sonnet__r2.log`),
and every deliberate red row is deterministic across those logs under `-count=30`
and `-race -count=5`. The loop amplifies weak models both ways: sonnet looped
spans 6/9/7 on cache and 8/4/7 on limiter, fable looped 9/9/9, 10/9/10, 9/9/10.

## Token forensics: routing works

From the run transcripts, in `it4/research/skill-token-forensics.md`. Per agent
in the routed round: writer ~5.4k tokens (SKILL.md, router README, two matched
exemplar files, one reference, one grep-extracted sample), reviser ~5.1k, critic
~13.1k reading the corpus whole as REVIEWING mode orders, grader 0 — it never
opens the skill, so impartiality holds by construction. Exemplar payload: 12.1k
characters for the writer against the critic's 35.7k, 34% of the corpus.
Pre-retrofit writers paid 10.1-11.7k for that corpus
(`it4/exemplar-pre-retrofit/`); the earlier small corpus cost ~4.2k.

## Exemplar provenance, corrected

The v4 exemplar began as a synthetic package built from eval outputs; the
maintainer rejected it (INTENT ruling 18): only verbatim attributed Go-team
snippets ship, and fixtures stay domain-distant from them. It was measurable —
that exemplar was a token bucket nearly identical to the limiter fixture, so "all
with_skill limiter suites are near-verbatim exemplar copies and that eval now
discriminates only on what each run did beyond the copy"
(calibration field, `it4/iteration-4/surrogate-review.json`). Honest-corpus
spot-check on the cache fixture vs the same models' synthetic-exemplar replicates:
fable 15/15 against a median of 14, opus 14/15 against 12, sonnet 13/15 against 12
(`it4/iteration-4/cache-write-tests/with_skill_*__snippets/grading.json`).

## Graduation on real code

The skill reviewed the pre-remaster tree in four scoped passes against the
`_scratch/TESTS-SURVEY.md` audit as ground truth (`it4/graduation/`). 72
findings: 30 familiar (42%), 42 novel and verified against the tree before being
granted, 0 hallucinations, 8 audit findings missed. Novelty clusters where a
package-level audit structurally cannot see: doc-vs-code drift, documented
clauses with zero coverage, self-test holes in harnesses the audit
blanket-praised (A11 and S7, found independently by two reviewers).

## Ruling 23 settled: bubble exit is a free leak detector

A `Close` that signals its janitor without waiting does NOT fail a synctest
bubble: the bubble waits for every goroutine in it to exit. A goroutine nobody
signals fails with `panic: deadlock: main bubble goroutine has exited but blocked
goroutines remain`. Reproducible under `evals/synctest-leak-probe/` on Go 1.26.5.

## Implementation swap: the suite pins the contract

`it4/research/cache-chan/cache.go` re-implements the fixture's expiry with
channel-passed map ownership, not a mutex. The winning run's test file is
byte-identical between `it4/research/cache-chan/cache_test.go` and
`it4/best-run-review/cache-write-tests--looped-fable-FINAL/with_skill/outputs/cache_test.go`,
and `go test` passes against both. That run also left the fixture untouched.
