# Trigger tuning: unfinished, and why

`trigger-eval.json` holds 20 queries (10 should-trigger, 10 deliberate
near-misses: non-Go frameworks, `go vet` failures, pprof, a mechanical
rename touching `_test.go`, a synctest explainer, fuzz-crash debugging,
godoc-only work, a test-plan doc). They were written to grade the skill's
`description` + `when_to_use`, and they have never produced a usable
measurement.

## What happened

The skill-creator's optimizer (`scripts/run_loop.py` in the official
skill-creator skill) was run twice against `skills/testing`, three
candidate descriptions per run, three samples per query, model
`claude-opus-5`. Every candidate scored the same: precision 100 percent,
recall 0 to 6 percent, accuracy about 50 percent. That is the signature of
a detector that never fires, not of descriptions that never match: one
candidate opened with "Triggers: write or add tests for a Go package or
symbol" and still scored 0/3 on "write the committed tests for it, i'm
opening the PR today".

Two causes were isolated:

1. `run_eval.py` proxies a skill by writing a **slash command** into
   `<project>/.claude/commands/<name>.md` and watching whether `claude -p`
   invokes it. Current Claude Code does not autonomously invoke slash
   commands, so the proxy cannot fire regardless of wording.
2. It resolves its project root by walking up for a `.claude/` directory,
   which landed on `$HOME`. The first query set cited paths like
   `./internal/retry`; those runs died on "no such path" before any skill
   decision. The queries here were rewritten to be self-contained (the
   package is described inline), which changed nothing — that is what
   isolated cause 1.

## How to measure it properly

Install the skill as a **skill** (a `SKILL.md` under a skills directory the
target session loads, or the plugin itself), run each query through
`claude -p --output-format stream-json`, and detect a `Skill` tool call
naming it. Then re-run this eval set unchanged; it is still a good set. A
holdout split matters: optimize on ~60 percent, report the rest.

## What shipped instead

The current `description` and `when_to_use` were written by judgment
against Anthropic's authoring guidance (key use case first, concrete
situations, an explicit skip list) rather than by measurement. Treat them
as unvalidated. The previous wording led with doctrine vocabulary ("the
golden hierarchy of contract authority") instead of the moments the skill
should fire, which is the specific weakness the rewrite targets.
