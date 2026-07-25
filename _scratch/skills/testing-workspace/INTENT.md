# Maintainer intent corpus — index and in-chat rulings

The authoritative sources, in order of arrival. Skill revisions and the
maintainer-surrogate calibrate against ALL of these; nothing here is loaded
by the skill itself.

## On-disk sources

- `feedback.json` — round-1 per-run grading (iteration-1 outputs).
- `feedback-2.json` — round-2 per-run grading (richest on style: hard-coded
  values, typed errors, b.Loop, package-property naming, examples).
- `feedback-3.json` — round-3 per-run grading (richest on audience doctrine,
  anti-slop comments, flows-over-pinning, helpers/sub-test placement).
- `~/Code/notorious-ai/claude-plugins/golang-dev/TESTS.md` — the maintainer's
  accumulated testing notes (the skill's original seed; includes sub-test
  Field=Value naming, case-construction side functions, helper types with
  *testing.T fields, concurrent-safe *testing.T in goroutines, WaitGroup
  against leaks, package code-smell signals).
- `research/attention-report.md` — evidence-based restructuring plan (v4).
- `iteration-3/surrogate-review.json` — the calibration experiment's output.

## In-chat rulings not stored elsewhere (2026-07-22/23, quoted or tight)

1. GOLDEN HIERARCHY (already in skill v3): prose most sacred → exported
   surface → code behavior → unexported comments → existing tests (least).
   Drift is a finding, never silently absorbed; ask, else side with prose
   unless history shows deliberate code movement with lagging prose (then
   fix the prose too). go-doc-first blind mode; blindness needs no curation
   because go doc output is machine-generated.
2. HELPERS, corrected: "test helpers are a good thing. However, it must not
   be *abused*." In trivial packages only panic-recovery earns one; complex
   packages legitimately use case-construction side functions for tabular
   values. Find proper exemplars in stdlib + golang.org/x. The advanced
   method-on-case pattern exists but is for packages that are much harder to
   test; "the line is usually drawn when the test helpers require tests
   themselves" (→ internal testing package).
3. FILE ORDER, softened: "build complexity as we read the file" — examples
   first (if not in a standalone example_test.go), then most typical call
   patterns → most complex. Helpers placed AFTER their first usage; don't be
   forceful beyond that. A convoluted helper opening the file kills the
   review ("I immediately don't want to continue reading").
