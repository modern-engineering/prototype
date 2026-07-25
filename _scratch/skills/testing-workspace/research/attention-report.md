# Keeping the testing skill effective as feedback accumulates

Research report for the architect and maintainer of `_scratch/skills/testing/SKILL.md`.
Evidence base: iteration-1 (skill v1) and iteration-2 (skill v2, identical to the current file) gradings, the maintainer's verbatim feedback (`feedback.json`, `feedback-2.json`), 12 weak-model run transcripts, and Anthropic's skill-authoring canon.

## Headline findings

1. **Every weak-model run read SKILL.md immediately.** All 12 opus/sonnet runs read it at tool call #1 or #2 of 11-27. "Never read" and "read too late" are ruled out. The failure mode is **read, then overridden by training priors** — and only for a specific class of guidance.
2. **Rules land; taste does not.** Structural, checkable rules (synctest, consolidate per-symbol tests, delete unpromised concurrency, fold a one-symbol test into siblings) flipped from failing to passing when v2 stated them explicitly, across all three models. Taste-level prose (straight-line bodies, doc-comment voice, "write an Example") keeps failing even when the model quotes the skill back: the i2 opus run announced it was following the skill, then opened all seven doc comments with the function's own name — the one thing the skill says never to do.
3. **The skill's "imitation" method is not executing.** No run in either iteration ever opened the canon the skill points at (hash package tests, io's multi/pipe tests, TestComments, cmd/go's testdata). The only canon-chasing observed was `go doc testing/synctest` — API lookup, not exemplar reading. Pointers to external canon function as connotation, not as exemplars. If imitation is the method, the exemplar must live in the skill directory and be named as a read-first step.
4. **The v1→v2 growth already doubled the token bill** (868 → 1,641 words, ~1.1k → ~2.2k tokens) for a net grading gain that came from four specific additions; the rest of the growth is not measurably pulling weight. Feedback-2 contains at least ten new nuances. Appending them at the same rate produces a ~2.5x-of-v1 file whose middle sections are precisely where today's failures cluster.
5. **Part of the "with-skill failure" signal is grader drift, not skill failure.** Two i2 assertions contradict the maintainer's verbatim feedback (details in "Grader vs. maintainer"). Fix the eval before concluding the wording failed.
6. **The maintainer has never seen the units-add-test outputs** ("I don't see any output to review", both iterations, all four configs) and could not see the i2 config-sonnet output either — those runs exist and were graded. A viewer/collection bug is silently deleting a third of the human-feedback corpus.

---

## Workstream 1 — What the canon says, and where our skill stands

Sources: `code.claude.com/docs/en/skills` (fetched), `platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices` (fetched), local skill-creator (`/Users/danielorbach/.claude/plugins/cache/claude-plugins-official/skill-creator/unknown/skills/skill-creator/SKILL.md`).

**Progressive disclosure.** Three levels: metadata (always in context), SKILL.md body (loads on trigger, "keep under 500 lines"), bundled resources (zero cost until read). References must be **one level deep** from SKILL.md — nested references get partially read (`head -100`). Reference files over ~100 lines need a table of contents.

**Budgets.** `description` max 1,024 chars; Claude Code caps the combined description + `when_to_use` listing entry at 1,536 chars and says "put the key use case first". Body: "every line is a recurring token cost" once loaded; it persists across turns and survives compaction only as its **first 5,000 tokens** (25k shared across skills). Our whole skill (~2.2k tokens) fits, but the compaction rule is a position argument: front-load what must survive.

**Trust the model.** Verbatim: "Default assumption: Claude is already very smart. Only add context Claude doesn't already have. Challenge each piece of information: … Does this paragraph justify its token cost?" And: "Set appropriate degrees of freedom" — high freedom (heuristics) where context decides, low freedom (exact steps) only where operations are fragile. The skill-creator adds: "Keep the prompt lean. Remove things that aren't pulling their weight," and "if you find yourself writing ALWAYS or NEVER in all caps … that's a yellow flag — reframe and explain the reasoning." Also directly relevant: "Test with all models you plan to use … What works perfectly for Opus might need more detail for Haiku."

**Examples over description.** The best-practices page is explicit: "Examples convey the desired style and level of detail to Claude more clearly than descriptions alone." This is the canonical warrant for an exemplar corpus.

