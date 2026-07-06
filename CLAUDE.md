# Working in This Repository

The project is a Go application library (the core) and, above it, a solution
layer: definitions compiled against a catalogue into an intermediate
representation that deployment environments reconcile. The design record lives
in `docs/` (analyses, requirements, decisions); read `docs/CLAUDE.md` before
authoring there.

## Mindset

- Prototype-grade evolution: prefer the permissive sane default, sketch how a
  future opt-in restriction may look, and move on. Breaking changes — syntax
  included — are cheap at this stage; do not over-decide upfront.
- Keep doors visibly open: record open paths where they will be found (mockup
  comments, an ADR's More Information, an analysis's open questions), with the
  trigger that would reopen them.
- Tangibility drives design: mockups and runnable sketches beat abstract debate.
  `solution/sample.sdl` is the syntax scratchpad; keep it current with the
  concepts, never polished.
- Concepts before syntax: grammar is instrumental and revisited freely; the
  concepts (catalogue, intermediate language, reconciliation, staged symbol
  binding) are what must hold.

## Scope

- Go-centric backend services built on the application library; the library is
  the project's core, and the catalogue admits only its citizens.
- Backing services (brokers, databases, identity) are substrate guaranteed by
  platform teams; solutions slice into or attach to them, never own them.

## Sources

- Field evidence from proprietary systems informs this work but stays anonymous
  everywhere in the repository — docs, comments, and commit messages alike.
