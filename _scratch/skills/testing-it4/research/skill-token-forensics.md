# Skill token forensics — what each agent actually loaded from skill-v4

Question: per-agent context cost of the skill, and whether archetype routing
(post-retrofit README + SKILL.md role split) made writers load partially while
critics load the corpus whole.

Method: parsed `agent-*.jsonl` transcripts under
`~/.claude/projects/.../0a1b1b62.../subagents/workflows/`. For every Read/Bash/Grep
tool call whose input touches a `skill-v4` path, summed the characters of the
tool_result payload actually delivered into context (includes `cat -n` line-number
overhead, ~5-7%). Tokens estimated as chars/4. Partial = Read with offset/limit or
a bash extraction (`grep -A`, `head`, `awk`); none of the Reads used offset/limit,
the one partial load below is a `grep -n ... -A 40` sample extraction.

Three phases of skill-v4 observed:

1. Jul 23 baseline (`wf_55d60ead-ae4`): SKILL.md 6.7k ch, exemplar = one Go pair
   (`bucket.go` + `bucket_test.go`), no routing.
2. Jul 24 15:42 `__snippets` spot-checks (`wf_d2656f3b-df0`): harvested 8-file canon
   corpus, pre-retrofit README (0.8k ch), prompt orders "read exemplar/ end to end".
3. Jul 24 16:10 `__final` chain (`wf_0ad1592a-b0f`): sample-tag retrofit done,
   router README 3.1k ch, SKILL.md writing-mode says read ONLY matched samples.

## Per-agent table

| Agent (wf/id) | Role | Skill files loaded | Full/partial | Est. skill tok |
|---|---|---|---|---|
| base ad687543 (plain fable r1) | writer | SKILL.md, bucket.go, bucket_test.go, refs/concurrency | all full | ~4.2k |
| base ac26a698 (looped fable r1) | writer | same set as above | all full | ~4.2k |
| base a8a04e86 (looped fable r1, rerun) | writer | above + refs/tables-and-subtests, refs/helpers | all full | ~5.6k |
| base adb1b8b9 (looped fable r1) | critic | SKILL.md, bucket pair | all full | ~3.9k |
| base ae16bd2a (looped fable r1, rd 2) | critic | SKILL.md, bucket pair, 3 refs | all full | ~5.5k |
| snip a447515d (sonnet) | writer | SKILL.md, README, all 8 exemplar, 5 refs | all full | ~11.7k |
| snip a5fffcff (fable) | writer | SKILL.md, README, all 8 exemplar, 3 refs | all full | ~10.8k |
| snip a724a925 (opus) | writer | SKILL.md, README, all 8 exemplar, 2 refs | all full | ~10.1k |
| final abd50eb3 (looped fable) | writer | SKILL.md, README router, database-sql.md, encoding-json.md, refs/concurrency; strings.md#example-edge-case | 2/8 exemplar full + 1 grep-extracted sample | ~5.4k |
| final a9b9b1ce | critic | SKILL.md, README, ALL 8 exemplar, 3 refs | all full | ~13.1k |
| final a346098d | reviser | SKILL.md, README, database-sql.md, encoding-json.md, refs/concurrency | all full, 2/8 exemplar | ~5.1k |
| final ada68054 | grader | none | — | 0 |

## Verdict on selective loading

Routing WORKS, and the split is exactly the one SKILL.md orders.

- The `__final` writer matched the cache package's clusters to lifecycle /
  async-assert / example, then loaded only `database-sql.md` and
  `encoding-json.md` in full plus `strings#example-edge-case` via
  `grep -n "example-edge-case" -A 40` (1.5k ch instead of strings.md's 4.8k).
  No unmatched exemplar file was opened. Exemplar payload: 12.1k ch vs the
  critic's 35.7k ch — writers carry 34% of the exemplar corpus.
- The reviser stayed selective on re-entry: same two matched files, no corpus sweep.
- The `__final` critic read every exemplar file end to end as the REVIEWING-mode
  instruction demands: 13.1k tok, the most expensive skill consumer observed.
- The grader never touched skill-v4 — impartiality by construction holds.

Cost trajectory for a writer: ~4.2k tok on the small Jul 23 corpus, ~10-11.7k tok
when the harvested 8-file corpus had to be read end to end (`__snippets` round,
2.4-2.8x the baseline), back to ~5.4k tok once routing landed. The retrofit paid
for the corpus growth: the final writer pays roughly baseline price (+29%) for a
corpus 2.5x larger, while only the critic, whose job is the grading sense, pays
the full ~13k. Spot-check caveat: the `__snippets` prompts themselves ordered
end-to-end reads, so that round measures the pre-routing corpus cost by design,
not a routing failure.

Not routing-related but visible in the same transcripts: all three `__snippets`
writers also ran a corpus-wide `grep` for `mustPanic`/panic helpers, i.e. writers
do reach for cross-file queries; the retrofit README's extraction one-liners are
the sanctioned channel for exactly that.
