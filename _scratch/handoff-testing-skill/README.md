# Carry-over package — the Go testing skill

A finished `golang-dev` testing skill, the eval harness that graded it through
four rounds, and the measured case behind its guidelines. Enough to ship the
skill and charter its sibling without the session that built them.

## What to do first

1. Read [INTENT.md] — the maintainer's rulings verbatim. It outranks every
   other document here, the skill included, on any revision question.
2. Skim [EVIDENCE.md] before touching a guideline: each earned its place.
3. Ship per [OPEN-ITEMS.md] step 1: placement, `TESTS.md` reconciliation, bump.
4. Charter the sibling from [SIBLING-SKILL-CHARTER.md]; eval loop in [METHOD.md].

## Files

| path                                               | purpose                                                            |
| -------------------------------------------------- | ------------------------------------------------------------------ |
| `INTENT.md`                                        | The maintainer's rulings 1-23; authority for every future revision |
| `EVIDENCE.md`                                      | Measured case for the design, with an archive path per claim       |
| `OPEN-ITEMS.md`                                    | Ranked ship steps, v5 candidates, a real defect to fix             |
| `SIBLING-SKILL-CHARTER.md`                         | Brief for the development-loop skill (rulings 10, 14)              |
| `METHOD.md`                                        | How the eval loop runs, so the practice continues                  |
| `skills/testing/SKILL.md`                          | Skill body: golden hierarchy, package-as-unit, role split          |
| `skills/testing/exemplar/README.md`                | Archetype router: writers match clusters, reviewers read all       |
| `skills/testing/exemplar/strings.md`               | transform, harness, example — case tables, shared runners          |
| `skills/testing/exemplar/hash-maphash.md`          | transform, stream — property names, one invariant many paths       |
| `skills/testing/exemplar/database-sql.md`          | lifecycle, async-assert — matrix runner, bubble sleeps             |
| `skills/testing/exemplar/io.md`                    | stream — one-behavior fakes, regression promoted to property       |
| `skills/testing/exemplar/archive-zip.md`           | contract-conformance — shipping a verifier and calling it          |
| `skills/testing/exemplar/go-parser.md`             | harness, golden-fixture — inline `ERROR` marker language           |
| `skills/testing/exemplar/cmd-gofmt.md`             | golden-fixture — testdata corpus, `-update`, idempotence           |
| `skills/testing/exemplar/encoding-json.md`         | example — pkgsite voice, plus a self-naming violation to avoid     |
| `skills/testing/references/helpers.md`             | Helpers, fakes, the method-on-case line, placement                 |
| `skills/testing/references/tables-and-subtests.md` | When a table fits, `Field=Value` sub-test names                    |
| `skills/testing/references/fixtures.md`            | File inputs, long literals, malformed cases                        |
| `skills/testing/references/examples.md`            | Runnable-example doctrine (ruling 20)                              |
| `skills/testing/references/concurrency.md`         | synctest, WaitGroups, panics, invalid inputs                       |
| `skills/testing/references/harnesses.md`           | Conformance suites, derived from Go-team practice (ruling 17)      |
| `skills/testing/references/evaluator-cases.md`     | Rulings evaluators mis-graded, as graded pairs                     |
| `evals/fixtures/limiter/`                          | Token bucket: lifecycle, synctest, documented panics               |
| `evals/fixtures/config/`                           | Loader plus a bad test file, for the review eval                   |
| `evals/fixtures/units/`                            | Internal package with a live suite, for the add-a-test eval        |
| `evals/fixtures/truncate/`                         | Doc-vs-code drift planted for the golden-hierarchy eval            |
| `evals/fixtures/cache/`                            | Expiring cache: round-4 fixture, unseen by earlier rounds          |
| `evals/limiter-write-tests/eval_metadata.json`     | Prompt plus 14 assertions: write a suite from scratch              |
| `evals/config-test-review/eval_metadata.json`      | Prompt plus 11 assertions: review and rework a bad suite           |
| `evals/units-add-test/eval_metadata.json`          | Prompt plus 6 assertions: add one symbol's test to a live suite    |
| `evals/truncate-drift/eval_metadata.json`          | Prompt plus 9 assertions: surface drift instead of absorbing it    |
| `evals/cache-write-tests/eval_metadata.json`       | Prompt plus 15 assertions: lifecycle, concurrency, examples        |
| `evals/trigger-eval.json`                          | 20 queries (10 positive, 10 negative) grading the description      |
| `evals/synctest-leak-probe/`                       | Runnable settlement of ruling 23; written here, not in the archive |

## Traceability

Archive worktree
`/Users/danielorbach/Code/modern-engineering/prototype/.claude/worktrees/master-tests-orig`,
branch `worktree-master-tests-orig`. The effort landed as `e9f8ef3..a31caac`,
19 commits on top of `main` at `64882f1`; `a31caac` is the branch tip.
Description-tuning artifacts (`trigger-tuning/`, `trigger-tuning.log`) were
still untracked when this package was assembled. Tags: `m1-baseline` (commit
`43ee0cb`, 2026-07-09) is the milestone-1 tree; `nightshift-complete` (tag
object `11cad7d`, commit `e790804`, 2026-07-13) is the pre-remaster tree
`_scratch/TESTS-SURVEY.md` audits and the graduation exam reviewed.

Staying behind by design: the 68 graded round-4 runs with their outputs
(`_scratch/skills/testing-it4/iteration-4/`), rounds 1-3 with the maintainer's
per-run reviews (`_scratch/skills/testing-workspace/`), the HTML review pages,
the research reports, the re-verification logs, and the graduation findings.
EVIDENCE.md cites them by path; read them there when a number is questioned.
