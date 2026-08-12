# Repository Guidance

## Documentation

- Analyze the problem before choosing a solution.
- Keep each artifact traceable to the artifact that motivated it.
- Adding or renumbering a document updates its family `README.md` index in the same change.
- Follow the decisions on document naming (D-03), lifecycle metadata (D-04), and document relationships (D-05).

## Cross-References

- In `docs/`, link only to documents that already exist.
- When a later document will address a concern, describe the gap in prose rather than linking to a document that does not exist.
  Update the earlier document with a link when the later document is added.
- In `docs/` prose, use reference-style links and define each target at the end of the file.
- Do not cite an identifier by itself: write `D-03 on naming documents`, not `D-03`.
  When referring to part of a long document, name the section: write `D-03 § Decision Outcome` and set its reference target to `adr/D-03-document-naming-convention.md#decision-outcome`.
- Prefer shortcut reference links when the link text names the target.
  For example, write `[D-03-document-naming-convention]` and define `[D-03-document-naming-convention]: adr/D-03-document-naming-convention.md` at the end of the file.
- Use the full reference form only when the same target needs different descriptive link text.
  For example, `[the document naming decision][D-03-document-naming-convention]` reuses the target of `[D-03-document-naming-convention]`.

## Frontmatter

- In YAML frontmatter, write relationship fields as block sequences, even for one value.
  Do not use flow sequences (`[value]`); frontmatter is data, not Markdown.
