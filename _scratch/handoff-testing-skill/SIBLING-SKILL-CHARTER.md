# Charter — the development-loop sibling skill

Decided in INTENT ruling 14: a SEPARATE golang-dev skill for the loop, while
the testing skill stays strictly about committed tests. Ruling 10 is its
philosophy, quoted below because the wording carries the intent.

## The hat-switch

Testing is the author's hat-switch into the user role — "the most left-shifted
feedback loop". Packages are developed WHILE their tests are developed, one
package at a time. Tests are not a phase after the code, they are the seat you
sit in to learn whether the API you just wrote is usable. The testing skill
states the standard for what gets committed; this skill owns how you get there.

## Non-committable playgrounds

Agents need playgrounds for bug-hunting: throwaway tests, scratch `main`s,
actually using the tool. None of it is committable, and the boundary is
load-bearing — a bug hunt is not a contract test. The testing skill refuses the
committed side ("a bug-hunt request gets run, not committed"); this skill says
where the hunt lives, how it stays out of the commit, and what gets promoted
when it finds something.

## Flows-first planning

Candidate shape from ruling 10: a review-experienced agent distills the typical
user flows and stories BEFORE the first test, handing the coding agent its
starting point — or the flows stay spontaneous, the open design question. Ruling
6 says why it matters: "thinking as users pins flows — that is by far superior",
while pinning language breeds mechanical, symbol-scanning suites.

## The critic-colleague ceremony

A fresh-context reviewer bound to the testing skill, reviewing as the package
maintainer. The two-evaluator variant: the main evaluator sees ONLY the
exported interface, a second may read the implementation as a best-effort end
pass. Measured caveats, all from EVIDENCE:

- The loop wins, but not uniformly: 8 of 9 cells improved, and it amplifies
  weaker models both ways — the round's single worst run was a looped one.
- The critic needs the owner-only line in front of it. One reviser edited the
  package under review, guard and exported doc, citing the bundled exemplar as
  evidence of intent. A critic may report drift; it may never resolve it.
- Critiques that land read like a colleague's: verification first (`gofmt`,
  `vet`, `-race`), findings ranked with the one that matters named as such.
  See the shipped example at
  `_scratch/skills/testing-it4/best-run-review/cache-write-tests--looped-fable-FINAL/with_skill/outputs/critique.md`.

Not this skill's business: anything about the shape of committed tests. That
line is the testing skill's, and it should stay quotable rather than restated.
