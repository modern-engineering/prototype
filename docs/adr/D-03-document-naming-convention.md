---
status: accepted
since: 2026-05-31
---

# Name Documents by Family, Number, and Stem

## Context and Problem Statement

The corpus spans several families — analyses, requirements, decision records —
and every document needs a name that a human recognises and a tool can resolve.
A bare sequence number ([D-01 decision records] started at `001`) does neither
well: people do not recall what "002" decided, and an unquoted number in YAML
frontmatter is read as an integer, losing its zero-padding and differing between
parsers.

## Decision Drivers

- A reader should recognise a reference on sight; the stem, not the number,
  carries that meaning.
- The number must stay a stable key that sorts and greps, and survive being
  written in frontmatter without coercion.
- One rule should resolve any identifier to a file, across every family.

## Considered Options

- **`FAMILY-[AREA]-NUMBER-STEM`** — a family letter, optional area, padded
  number, and a human stem (adapted from the [eddt project][eddt]).
- **`FAMILY-NUMBER`** — family letter and number only, the stem kept out of the
  name.
- **Bare `NUMBER`** — the starting scheme.

## Decision Outcome

Chosen option: "`FAMILY-[AREA]-NUMBER-STEM`." A document's name, its filename
(minus `.md`), and the form references use are one and the same, for example
`D-02-madr-format-with-frontmatter`:

- `FAMILY` is one letter that also selects the directory: `A` analyses
  (`analyses/`), `R` requirements (`reqs/`), `D` decision records (`adr/`).
- `AREA` is an optional sub-namespace within a family, omitted until one is
  needed.
- `NUMBER` is zero-padded to a per-family width — two digits for `A` and `D`,
  three for `R` — sized to the records each family expects.
- `STEM` is the kebab-case description, and is paramount: it is what a reader
  recognises and cites. The `FAMILY-[AREA]-NUMBER` prefix is only the stable key
  that files, sorts, and greps; the stem is never dropped from a reference.

A leading letter keeps every identifier a string rather than a YAML integer, and
the prefix resolves a reference to its file by `<family-dir>/<prefix>-*.md`.

### Consequences

- Existing records take the new names; the number is preserved, the stem added.
- References in prose and frontmatter are self-describing, for example
  `derived-from: A-03-parameterization`.
- A stem is refined only by renaming the file, which git records and a prefix
  search makes easy to follow.

## More Information

- [eddt identifier grammar][eddt]

[D-01 decision records]: D-01-use-architectural-decision-records.md
[eddt]: https://github.com/resystems-io/eddt
