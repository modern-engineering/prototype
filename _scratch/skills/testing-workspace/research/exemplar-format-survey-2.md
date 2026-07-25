# Exemplar format survey 2 — official sources only

Second opinion on the exemplar corpus format: Markdown bodies plus light XML anchors
(`<sample id/archetypes/source/lines>` wrapping fenced code, `<highlight archetype>` lines).
Evidence is restricted to Anthropic-official material; the maintainer's own plugins
(notorious-ai) were excluded by design. Extraction commands were tested hands-on against the
live corpus at `_scratch/skills/testing-it4/skill-v4/exemplar/` (9 files, 72-148 lines each,
22 samples, 8 archetypes).

## Official evidence

| Source | What it says | Bearing on the hybrid |
|---|---|---|
| [Skill authoring best practices][bp] | Reference files over 100 lines need a table of contents; partial reads (`head -100`) happen, so a file's scope must be visible up top. | Five corpus files exceed 100 lines. The README indexes the corpus, but per-file scope is only visible via grep. Adjustment 1 below. |
| [Best practices][bp], Pattern 2 | SKILL.md may ship a "Quick search" section of literal grep one-liners against reference files. | Direct official precedent for publishing extraction one-liners in the corpus README. |
| [Best practices][bp], examples pattern | Input/output example pairs inside skill files work "just like in regular prompting". | Prompting techniques officially apply inside files agents read. |
| [Best practices][bp], frontmatter rules | Only `name`/`description` "cannot contain XML tags". No restriction on the body or supporting files. | Body XML is implicitly permitted. |
| [Prompting guidance on XML tags][xml] | XML tags "help Claude parse complex prompts unambiguously, especially when your prompt mixes instructions, context, examples, and variable inputs". Multi-document pattern: wrap each document in `<document index=..>` with a `<source>` metadata subtag. Use consistent descriptive names; nest for hierarchy. | `<sample id/source/lines>` is the same shape as the official `<document index/source>` wrapper. Skill files enter the context on Read, so the guidance transfers. |
| [code.claude.com skills doc][cc] | Supporting files (`reference.md`, `examples.md`) are plain Markdown, referenced from SKILL.md; keep SKILL.md under 500 lines; one level deep. | Format-agnostic beyond "Markdown". |
| Official plugins cache (`~/.claude/plugins/cache/claude-plugins-official/`: skill-creator, plugin-dev, claude-md-management, notion, code-review, playground) | Every references/ and examples/ Markdown file is headings + fenced code + prose ("Pattern N" ... "**Use for:** ..."). No XML anchors in any reference file. XML appears only as `<example>` blocks in plugin-dev's agents/*.md. No documented extraction commands anywhere. | No precedent for XML anchors in reference files, but a live precedent for XML blocks inside agent-read Markdown. |
| [anthropics/skills][repo] (pdf, docx, mcp-builder sampled) | Same shape: headings + fenced code + prose. Code meant for verbatim reuse ships as real files under scripts/, not as annotated blocks. | The scripts/ pattern does not fit quoted exemplar code (see final section). |

## Verdict: KEEP, with two adjustments

The hybrid survives official scrutiny. No official reference file uses XML anchors, but the
official prompting guidance directly endorses the pattern for this exact problem: verbatim
documents interleaved with prose, each needing provenance metadata and unambiguous
boundaries. The `<document>`+`<source>` wrapper is the corpus's `<sample source= lines=>`
under another name, and nothing outside frontmatter forbids XML. Plain headings would lose
the machine-checkable ids, archetype attributes, and source/line provenance.

Adjustments:

1. Per-file scope line. Official guidance wants a ToC in reference files over 100 lines. Add
   one line after each file's title: `Samples: case-table, family-shared-runner, ...`.
2. Publish the verified one-liners in the corpus README (Pattern 2 precedent), together with
   the invariants they depend on: tags start at column 0; `id` is the first attribute; one
   `<highlight>` per archetype; `</sample>` at column 0.

Whole-file Read is the right default. Files are at most 148 lines; official docs prefer
complete reads and warn that partial reads yield incomplete information. Reserve the
one-liners for the writer agent's cross-file archetype queries and single-sample pulls.

## Verified extraction one-liners (run from exemplar/)

```bash
# Extract one sample by id (the closing quote makes the id exact; no substring hits)
awk '/^<sample id="case-table"/,/^<\/sample>/' strings.md

# List the sample ids in a file
grep -o '^<sample id="[^"]*"' strings.md | cut -d'"' -f2

# Files whose samples carry an archetype (comma-safe against substring names)
grep -El '^<sample .*archetypes="([^"]*,)?golden-fixture(,[^"]*)?"' *.md

# Every highlight for an archetype, across the corpus (multiline-safe)
awk '/^<highlight archetype="stream"/,/<\/highlight>/{print FILENAME": "$0}' *.md
```

Failure modes, tested:

- Unquoted id patterns over-match: `id="example` would hit `example-edge-case`. With the
  closing quote, the query returns zero rows instead. Keep the quote.
- Naive `archetypes="[^"]*assert` matches only `async-assert` today but would conflate a
  future plain `assert`; the comma-safe form disambiguates.
- The README mentions tags in backticks, so unanchored greps hit it. The `^` anchor excludes
  it (0 anchored hits) because real tags always start at column 0.
- An unclosed `<sample>` (file mid-write) makes a multi-file awk range leak into the next
  file. Sanity check per file: `grep -c '^<sample'` must equal `grep -c '^</sample>'`
  (the corpus passes today).
- Same-line open/close highlights are safe: awk ends a range on its own start line.

## Copy or ignore from the official skills

Copy: the Pattern 2 grep quick-search section; descriptive per-domain file names (already
done, one file per Go package); a ToC line for files over 100 lines; references one level
deep from the entry document.

Ignore: shipping code as scripts/ files. That pattern serves executable utilities; exemplar
code is quotation with mandatory attribution and paired prose, and splitting it into bare
.go files would sever the annotation from the evidence. Also ignore notion's narrative
walkthrough example format; it documents tool workflows, not code exemplars.

[bp]: https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices
[xml]: https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/use-xml-tags
[cc]: https://code.claude.com/docs/en/skills
[repo]: https://github.com/anthropics/skills