4. AUDIENCE DOCTRINE (the round-3 center): example comments (doc AND
   in-function) speak to USERS on pkgsite and about the ATTRIBUTED SYMBOL /
   scenario — never narrate the code, never address maintainers, never
   describe the example function itself. Test function comments speak ONLY
   to maintainers. In-function example comments are prime real estate:
   repeat non-Go contract parts at call sites, explain non-meaningful
   literals (`New(1, 2)` needs a comment; `Every(time.Second)` wouldn't).
   A representative example of the anticipated call pattern is MANDATORY
   per package (e.g. Example_throttle; Wait-after-Allow-false).
5. ANTI-SLOP COMMENTS: never explain conventions, test ordering ("a loser
   tell-sign of an LLM agent leaking its prompt"), synctest mechanics, or
   "this test pins the contract" (true of all tests). "IF YOU HAVE NOTHING
   GOOD TO SAY THEN DON'T SAY NOTHING." Rationale doc comments are for
   NON-TRIVIAL tests only — the blanket every-function demand (my grader
   assertion) was wrong and likely caused comment spam.
6. FLOWS OVER PINNING: "thinking as users pins **flows** — that is by far
   superior"; "pinning" language breeds mechanical, fine-grained,
   symbol-scanning suites. Coverage "in the nebulous sense" spans the whole
   suite's code, not test titles. Reports' findings deserve author-context
   inferences (promise it? refuse to harden?) — that context belongs in
   commit messages; non-trivial tests land as fine-grained commits with
   contextual bodies (ties to golang-dev:committing).
7. SECONDARY REVIEW GOAL — package smells through tests: if tests repeatedly
   need a comment translating an API value (limiter's rate→frequency),
   production users will too → flag the package for rethink (TESTS.md
   "Package code-smell signals"). Also: const/var blocks with interrelated
   values need a comment explaining the interplay for auditors.
8. CONCURRENCY refinements: goroutines outside synctest take a WaitGroup;
   assert inside goroutines (*testing.T is concurrent-safe) instead of
   collecting values; no statistical race-hunting via unrealistic patterns;
   docs OMITTING concurrency ≠ negating it for an obviously-concurrent
   package — suite-wide synctest use IS the concurrency testing; panics are
   valid test failures (recovery usually futile unless expected — for a
   blocking bug-panic: comment out to see work through, restore at
   hand-off); test technically-accepted invalid inputs (zero-ticker rate).
9. STYLE MISC: reuse existing fixture fields over purpose-built ones
   (elegance via reuse); no global test-case slices; named case types (even
   in-function) fine, doc-commented when multi-use; name struct fields
   beyond 2-3 fields (golangci-lint default); sub-tests only when names read
   like top-level Go names — prose-like names mean one continuous test;
   name inputs instead of dumping multiline strings into logs; don't shorten
   examples ("Why cheap out on me?"); consider merged case structs when two
   functions mirror each other ({"hello", 3, "hel", "llo"}); Go 1.26
   errors.AsType[T] (verified api/go1.26.txt #51945) preferred over
   errors.As ceremony; local-package conventions rule — respect prevailing
   style of PRE-EXISTING packages only (tricky in review: fresh context
   makes everything look pre-existing).
10. DEVELOPMENT-LOOP PHILOSOPHY (scope TBD): testing is the author's
    hat-switch into the user role — "the most left-shifted feedback loop";
    packages are developed WHILE developing their tests, one package at a
    time; agents need non-committable playgrounds (throwaway tests, scratch
    mains, using the tool) for bug-hunting; candidate: a review-experienced
    agent distills typical user flows/stories BEFORE the first tests as the
    coding agent's starting point (or leave spontaneous). Mini-loop critic:
    fresh-context reviewer bound to the skill; two-evaluator variant — main
    evaluator sees ONLY the exported interface, a second may read the
    implementation as a best-effort end pass.
11. GRADER FIXES OWED (iteration 4): non-trivial-only doc-comment assertion;
    helpers-not-abused instead of panic-table-only; drift eval's
    surfaced-the-drift assertion did not discriminate (all configs pass when
    report.md is required) — differential is committed-artifact shape;
    units every-function artifact (pre-existing ExampleParseSize).
12. SURROGATE CALIBRATION, round 3: directionally right on config/units;
    over-rated limiter-fable and truncate-fable because round 3 applied NEW
    criteria (audience, helper placement) absent from its corpus — the
    surrogate enforces revealed intent only; the corpus is not saturated.

## Maintainer answers (2026-07-23, pre-compaction)

13. HARNESS EXEMPLAR SOURCE: `github.com/go-digitaltwins/go-digitaltwins`
    (the maintainer develops and maintains it; TESTS.md names its
    `enginetest` beside httptest/analysistest). Study it for the harness
    doctrine — the one skill area with zero eval evidence.
14. DEVELOPMENT-LOOP SCOPE, decided: a SEPARATE sibling golang-dev skill
    (hat-switching philosophy, playgrounds, flows-first planning agent,
    mini-loop critic ceremony); the testing skill stays strictly about
    committed tests.
15. STRUCTURAL DIRECTION for v4, decided: SKILL.md focuses on the COMMON
    canon — navigating the model's latent training space — while the more
    complex scenarios from the feedback rounds and transcript move to
    specialized files loaded via progressive disclosure. Code-smell signal
    list considered covered for now.
16. The maintainer HAND-EDITED skill v3 (SKILL.md, YAML-folded frontmatter,
    refined prose) and evals/evals.json after round 3 — the current files
    are authoritative over any fleet-written version; diff against
    workspace snapshots to see their touch, never revert.
17. CANON EPISTEMOLOGY (2026-07-23): the Go team's practice (httptest,
    analysistest and kin) is the SOURCE of harness directives; the
    maintainer's own `enginetest` (local clone:
    `~/Code/go-digitaltwin/go-digitaltwin/enginetest`) is "a good example on
    how to follow the directives that should come from learning the Go
    team's practice" — a worked demonstration of compliance, never a canon
    source: "You must not learn from my code any more than you should from
    theirs." Every such package is a whole world; the intent is the same.
    The same epistemology governs the skill's own bundled exemplar: its
    authority derives from embodying canon + intent, not from being canon.
    [Done 2026-07-23: harnesses.md rewritten to 75 lines — directives
    sourced from httptest/fstest/iotest/slogtest/analysistest each with a
    one-clause why; enginetest appears only in a closing "The derivation
    transfers" section, explicitly "not a source", citing its four moves as
    consequences of following the canon; plural spelling fixed.]
18. EXEMPLAR PROVENANCE (2026-07-24, after the maintainer caught the v4
    exemplar being a synthetic package built FROM eval outputs — "stupid as
    fuck"): for the current scope, before any complex packages exist to
    train for, the ONLY samples allowed in the skill's exemplars are
    SNIPPETS FROM THE GO TEAM'S PACKAGES (standard and golang.org/x
    extended), verbatim and attributed, with OUR annotations to few-shot
    from. Never synthetic agent-written code; never anything derived from
    eval fixtures or eval outputs; eval fixtures stay domain-distant from
    whatever the exemplars show. (Corollary of ruling 17's epistemology,
    one level down.)

## Round-4 rulings (2026-07-25, feedback-4-best.json + chat)

19b. ARCHETYPE RECOGNITION: lifecycle (and kin) have MANY signature shapes
    (Close() error, Stop(), Shutdown(context.Context) error, inconsistent
    names); recognition is LLM language-reasoning over docs + signatures —
    "we don't really need to peek at the code because docs and signatures
    are more than enough for the vast majority of cases."
20. RUNNABLE-EXAMPLES DOCTRINE (own reference file): evaluator VALIDATES the
    expected Output demonstrates what the example claims (intent signals:
    doc, name, inline comments); every character serves package USERS (doc,
    code, comments, Output all reach pkgsite); speak about USAGE of the
    package, never about exported symbols (single-symbol demos still lead
    into usage as importers); // Unordered Output: fits very specific cases
    only, not a concurrency magic bullet (seek std/x canon); sleeps valid
    but never as a sync primitive for concurrent-heavy patterns, complete
    within ~1s; trivial panic tests may be runnable examples (downside:
    pins panic text; works for some libraries); SUBTLE: examples guarantee
    output, so the evaluator must ensure ALL possible failures manifest as
    output other than exactly expected.
21. EXACT BUBBLE TIMING: inside synctest bubbles, overshooting is "a noob
    move... a hopeful approach to timing" — use the most accurate timing:
    the exact moment (sleep the full TTL) or the very next quantum
    (duration+1). Wide margins remain correct OUTSIDE bubbles (examples).
22. STYLE: unrolled or named cases when values are semantically meaningless
    (name them "negative"/"zero", two explicit t.Run calls or name+value
    struct); concurrency stress uses generously high goroutine counts
    (64/128/1024) since 16 may not contend on some hardware.
23. GOROUTINE-LEAK QUESTION (open, "check it please"): cache fixture's
    Close guarantees janitor exit but does not WAIT; does a synctest bubble
    exiting with a live-but-exiting goroutine fail? Verify empirically;
    make the evaluator flag potential leaks; find an elegant leak test or
    consult the maintainer if none exists.