**One tension to flag.** code.claude.com says "State what to do rather than narrating how or why"; the skill-creator says "Try hard to explain the why … in lieu of heavy-handed musty MUSTs." The maintainer's constraint (guidelines carry their why) sides with the skill-creator, and our data supports it: the two v2 additions that flipped results to passing both carried a why (synctest bubbles — "sleeping inside one is idiomatic"; unpromised concurrency — "doc conventions presume…"). Keep the whys, but one clause each, not paragraphs.

**Where the current skill violates canon:**

- *Trust-the-model:* several guidelines restate what the baseline already does (evidence table below) — pure token cost.
- *Imitation by remote pointer:* canon expects bundled resources for material the model must actually read; our canon lives in GOROOT and the Go wiki and is never fetched.
- *Density:* no hard limit violated (145 lines << 500), but v2 packs ~44 sentences with near-zero filler tolerance for the reader; the failing guidance sits mid-file (lines 60-101), the classic attention sag.
- *Description:* fine. Combined description + when_to_use ≈ 660 chars, third person, triggers listed. No action needed.

---

## Workstream 2 — What our own evals say

Configs per eval: `with_skill` (Fable, strongest), `with_skill_opus`, `with_skill_sonnet`, `without_skill` (Fable, no skill). Iteration-1 ran skill v1; iteration-2 ran v2 (= current file).

### (a) Baseline passes consistently → the model's defaults already cover these (pruning candidates)

| Assertion | Baseline evidence | Skill lines carrying it |
|---|---|---|
| Scenario grain when *writing* tests (no bare TestNew/TestAllow/…; composes symbols) | i1+i2 limiter without_skill PASS ("TestConcurrentUse … composes New, Allow, Wait, and Close") | 49-52 partially |
| Plain-English names without receiver stutter (writing mode) | i2 limiter without_skill PASS ("No TestLimiter* prefix anywhere") | 62-67 |
| Assertions trace to documented promises | i1 limiter without_skill PASS | 38-40 partially |
| Delete dev scaffolds in review | i1+i2 config without_skill PASS (TestDebugDump gone) | 129-131 partially |
| Don't reach into internals | i1+i2 config without_skill PASS | — |
| Preserve good existing tests; new doc comment opens with contract clause (units) | i1+i2 units without_skill PASS | — |
| b.Loop benchmarks (strong model only) | i2 config without_skill PASS | 97-100 |

Caution on two of these: the naming/grain defaults hold for the *strong* baseline; sonnet still misfired in-skill (i2 `TestAllow`). And "no doc comment opens with its own name" passes in baseline write-mode **vacuously** — the baseline mostly writes no doc comments at all. The moment models write comments (which feedback-2 now demands), the default convention "comment starts with the name" reasserts itself: i2 opus and sonnet, with the skill loaded, both regressed on exactly this.

### (b) Failing WITH the skill, by model → wording not landing

| Assertion | Fable | Opus | Sonnet | Reading |
|---|---|---|---|---|
| Straight-line bodies, hard-coded values, no case loops (i2 limiter) | FAIL | FAIL | FAIL | 0/3 with skill (baseline also FAIL). The v2 paragraph (72-76) transcribes feedback-1 almost verbatim and lands nowhere. Partly grader over-reach (see below); the residue is taste that prose cannot carry — every model reaches for a table+`t.Run` for panic cases because that is the most-trained Go pattern. |
| Doc comment must not open with its own name (i2 limiter) | PASS | FAIL (7/7 comments open with the name) | FAIL (3 do) | Weak models only, and only once they write comments at all. Negative rule loses to convention prior; needs the positive form plus an exemplar to imitate. |
| At least one runnable Example (i2) | limiter PASS / config FAIL | limiter FAIL / config PASS | FAIL both | 4/9 with skill across both iterations, 0/3 baseline. Load-bearing but unreliable; guidance sits mid-file (85-91) and reads as commentary, not as an expectation that a committed suite includes examples. |
| Flag string-only errors → typed/sentinel (i2 config REVIEW DEPTH) | FAIL | PASS | FAIL | The doctrine is at line 136-138 (last section). Fable actually rewrote the tests well but never *flagged* the API redesign; sonnet never mentioned it anywhere. Also a harness gap: no report artifact is required, so the flag has nowhere durable to land. |
| External test package (i2 limiter) | PASS | PASS | FAIL (`package limiter`) | The skill never says `package foo_test` explicitly — it must be derived from "exercise each package as its user" (44-47). Fable and opus make the derivation; sonnet doesn't (and the same sonnet chose an external package in the config eval — coin-flip inference). The abandoned draft (`material/SKILL-draft1.md`) said "Prefer `package foo_test`" outright; v1 dropped it. Restore one clause. |
| synctest, no real-time waits (i1, v1 wording) | FAIL | FAIL | FAIL | v1 said only "Prefer testing/synctest … read it for virtual time". All models used synctest but kept real `time.Sleep`. v2's added why ("bubbles run on virtual time, so sleeping inside one is idiomatic and stable", 115-119) fixed it: i2 3/3 PASS. Proof that a one-clause mechanism-why beats a bare preference. |

