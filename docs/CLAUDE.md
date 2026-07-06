# Authoring documentation in `docs/`

- Apply INCOSE systems-engineering practice. Most notably:
  - analyze the problem before choosing a solution;
  - keep every artifact traceable to the one that motivated it.
- Consult @README.md for the document index.

## Conventions

- A new document takes the next free number in its family; never reuse one.
- Frontmatter follows the schema the `adr/` records define:
  - `status` (`draft`/`proposed` while unstable, `accepted` once settled,
    `deprecated`/`superseded` later)
  - `since` (date of that status)
  - plus traceability (`derived-from` on requirements,
    `supersedes`/`superseded-by`)
- Working names analyses introduce in brackets (`Descriptor`, `Instance`) are
  provisional until a requirement locks them.
- Adding or renumbering a document updates its family `README.md` index in the
  same change.

## Analyses

- In the INCOSE spirit, an analysis surveys the viable paths through its problem
  space and records the forces acting on each; it may close with a
  recommendation, but committing to one path is a decision record's job.

## Sourcing

- Never name proprietary systems. Field evidence is anonymized ("a multi-tenant
  production system we studied"); lessons enter the corpus, names and internals
  stay off the public record.
- Quote illustrative excerpts inline rather than referencing repository files;
  committed files drift freely and leave such references stale. Snippets are
  sketches, not references to committed code.

## Cross-references

- Reference only documents that already exist. Name a concern a later document
  will take up as a gap in prose ("a later analysis develops this"), never as a
  link to a document not yet written; the change that adds that document edits
  the earlier ones to turn the gap into a reference. References accrete in
  commit order, so every link points back to something already in the corpus.
- Link between documents reference-style, with definitions at the foot of the
  file.
- Prefer shortcut reference links (`[A-03 Parameterization]`) or collapsed
  (`[A-03 Parameterization][]`), where the words naming the target double as the
  label; reserve the full `[text][label]` form for when the label must differ.
  Pair the ID with words, never bare `[A-03]`; for a long target, name the
  section and point its definition at the `#fragment`.
- This overrides the global inline-link default and is scoped to `docs/`; do not
  revert it.
- Keep inline links only in `README.md` index tables, where the row supplies
  context.
