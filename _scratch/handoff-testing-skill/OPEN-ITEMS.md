# Open items, ranked

## 1. Ship the skill

Copy `skills/testing/` to `golang-dev/skills/testing/` in
`~/Code/notorious-ai/claude-plugins`, `SKILL.md` plus `references/` and
`exemplar/` intact, since SKILL.md quotes those paths. Convention gap: the three
skills shipped there carry `name` plus `description` only, this one splits out
`when_to_use`, and the maintainer hand-folded that YAML (INTENT ruling 16) — ask
before rewriting. `CHANGELOG.md` wants a per-skill section, `plugin.json` a bump.

`golang-dev/TESTS.md` is untracked scratch, the skill's seed, and INTENT.md cites
it as a source: reconcile, do not delete. All of it now lives in the skill except
one line — "underscore in test-names is meaningless, it is only meaningful in
example tests for go-doc to attribute it to some exported symbols". Neither
SKILL.md nor `references/examples.md` states that `Example_name` /
`ExampleType_Method` rule; add it there, then reduce TESTS.md to a pointer. Its
line 22, on testing packages, is answered by `harnesses.md`.

Re-run the description eval after installation: the archived tuning run
(`_scratch/skills/testing-it4/trigger-tuning/`, untracked) scored 10/20 with every
positive at a 0.0-0.33 trigger rate and every negative passing, the signature of a
skill the harness could not load. Treat `evals/trigger-eval.json` as unmeasured.

## 2. v5 candidates, with their evidence

- `references/evaluator-cases.md`'s goroutine-leak rule still calls the
  bubble-exit question "under investigation". It is settled (EVIDENCE, ruling
  23): make it "bubble exit is the leak detector; outside bubbles, a WaitGroup".
- Ruling 21 (exact bubble timing: the deadline instant or `duration+1`, never a
  hopeful overshoot) and ruling 22 (named cases for semantically meaningless
  values; 64/128/1024 goroutines) came with round 4's feedback, unlanded.
- Harnesses stay the one area with zero eval evidence (rulings 13, 17); an eval
  whose fixture ships a conformance verifier would close it.

## 3. Review fan-out lessons from the graduation misses

Eight in-scope audit findings went unsurfaced (`graduation-report.md`, "Misses").
The pattern is scoping, not blindness: a reviewer touched `lspcmd` twice and walked
past the audit's most actionable lsp item, and `httpapp`, the audit's top-ranked
package, was lost to a self-declared sub-scope. Assign scopes exhaustively, forbid
self-narrowing, sweep the leftovers.

## 4. A real defect to fix in the prototype repo

At `nightshift-complete`, `application/descriptortest.go` and
`solution/provisiontest.go` both enforce a stable-flag-schema invariant across two
`Make` calls (nil-ness, flag count, per-flag usage and default), and neither
self-test violates it: `descriptortest_test.go`'s six rejection cases and
`provisiontest_test.go`'s ten both stop at the shared-FlagSet fault. The twin
`schemetest_test.go` has an "unstable schema" case — copy it (graduation A11, S7).

## 5. Eval-design rules learned

Fixtures stay domain-distant from whatever the exemplars show: a token-bucket
exemplar beside a token-bucket fixture turned the limiter eval into a copying test
that discriminated on nothing (ruling 18). Assertions must not contradict the
intent corpus: round 2 graded `TestLimiter` a naming failure the maintainer had
praised and banned tables he had asked for (`testing-workspace/research/attention-report.md`,
"Grader vs. maintainer"). Review evals need a committed artifact to grade.

## Trigger measurement (open, instrument broken)

The skill's `description` and `when_to_use` were rewritten by judgment, not
measured: the skill-creator's optimizer proxies triggering with an
auto-invoked slash command, which current Claude Code never does, so all
six candidates scored an identical ~50% accuracy with 0-6% recall. The
20-query eval set survives at `evals/trigger-eval.json` and the diagnosis
with a working measurement recipe is in `evals/TRIGGER-TUNING.md`. Re-run it
once the skill is installed as a skill in the plugin, detecting the `Skill`
tool call rather than a command invocation.
