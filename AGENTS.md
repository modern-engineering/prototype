# Repository Guidance

## Documentation

- Establish the mission before developing an operational concept.
- Analyze the problem before choosing a solution.
- Keep each artifact traceable to the artifact that motivated it.
- A new document takes the next free number in its family; never reuse one.
- Adding or renumbering a document updates its family `README.md` index in the same change.
- Follow D-03, D-04, and D-05 for document naming, lifecycle metadata, and relationships.

## Cross-References

- In `docs/`, link only to documents that already exist.
- When a later document will address a concern, describe the gap in prose rather than linking to a document that does not exist.
  Update the earlier document with a link when the later document is added.
- In `docs/` prose, use reference-style links and define each target at the end of the file.
- Prefer shortcut or collapsed reference links when the link text names the target.
  Use the full reference form only when the label must differ from the text.
- Pair an identifier with words rather than citing a bare identifier.
  For a long target, name the section and point the reference definition to its fragment.
- Keep inline links to `README.md` index tables, where the row supplies their context.

## Frontmatter

- In YAML frontmatter, write relationship fields as block sequences, even for one value.
  Do not use flow sequences (`[value]`); frontmatter is data, not Markdown.
