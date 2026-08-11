# Authoring documentation in `docs/`

- Apply INCOSE systems-engineering practice.
  Most notably:
  - establish the mission before developing the operational concept;
  - analyze the problem before choosing a solution;
  - keep every artifact traceable to the one that motivated it.
- Consult @README.md for the document index.

## Conventions

- A new document takes the next free number in its family; never reuse one.
- Mission documents establish the enduring frame for downstream work.
  Only the sponsor may accept one, and only after reviewing its exact revision.
- Frontmatter follows the schema the `adr/` records define:
  - `status` (`draft`/`proposed` while unstable, `accepted` once settled, `deprecated`/`superseded` later)
  - `since` (date of that status)
  - plus traceability (`derived-from` on requirements, `supersedes`/`superseded-by`)
- Working names analyses introduce in brackets (`Descriptor`, `Instance`) are provisional until a requirement locks them.
- Adding or renumbering a document updates its family `README.md` index in the same change.

## Cross-references

- Reference only documents that already exist.
  Name a concern a later document will take up as a gap in prose ("a later analysis develops this"), never as a link to a document not yet written; the change that adds that document edits the earlier ones to turn the gap into a reference.
  References accrete in commit order, so every link points back to something already in the corpus.
- Link between documents reference-style, with definitions at the foot of the file.
- Prefer shortcut reference links (`[D-03 Document Naming]`) or collapsed (`[D-03 Document Naming][]`), where the words naming the target double as the label; reserve the full `[text][label]` form for when the label must differ.
  Pair the ID with words, never a bare identifier; for a long target, name the section and point its definition at the `#fragment`.
- This overrides the global inline-link default and is scoped to `docs/`; do not revert it.
- Keep inline links only in `README.md` index tables, where the row supplies context.