### (c) Passing only with the skill → the load-bearing set (protect these)

| Guideline (current lines) | Evidence |
|---|---|
| synctest + bubble-sleep idiom + minimal orchestration (115-119) | i2: 3/3 with-skill PASS; baseline FAIL both iterations (real sleeps, polling loop). |
| Delete tests pinning unpromised concurrency; doc-conventions presumption (110-113) | i2 config: 3/3 with-skill PASS **with the reasoning stated**; baseline rewrote the smoke instead of deleting it. New in v2, immediately effective. |
| Add-one-test folds into existing patterns / sibling precedent (54-58) | i1 (absent from v1): 0/3 with-skill, all wrote `TestParseDurationOrDefault`. v2 added the paragraph → i2: 3/3 with-skill PASS; baseline still FAIL. The cleanest wording-works datapoint in the corpus. |
| Consolidate per-symbol grain in review (49-52, 129-132) | Baseline FAIL both iterations; with-skill PASS 5/6 runs. |
| External test package (implicit in 44-47) | Baseline FAIL both iterations; with-skill 5/6 (sonnet miss above). |
| Runnable examples (85-91) | Baseline 0/3 ever; with-skill 4/9. Load-bearing but under-delivered — protect and strengthen. |
| b.Loop (97-100) | i1 sonnet FAIL pre-guidance (real bug: benchmark optimizable away); i2 all PASS. Keep for weak models even though the strong baseline now passes. |

### Adherence forensics (transcripts)

Transcript dirs: `wf_b8d51376-fed` (iteration-1), `wf_d9c77c75-fe2` (iteration-2) under `/Users/danielorbach/.claude/projects/-Users-danielorbach-Code-modern-engineering-prototype/7a979c9d-aba6-4170-a7ba-fed10f6377de/subagents/workflows/`.

**All 12 weak-model runs: skill read at tool call #1 or #2.** Examples: i2 limiter opus (`agent-a7066bc3956e4dc22`) read it 2nd of 13 calls; i2 limiter sonnet (`agent-ac80783572323e65a`) 2nd of 15; i1 limiter opus (`agent-a608b0c058f99277a`) 2nd of 13.

**Case 1 — i2 limiter opus: read, cited, then overridden on the very rule it read.** After reading, it wrote: *"I'm using an external test package (`limiter_test`) to exercise only the exported API as a real user would, `testing/synctest` for deterministic virtual-time control … and `errors.Is` against the sentinel/typed errors rather than string matching"* and later *"under race and constrained CPU … as the skill advises"*. Structural guidance: fully absorbed. Then it opened all seven doc comments with the function's own name (`// TestLimiter walks the typical gateway flow…`, `// TestBurstZeroAdmitsOnlyOnRefill pins…`) and produced no Example. Verdict: **read then ignored — selectively**. The ignored items are exactly the counter-convention taste rules.

