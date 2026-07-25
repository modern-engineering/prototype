# Method — how the eval loop runs

The loop that produced four skill versions; reproduce it for the sibling. These
shapes are what made the feedback usable.

## Fixtures, prompts, assertions

Each eval is one small Go module under `example.invalid/…` with real doc comments
and a planted situation: `truncate` hides doc-vs-code drift, `config` ships a bad
suite to review, `units` a healthy suite to extend, `limiter` and `cache` are
write-from-scratch packages with documented boundaries. The prompt in
`eval_metadata.json` is a colleague's ask in chat register ("mind writing the
committed tests before this goes into the services repo?"), never naming the
skill or its vocabulary — trigger behavior is part of what is measured.

Assertions are the graded contract, one clause each, quoting the maintainer's own
words where possible. Stability is an assertion too: `go test -count=30` and
`-race -count=5`. Review evals assert a committed artifact (`report.md`), since
transcript-only review depth cannot be graded.

## Graders and the surrogate

The grader is a fresh agent that never reads the skill (verified in the token
forensics: zero skill bytes) and writes `grading.json` with pass/fail plus quoted
evidence per assertion, re-running the suites itself rather than trusting a
report. `benchmark.json` aggregates the runs and `review.html` renders one page per
run: the maintainer could not see a third of round 3's outputs in a combined view.

The maintainer surrogate reviews in the maintainer's voice and rates 1-10,
calibrated against the whole intent corpus. Its calibration record ships with its
output (`_scratch/skills/testing-it4/iteration-4/surrogate-review.json`,
`calibration`): directionally right on 11 of 16 round-3 runs, own errors named,
including over-rating runs where that round revealed criteria absent from its
corpus. Two rules keep it honest: enforce revealed intent only, never award a 10.

## The replicate grid

Three replicates per configuration, three models (fable, opus, sonnet), two arms.
Single runs had confounded the skill's effect with run variance for three rounds;
the grid turned "the loop helps" into "8 of 9 cells, here is the loser". Introduce
a fresh, unseen fixture in any round whose question is contamination.

## The keep-or-prune rule

Every guideline needs failing-baseline evidence. If the no-skill baseline passes
an assertion consistently, the guideline restates what the model already does and
is a pruning candidate: the table in
`_scratch/skills/testing-workspace/research/attention-report.md` lists seven. Its
corollary — structural rules land from prose, taste does not — is why taste needs
exemplars and a critic rather than more sentences.
