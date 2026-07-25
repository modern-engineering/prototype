# Exemplar corpus format survey

Four consumers: (a) whole-file reading by a review agent, (b) targeted extraction by a writer agent
arriving via an archetype index, (c) human reading/editing, (d) mechanical greppability.

## Evidence base

- Plugin precedent (`~/.claude/plugins/marketplaces/notorious-ai/golang-dev/`): all three skills'
  `examples/` files wrap entries in `<example>`/`<anti-pattern>` with `<category>`, `<good>`,
  `<bad>`, `<why>` sub-tags. `committing/examples/go-code-examples.md` documents the tag table at
  the top and states the purpose: "XML tags structure each entry for selective searching." Weakness
  observed: code inside `<good>` is never fenced (zero ``` in `stdlib-examples.md`), so no syntax
  highlighting, and GitHub's sanitizer strips the tags and mangles the code into prose.
- Anthropic skill authoring (platform.claude.com best-practices; code.claude.com/docs/en/skills):
  every supporting-file example is Markdown. Explicitly recommends grep against reference files
  ("Quick search: `grep -i "revenue" reference/finance.md`") and a table of contents for files over
  100 lines because agents preview with partial reads. No XML-file guidance.
- Anthropic prompting guidance: "XML tags help Claude parse complex prompts unambiguously,
  especially when your prompt mixes instructions, context, examples, and variable inputs."
  Recommends nesting with attributes (`<document index="n">`).
- In-flight harvest (`_scratch/skills/testing-it4/skill-v4/exemplar/`, 8 files + README): the
  harvester chose pure Markdown. One H1 per file; snippets introduced by prose labels ("Lines
  382-387:"), fenced ```go blocks, annotation paragraph after each fence. Tabs survive in fences.
  The code contains `<-afterPutConn` and `&&`, i.e. XML-hostile characters.
- Mechanics verified locally: `awk '/<example>/{n++} n==3, /<\/example>/'` extracts exactly one
  tagged block in one command. The harvest file offers no per-snippet anchor at all (only the H1),
  so targeted extraction today means whole-file read or brittle line offsets in the index.
  `deno fmt` leaves XML-tags-around-fenced-go untouched, tabs intact.

## Candidates

### Pure Markdown, fenced blocks (harvester's current choice)

Pros: best human rendering (Go highlighting on GitHub); verbatim/gofmt fidelity in fences; matches
all of Anthropic's own skill examples; zero ceremony. Serves (a) and (c) perfectly. Cons: no
per-snippet anchor — (b) needs line offsets that rot on every edit; headers cannot carry archetype
keys or attribution as data; `##`-to-`##` slicing requires one header per snippet and still yields
no attribute slots. (b) and (d) are weak.

### Markdown with XML anchor tags around samples (fenced Go inside)

Pros: exact paired anchors with attributes (`<sample archetypes="..." source="..." lines="...">`)
make (b) a one-command awk range and (d) a plain grep; matches the maintainer's own documented
precedent and Anthropic's XML-parsing guidance; fences inside tags keep highlighting, tabs, and
copy-paste fidelity (the file is parsed as Markdown, so `<-` needs no escaping); whole-file reads
are unharmed; `deno fmt` compatible. Fixes the precedent's one observed flaw (unfenced code). Cons:
GitHub strips unknown tags from the render, so attribute metadata vanishes for web readers —
mitigate by duplicating attribution as a prose line inside the block; mild raw-view noise.

### Pure XML

`<-afterPutConn` and `&&` in the harvested code force CDATA or entity escaping, breaking the
verbatim guarantee that is the corpus's whole point. No render, no highlighting. Reject.

### txtar

Verbatim and Go-native, but structurally wrong: one comment section before the first `-- file --`
marker, so per-snippet interleaved annotations have no home. For executable file trees, not
annotated exemplars. Reject.

## Recommendation

Markdown with light XML anchor tags: keep the harvest files' Markdown body (fenced verbatim Go,
prose annotations) and wrap each snippet entry in a `<sample>` element carrying archetype, source,
and line attributes, with `<highlight archetype="...">` lines for the archetype index to key into.
This is the union of the harvester's fidelity and the plugin precedent's searchability. Mock entry:

````markdown
<sample id="wait-before-read" archetypes="lifecycle,async-assert" source="database/sql/sql_test.go" lines="382-387">

Source: `database/sql/sql_test.go` lines 382-387 (Go 1.27 dev). Verbatim; BSD-3-Clause.

```go
func (db *DB) numFreeConns() int {
	synctest.Wait()
	db.mu.Lock()
	defer db.mu.Unlock()
	return len(db.freeConn)
}
```

<highlight archetype="lifecycle">`synctest.Wait()` runs before the read, so the bubble's goroutines
settle first; the accessor never races the pool.</highlight>
<highlight archetype="async-assert">Async-state assertions become deterministic one-liners; no poll
loops, no sleeps.</highlight>

<note>The wait-before-read accessor: wait, then an ordinary locked read.</note>

</sample>
````