**Case 2 — i2 limiter sonnet: read, no visible engagement, defaults win where derivation is needed.** The transcript contains no reasoning text connecting the skill to decisions (read at #2, wrote the test file at #8). It adopted synctest correctly but wrote `package limiter`, a `TestAllow` per-symbol test, name-echoing doc comments, no Example. Verdict: **read then partially applied**; whatever requires an inference step (external package from "exercise as its user") or fights a prior (comment conventions) is lost. Matches the canon's warning that what works for Opus "might need more detail" for weaker models.

**Case 3 — i1 limiter opus (the "MUCH WORSE" run): a gap, not disobedience.** It cited the skill (*"testing/synctest … is exactly what the skill points to for goroutine/context code"*), used an external package and synctest — and produced the loops-and-arithmetic bodies the maintainer refused to finish reading. v1 contained **no** body-style guidance; the model filled the vacuum with its priors. Lesson: an absent guideline is filled by the prior; a present-but-abstract guideline is *also* filled by the prior (see case 1). Only concrete form (exemplar) displaces the prior.

**Case 4 — i2 config sonnet: work done, evidence lost.** Its final structured output reports deleting the race-condition smoke ("the docs promise no concurrency safety"), deleting the scaffold, consolidating to `TestConfig` in an external package, converting to `b.Loop`. But it saved no review report, never mentioned typed errors, and the viewer showed the maintainer nothing ("I don't see any outputs for this case"). Verdict: **read and largely followed**; the REVIEW DEPTH failure is real (typed-errors flag genuinely absent) but compounded by a harness gap — nothing in the task or skill asks for a durable review artifact.

**Canon pointers are dead weight in-run.** Across all 12 transcripts, zero reads of hash package tests, io multi/pipe tests, TestComments, TableDrivenTests, cmd/go testdata, or analysistest. The skill's stated method — "each section pairs a guideline with canon to read and mimic first" — was never once executed as written.

### Grader vs. maintainer: fix the eval where it contradicts the intent corpus

1. **`TestLimiter` was graded a naming FAIL** ("a bare type name claiming no property") — but the skill blesses it ("a whole-package scenario test named for the package, TestConfig for package config, is good", lines 66-67) and the maintainer's own feedback praised it ("The first `TestLimiter` is nice, as it is clearly a test function that exercises the entire package"). The i2 opus naming failure is spurious. Correct the assertion to exempt the whole-package scenario name.
2. **The straight-line-bodies assertion bans all tables/loops**, but feedback-2 says: "I'd have tested panics the best with a **table-driven test** and a **helper** to recover from panics … name the inputs in the failure messages." The maintainer's objection was run-time arithmetic deriving expectations and structure that saves lines at the cost of readability — not tables per se. As graded, the maintainer's own preferred solution would fail. Re-scope the assertion to: no run-time computation of expected values; no sub-tests/helpers whose only payoff is line count; tables acceptable where the maintainer's own feedback endorses them.
3. **REVIEW DEPTH is ungradeable without an artifact.** Require the review task to save a report file; otherwise the flag can only live in a transcript nobody grades.
4. **Viewer/collection bug:** units-add-test outputs (both iterations, all configs) and i2 config-sonnet outputs never reached the maintainer, though `outputs/` and `grading.json` exist on disk. A third of the qualitative corpus is silently missing; fix before the next feedback round.

---

## Workstream 3 — Structure options

Constraints honored: single-file preference until a corpus is justified; no checklists; guidelines carry their why. Grounding: taste fails as prose across all models (empirics b); models never chase remote canon (forensics); examples beat descriptions (canon); first-5,000-token compaction survival and mid-file sag (canon + empirics); the maintainer's feedback-2 adds ~10 nuances that must land somewhere.

### (a) Single file, principles-first, compact counter-default block

Shape: 3-4 generative principles up top (contract, package-as-unit, testability-as-design-signal), then one tight section of counter-default corrections (external package, synctest idiom, no unpromised concurrency, no error-string asserts, examples expected, b.Loop), each one sentence + one why-clause; review-mode and add-one-mode notes; taste described in prose as today.

- Pros: smallest always-loaded footprint (~1.3-1.5k tokens, below today's 2.2k); everything survives compaction; no routing risk; easiest diff against maintainer intent.
- Cons: does nothing for the class of failures that prose has now twice failed to fix (bodies, comment voice, Example presence). Feedback-2's nuances either bloat it back to 2.2k+ or get dropped. The "compact block" flirts with the no-checklists constraint; it stays legal only while every line keeps its why and the block stays under ~10 lines.
- Token cost: ~1.4k loaded always; no on-demand cost.
- Verdict: best pruning discipline, no answer to the taste gap. Insufficient alone.

### (b) Single file + exemplar corpus (imitation over instruction)

Shape: keep one SKILL.md; add one bundled exemplar — a txtar-shaped (or plain directory) miniature package: a small API with doc comments plus its committed `_test.go` written exactly to the maintainer's taste (names, straight-line bodies with hard-coded numbers and one delicate-math comment, contract-clause doc comments carrying rationale in layman terms, a package-level Example, a table+recover helper for panics with inputs named in failure messages, one synctest bubble with an idiomatic sleep). SKILL.md orders the read: "Before writing any test, read `exemplar/` end to end; your file should look like it grew in the same repo."

- Pros: attacks precisely the 0/3 failures; canon-endorsed ("examples convey the desired style … more clearly than descriptions alone"); local file, one level deep, so it actually gets read (unlike GOROOT canon — 0/12 chased); the exemplar absorbs most feedback-2 nuances *silently* (comment voice, panic-table shape, simplest-contract-first ordering, inlined trivial sub-tests) without adding a single rule sentence; future taste feedback lands as exemplar edits, not prose growth — this is the anti-dilution mechanism the maintainer is asking for.
- Cons: it is the corpus the maintainer wanted deferred — the justification bar must be met (it is: two iterations of prose failing on taste is the evidence); an exemplar can be imitated too literally (mitigate: exemplar covers one small domain, SKILL.md keeps the principles that generalize); maintenance of a second artifact; weak models must still be told to read it (they follow read instructions well — 12/12 read SKILL.md when instructed).
- Token cost: SKILL.md ~1.6k loaded; exemplar ~1.0-1.4k on demand per writing task. Net vs today: roughly even, with far better expected yield per token.

### (c) SKILL.md + per-task reference files (write-new / review-existing / add-one-test)

- Pros: per-run context is minimal; review-only doctrine (consolidate, delete unpromised concurrency, flag string errors) stops taxing write tasks; canonical progressive-disclosure shape.
- Cons: our failures are cross-cutting, not mode-specific — bodies, comment voice, and Example presence fail in write mode and review mode alike, so every mode file re-states the same taste and the duplication *worsens* drift; the skill is 145 lines, nowhere near the 500-line split threshold; adds a routing step that weak models can fumble (the same models that missed a one-step inference); nested-read hazard if mode files cross-reference. The mode distinction is already handled adequately by two short sections (54-58, 127-132) totaling 11 lines. Add-one-test mode went 0/3 → 3/3 from a *five-line paragraph*; it does not need a file.
- Token cost: ~0.8k SKILL + ~0.6k mode file per run ≈ 1.4k effective, but with routing risk and 3x taste duplication to maintain.
- Verdict: premature at this size; re-open only if the review corpus grows its own canon (harness doctrine, fixture formats) past ~80 lines.

### (d) Recommended hybrid: restructured single file + one exemplar, mode files deferred

(a)'s pruning and ordering discipline applied to the single file, plus (b)'s exemplar as the sole bundled resource. No per-mode files. Rules that grade as landed stay one-sentence-with-why in SKILL.md; taste migrates out of prose into the exemplar; review-mode remains an in-file section. Position plan: generative principles first (compaction + primacy), counter-default corrections and the exemplar instruction early-middle, review section late but before the closer, and the design-signal principle as the closer (recency; it is also the skill's escape hatch when no guidance fits).

- Token cost: ~1.5k loaded + ~1.2k exemplar on demand for writing tasks. Comparable to today's always-loaded 2.2k, better placed.
- Density target: ≥0.8 load-bearing ratio (v2 today is roughly 0.6 by the evidence table — the seven baseline-covered restatements and the never-chased canon pointers are the filler).

---

## Workstream 4 — Concrete v4 plan

Nothing below changes the maintainer's intent; every keep/cut/move cites the evidence. Line numbers refer to the current `_scratch/skills/testing/SKILL.md`.

### Classification of current content

**Generative principles — keep, first (budget ~28 lines):**
- Test freely / commit only the contract; contract = promises to users; assertions trace to documented sentences (20-40, compressed). Feedback echoes it constantly; it is the skill's decision procedure.
- The unit is the package; exercise as its user (42-47). Add the one missing explicit clause: *committed tests live in an external `package foo_test` unless a peephole is required* — restores the draft-1 sentence whose absence cost the i2 sonnet run (baseline failed it in both iterations too).
- Testability is a design signal; hard-to-write tests indict the API (140-144). Keep as the closing section (recency + escape hatch).

**Counter-default corrections — keep, explicit, each with its one-clause why (budget ~22 lines):**
- synctest with bubble-sleep idiom and least-orchestration (115-119). Evidence: 0/4 → 3/3 across v1→v2. Do not touch the wording that worked.
- Never pin concurrency the docs don't promise; delete in review; doc-conventions presumption (110-113). Evidence: 3/3 new passes in i2.
- Add-one-test folds into siblings / sub-test / table row / Example (54-58). Evidence: 0/3 → 3/3.
- Never assert error strings; typed/sentinel via errors.Is/As; a grep-only test is a redesign signal (136-138). Evidence: load-bearing but weak (1/3) — **promote out of the last section** into the corrections cluster, and pair with the review-report expectation below.
- A committed suite includes runnable examples; package-level Example for shallow APIs; an ExampleLoad that just calls Load earns no commit (85-91, compressed). Evidence: baseline 0/3 ever; with-skill only 4/9 — restate as an expectation of the *deliverable*, not advice, and let the exemplar demonstrate it.
- b.Loop for benchmarks, canon postdating training data (97-100, compressed to 2 lines). Evidence: fixed i1 sonnet's optimizable benchmark.

**Recent-canon facts — keep, compact (folded into the corrections above):** synctest (Go 1.25), b.Loop (Go 1.24), TestComments/TableDrivenTests as named look-ups (79-83 → one line). `t.Context()` is *pending*: feedback-2 asks for verification first ("I send you to research the release notes … verify this and apply appropriately") — add only with a grading assertion attached (speculative until then).

**Derivable consequences — compress into their principle (net cut ~20 lines):**
- "dissolves most coverage arguments…" (28-29), "mark an inexperienced author…" (34-35), "Nor are unit tests bug hunts…" (35-36): fold into the contract principle; one sentence total.
- Constrained-CPU flake-hunting while uncommitted (24-26): maintainer intent (feedback-1: would run 100x under a cgroup) and observed in the i2 opus run; keep one clause inside "test freely".
- Naming stutter examples (62-70): keep the *claim-a-property* rule and the whole-package-name blessing (grader must be fixed to match); move the worked name pairs into the exemplar. Baseline already passes writing-mode naming — the prose earns 3 lines, not 10.
- "browse golang.org/x for inspiration" (144): never exercised in any run; harmless closer clause — keep only if a line is spare.

**Taste better carried by the exemplar (moves out of prose):**
- Straight-line bodies, hard-coded numbers, adjacent comment for delicate math, no premature helpers (72-76). Evidence: 0/3 with-skill as prose. Prose shrinks to one sentence stating the principle and its why (the math is performed at coding time so the reader never re-derives it); the exemplar *shows* it.
- Doc-comment voice: never opens with the function name; states the contract clause and the test's rationale in layman terms (78-83 + feedback-2's new demand). Evidence: opus 7/7 violations with the rule in context. The exemplar carries correct comments on every function; prose keeps the rule + the positive form in two lines.
- Panic testing: table + recover helper, inputs named in the failure sentence (new, feedback-2). Exemplar-only.
- Sub-test economy: inline trivial sub-tests; log-completing names like `Load(%q) succeeded for a config with %s` (68-70 + feedback-2). Exemplar shows one; prose keeps one line.
- Fixture voice: testdata/ file whose comments speak to both audiences ("Comments are the rest of the line following the # sign"), self-describing invalid-input fixtures (121-125 + feedback-2). Keep the two-line fixture guideline; the exemplar's testdata file demonstrates the voice.
- Simplest-contract-first ordering; hang-is-a-valid-failure (feedback-2). Exemplar ordering shows the former; the latter becomes one clause in the synctest paragraph (it is a why: don't contort orchestration to dodge hangs — a hanging test is a failing test).

**Named exemplar to add:** `exemplar/` — a rate-limiter-like miniature (the domain where all taste failures occurred): `bucket.go` (~40 lines, doc comments carrying the contract) + `bucket_test.go` (~90 lines: whole-package scenario test named for the package; simplest contract first; straight-line bodies with hard-coded numbers and one delicate-math comment; a panic table with recover helper; one synctest bubble with idiomatic sleep and zero gratuitous Wait; contract-clause doc comments with rationale; package-level Example) + `testdata/` one self-describing fixture. Built from the maintainer's graded outputs (i2 Fable limiter run passed 8/9 — start from it and apply feedback-2 line by line), so the exemplar *is* the intent corpus made executable.

**No supporting eval evidence yet — flag speculative, eval before keeping:**
- Harness packages per applicative layer; httptest/fstest/iotest shapes (104-108). No eval touches harnesses. Keep one sentence at most, or cut and add a harness eval first.
- "Never test a library through a distributed CLI binary" (46-47). No eval; plausible intent, zero signal. One clause or eval it.
- io multi/pipe bespoke-fakes pointer (93-95). Never chased, never graded. The exemplar's fake (if the miniature needs one) replaces it; otherwise cut the pointer, keep "a fake read in full keeps the test honest" as one clause.
- hash-package and cmd/go/analysistest pointers (50-52, 123-125). Never chased in 12 runs. Cut the instruction-to-read; a bare attribution ("the shape the stdlib uses") costs three words and keeps the lineage.
- `t.Context()` — pending maintainer verification, as above.

### Proposed v4 section map (single file + exemplar, ~95-105 lines total prose)

| # | Section | Lines | Content |
|---|---|---|---|
| — | frontmatter | 13 | unchanged (it grades fine) |
| 1 | Purpose + method | 4 | contract standard; method: read `exemplar/` before writing, imitate it |
| 2 | Test freely; commit only the contract | 12 | principle + traceability + constrained-CPU clause |
| 3 | The unit is the package | 10 | as-its-user + **explicit `package foo_test`** + fewer-tests-covering-more + add-one-test paragraph (verbatim from v2 — it works) |
| 4 | What the committed file contains | 14 | corrections cluster: examples expected; names claim properties (3 lines); bodies principle (1 line, exemplar carries the form); doc-comment rule + positive form (2 lines); b.Loop (2 lines); TestComments/TableDrivenTests one-line lookup |
| 5 | Concurrency and time | 10 | synctest paragraph verbatim from v2 + unpromised-concurrency paragraph verbatim + hang-is-failure clause |
| 6 | Fixtures | 5 | one reviewable fixture; testdata over long consts; fixtures speak (exemplar shows the voice) |
| 7 | Reviewing an existing suite | 8 | consolidation doctrine + **error-string flag promoted here** + deliverable: written review naming redesign signals |
| 8 | Testability is a design signal | 6 | closer: hard-to-write → redesign; io interfaces over paths |

Always-loaded cost ≈ 1.4-1.6k tokens (down ~30%), exemplar ≈ 1.2k on demand. Everything in sections 1-4 sits inside the first ~800 tokens (compaction-safe, primacy-weighted); the two sections that must fire during review (7) and design (8) hold the recency slots.

### Pruning list with evidence

| Cut / compress (current lines) | Evidence |
|---|---|
| "dissolves most coverage arguments", "worst kind there is / inexperienced author", "Nor are unit tests bug hunts" rhetoric (28-29, 33-36) | Derivable from the contract principle; baseline already avoids bug-hunt suites in 3 of 4 relevant gradings; contributes no unique assertion pass |
| 7 of the 10 naming-example lines (62-70) | Baseline passes writing-mode naming (i2 limiter without_skill PASS); worked pairs move to exemplar |
| Bodies paragraph as prose (72-76) | 0/3 with-skill in i2 — twice-failed prose; principle stays one line, form moves to exemplar |
| Read-the-hash-tests, io multi/pipe read instruction, cmd/go/analysistest read instruction (50-52, 93-95, 123-125) | 0/12 runs chased any external canon pointer; keep three-word attributions only |
| Harness paragraph to one sentence (104-108) | No eval exercises harnesses; speculative until a harness eval exists |
| "browse golang.org/x" (144) | Never exercised; keep only if a spare line remains |
| Example-section commentary compressed (85-91 → 3 lines) | The load-bearing part is the expectation + the ExampleLoad-earns-no-commit test; the rest is demonstrated by the exemplar |

### Do not touch

The synctest paragraph (115-119), the unpromised-concurrency paragraph (110-113), and the add-one-test paragraph (54-58): these are the three wording units with proven 0→3/3 flips. Move them intact; rewording risks losing the effect.

### Eval changes to land with v4 (so the next feedback round measures intent, not noise)

1. Exempt whole-package scenario names (TestLimiter/TestConfig) in the naming assertion.
2. Re-scope the straight-line assertion: forbid run-time-derived expectations and line-saving indirection; allow the maintainer-endorsed panic table with recover helper.
3. Review eval requires a saved report artifact; add an assertion that it names the string-error redesign.
4. Fix the viewer/output collection for units-add-test and the missing sonnet config outputs — two feedback rounds have silently lost this data.
5. Add one assertion each for any feedback-2 nuance kept as prose (t.Context, hang-is-failure) — no unevidenced rule rides along.
