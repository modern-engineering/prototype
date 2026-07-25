# Twoshot Follow-up Plan — Deployable Solution + Accumulated Revisions

Persisted 2026-07-13 as the context anchor: the conversation that produced it
may be compacted; everything needed to continue lives here, in `docs/`, and in
git history. This file is read by the architect session AND by every workflow
subagent — keep it current as phases land.

(Restored 2026-07-22 from session context after `_scratch/` was accidentally
deleted; content is the final 2026-07-13 state.)

## Status — night shift 2026-07-13

- Baseline: annotated tag `m1-baseline` (= 43ee0cb, the session start).
  Review the whole delta with `git log --oneline m1-baseline..`.
- **P1 DONE & VERIFIED** (run `wf_e7d06482-c1a`, 7/7 agents, 18 commits
  `74adff4..d5bbd45`): mockup 7 live in examples/sample + solution/sample.sdl;
  section qualifier in ast/parser/printer; image compartments
  Params/Deployment/Extensions/Metadata (On retired, no shim); D-12 ADR;
  cmd/sdl doc.go carries the compilation architecture + phase glossary; both
  modules green, sample golden byte-identical, echo round-trip closed,
  migration errors positioned (verify agent report in workflow journal).
- **P2 DONE — MILESTONE M1 SHIPS** (run `wf_fee8541d-1cc`, 7/7 agents, 37
  commits total at `9a050a8`; spec `_scratch/P2-SPEC.md`). Operator gate +
  architect witness-run both green: `sdl run -extern natsEndpoint=localhost:0
  examples/pingpong` → enact header, provenance-stamped audit lines, 3 apps
  concurrent on application.Runtime, SIGTERM → "shutdown complete", exit 0;
  Mode P (`examples/host` on the encoded image) semantically identical;
  dev-loop ~1.5s warm (A-13 MoE "budget of seconds" met); `sdl run
  examples/sample` → slice teaching error exit 2; unbound-extern gate exit 2.
  D-13 (two-phase enactment on host-agnostic images) + D-14 (drivers ride the
  declaring type) landed with accretion edges.
- **Morning review items (P2 judgment calls to ratify or reverse):**
  1. Round-trip byte-identity pins deliberately weakened to image.Equal +
     bytes-must-differ (unit digests in the governance block force it).
  2. Exit-seam asymmetry: `sdl run` parse/import fault → exit 1 (front half
     shares build's DiagnosticsError contract), link fault → exit 2 (child
     host semantics). Documented in run help text.
  3. `go run ./cmd/sdl`-fronted processes: cmd/go does NOT forward SIGTERM;
     signal the sdl process itself. Doc line suggested (D-13 or cmd doc).
  4. Build.Settings is empty on dir-replace checkouts by design (only
     ordinary module versions recorded, machine-independent goldens);
     installed-binary flows record versions (proxy path proves v0.1.0).
  5. Docs hygiene commit d29e61f (restores D-05 inverse edges on
     A-01/A-09/A-11) is revertable if P2-scope purity preferred.
  6. Mode P / Mode T entered D-13 unbracketed (naming door not marked);
     `Main` exits 0 on clean completion (no long-runningness imposed);
     MakeFunc services have no Shutdown promotion → grace rides Cancel.
- **P3+P4+P5 DONE — PLAN FULFILLED** (run `wf_e9be36cc-f4e`, 9/9 agents,
  spec `_scratch/P345-SPEC.md`; final HEAD `e790804`, 92 commits since
  `m1-baseline`, tree pristine). Combined gate PASS on all 8 validation
  scenarios; architect re-witnessed vet/highlight/completion + the M1 canary
  at final HEAD (exit 0, "shutdown complete"). Landed: schemes as
  self-naming citizens (examples/k8s, both advisory halves proven live,
  D-15 + D-09/D-12 edges); image Canonicalize/Validate + composed-image
  round-trip/echo/wet-path proofs (peer frontend real); `sdl highlight`
  (generated, anti-drift-gated), `sdl completion` (bash/zsh, syntax-checked),
  `sdl vet` (v0 checks incl. unused-symbol, corpus sweep green), `sdl lsp`
  (diagnostics + completion over stdio, scripted session verified).
  Dev-loop re-check 1.34s warm.
- **P345 review items (fleet judgment calls to ratify):**
  1. Checked schemes validate LITERALS only; bare tokens pass typed keys
     (`replicas: notAnInt` rides as a token — the spec's own passthrough
     rule). Near-miss-qualifier vet warning surfaced as an idea, recorded
     nowhere yet (candidate D-15 door or vet check).
  2. `default k8s.Pod { }` refused with a teaching error pointing at
     `default deploy { with k8s.pod { } }` (beyond-spec; per-scheme stanza
     defaults are a door in collectDefault).
  3. Scheme lookup is solution-wide (ignores per-unit imports; D-15 records
     it); unclaimed qualifiers stay SILENT by pinned test (the reevaluation
     trigger for an opt-in advisory note).
  4. image.Validate skips the instance/symbol shared-namespace collision
     (a composed image can echo to a non-recompilable unit; one-line check
     named as the cheap fix) and mirrors the linker's recursive cycle DFS
     (hostile-image depth door, c1a2d55 precedent, shared with linker).
  5. vet accepts a SUPERSET of build (nested sections, dotted refs in
     opaque compartments pass vet); shared-checker door recorded. Highlight
     contextual words are regex-approximate by design; LSP is the real fix.
  6. LSP doors: byte-count columns (vs UTF-16), no initialize gate,
     full-sync only — all documented in-package. Completion sniffs
     "(repeatable)" in flag usage text (soft convention).
  7. D-15 opened the discovered-schemes door BEFORE its original trigger
     (first in-earnest stanza consumer) on authoring-loop grounds; D-12's
     old "error nowhere" sentence now slightly over-promises (left as the
     decision-time record).
  8. onelayer simswap hostsmoke is timing-sensitive under cross-module
     CONCURRENT test load (green serially; pre-existing, untouched).
- **NIGHT SHIFT COMPLETE 2026-07-13 ~08:45**: all planned phases landed and
  verified. Review the delta with `git log --oneline m1-baseline..` (92
  commits) or `git diff m1-baseline --stat`. Specs preserved:
  P2-SPEC.md, P345-SPEC.md beside this file.
- **TEST REMASTER (branched session "m1-fleet-tests-remastered",
  2026-07-13 morning)**: tag `nightshift-complete` = e790804 marks the
  baseline. User doctrine: committed tests are USER-contract pins enabling
  maintainer change (never bug hunts, never CLI-artifact tests inside
  libraries); fewer tests covering more exported surface together; trivial
  cases as Examples; synctest for concurrency; test doc comments never open
  with the function's name. Survey digests at `_scratch/TESTS-SURVEY.md`
  (stdlib hash doctrine, httptest/fstest/synctest/txtar harness patterns,
  cmd/go script tests + testscript judgment, merciless repo inventory).
  Rulings + track orders at `_scratch/TESTS-SPEC.md` (R1 testscript
  adopted; R2 e2e wording dedup; R3 application prune; R5 host signal
  seam; R6 base writer seam; R7 lsp run(ctx,in,out); R9 examples/host
  smoke; R10 doc-comment sweep ~170 sites).
  **DONE — 53 commits `nightshift-complete..2514658`, TV gate PASS, tree
  clean, architect re-witnessed short suites green.** Corpus delta: test
  funcs 249→228 (−8.4%), Examples 4→10, Go test LOC 15,262→15,162, +15
  transcript scripts (743 lines; net material +4.2% — the growth is
  fixture-beside-assertion transcripts plus FIRST-TIME pins: loaderflags,
  shutdown grace budget under synctest, exit taxonomy in-process, lsp
  interrupt, examples/host smoke). Review-worthy judgment calls: R3
  extended to NewHTTPServer (rule-authorized, beyond the enumerated list);
  Proc.String on non-Stringer runners now renders the dynamic type (race
  fix changed exported behavior); custom `status` script command because
  testscript's `!` cannot see exit codes (1-vs-2 taxonomy in transcripts);
  four scripts drive repo examples via $SDLREPO (transcript↔layout
  coupling); e2e wording matrices deleted in favor of solution's tables +
  one representative script per family. Full per-track deletion
  justifications in the workflow reports (wf_08d10fa2-fba).
- **Night-shift authorization** (user, 2026-07-13, absent ~8h): proceed
  through the phases at the architect's discretion — the M1 user-feedback
  gate is waived for tonight and replaced by a morning review packet (this
  Status section, kept current + git log against `m1-baseline` + the ADR
  set). Standing guidelines to honor in EVERY phase: INCOSE practice with
  explicit verification AND validation loops (verify = build/tests/round
  trips; validate = exercise the stakeholder scenario end to end against
  A-13's ConOps/MoEs); Go mechanism/construct inspirations anchored to named
  reference projects (go test harness, debug.BuildInfo, flag,
  analysis.Analyzer, cmd/go dispatcher, controller-gen precedent); ample
  documents at every goal level (mission analyses, milestone ADRs, package
  overviews, code comments that carry constraints).
- Chain discipline: never end an architect turn without either a background
  task in flight (its completion re-invokes the session) or the plan
  fulfilled/blocked; review each phase before launching the next.

## Mission

Ship a functional, deployable solution end to end on the twoshot line, then
layer the accumulated revisions. The user reviews a SIMPLE loop before any
onelayer-scale deployment.

**Milestone M1 — deployable Ping-Pong:**

- One solution directory (examples/sample; ping+pong from the examples/ff
  catalogue), authored in mockup-7 SDL.
- `sdl build` → image (new body anatomy, governance block, typed scalar
  outputs).
- Deploy the image to ONE host process hosting all instances concurrently via
  `application.Runtime` (the N-app runner; the plan originally said
  "application.Group" — no such type exists).
- Provisioning: ONE dummy attach-style provision type whose driver returns
  hard-coded outputs, wired into app params through the image. No real
  infrastructure.
- Explicitly deferred: onelayer-scale deployment; real NATS/Postgres drivers.

## Method

- The main session is the systems architect (INCOSE): decompose, spec, verify,
  decide; record rationale in docs/commits, not chat.
- Subagents play the A-13 stakeholder roles (application developer, solution
  engineer, platform engineer, operator, demo engineer, framework maintainer)
  through workflows; long-horizon state lives in this file + git history.
- Commits: fine-grained, imperative, log style `pkg: summary` (see git log:
  `sdl/parser: …`, `solution/image: …`, `cmd/sdl: …`, `docs/…`). Never push.
- Prototype mindset (root CLAUDE.md): permissive sane default now, opt-in
  restriction sketched as a door with its reopening trigger.

## Locked decisions (2026-07-12/13 dialogue)

1. **Statement body = mockup 7.** Top-level fields are a CLOSED per-verb
   scheme (like a k8s resource's top-level keys), e.g. `location:` on deploy —
   no `{}` block. `params { }` holds the catalogue-typed application
   parameters (the "spec", driven by the element). `with <qualifier> { }`
   (RENAME of `on`) are named controller/community scheme blocks — several per
   statement allowed, ADVISORY (a controller that doesn't recognize a
   qualifier ignores it; unknown stanzas ride the image opaquely).
   `metadata { }` unchanged. Anatomy is UNIFORM across deploy/provision and
   the default layers. Qualifiers are DOTTED idents (`with k8s.pod`) —
   hyphens don't lex (collide with signed-INT scanning); dotted needs zero
   lexer change and gives natural namespacing. Spelling provisional.
2. **`with`-stanza schemes (P3).** Discovered VALUE schemes — never an
   `init()` registry (D-06/D-11 stance). Compile-time checked when the scheme
   package is importable; tolerated opaque otherwise. Controllers LOOK UP
   stanzas at enactment and may require several (`k8s.pod` + `k8s.workload`
   + company-specific). Stanzas are reusable building blocks shared by
   community/platform teams — a BMA/A-13 enhancement.
3. **Host-agnostic IR; two host modes.** The IR never names a host. It must
   carry the values (direct, `var`, `extern`, provision outputs) that ANY
   host binds when parameterizing the applications it runs (`Set` +
   `application/loader` are the wet-binding blocks). Mode P (production): one
   PREBUILT host binary, built against a catalogue, serves MANY solutions —
   app-dev and solution-engineering cadences are disparate. Mode T
   (tailored): a generated+built host per invocation, mimicking `go test`
   mechanics (e2e harness, dev loop, `sdl run`). Per-production-solution
   codegen never happens.
4. **Enactment = PROVISION then DEPLOY.** The wet half is TWO distinct
   phases: provisioning of backing services (after link, before apps serve),
   then deployment of apps. Same or different binaries — the controller
   decides; reification packages must never assume same-binary. "Reconcile"
   names the convergence discipline (observe/diff/converge/prune, D-08), not
   a phase.
5. **Provisioning execution tied to declaration.** The driver rides the
   `ProvisionType` value (mirror of `Descriptor.Make`), resolved through the
   host's catalogue/`Set`, run wet at PROVISION. Attach-first (verify +
   outputs); slice lifecycle (create/mutate/destroy, delete protection,
   prune) later. "NATS: admin creates the solution's account if absent" is
   the acceptance spec for the first REAL slice driver (post-M1; M1 uses the
   dummy attach).
6. **Typed outputs, scalar, in the IR.** Output declarations carry a type in
   the image; the linker checks reference sites at compile; reification
   packages expose typed read (host) / typed write (driver) across the
   binary boundary. Structured outputs deferred (need composite values).
7. **Governance block ≠ symbol table.** The image embeds a
   `debug.BuildInfo`-style governance/provenance block: solution identity +
   monotonic generation, pinned catalogue (module paths, versions, schemas
   compiled against), sdl tool + prototype library versions, source file set
   + content hashes, build stamp, guardrails (delete protection) — contents
   serve skew detection, audit, and blast-radius tracing. SYMBOLS (extern
   MUST-bind, var overridability, sensitivity/taint) are semantic IR: they
   stay in the symbol table, never in the provenance block.
8. **Programmatic IR composition is first-class.** A public image
   construction API is a peer frontend (text notation now; visual and
   code-first builders later). Analogy: SDL text ↔ a C-like low-level
   language; plumbing CLI ↔ the assembler; encoded IR ↔ bytecode.
9. **Codegen stance.** Minimize END-USER codegen. Repo-internal generated
   code committed here (e.g. buf/protobuf for the IR encoding) is
   acceptable. Encoding choice itself deferred — JSON stays until typed
   outputs or the builder API demand more.
10. **Lint is source-level.** A source lint verb (candidate `sdl vet`,
    Go-style) over `.sdl` sources, sibling of `fmt`; at least basic checks
    wanted. IR-side validation is NEVER called lint (it is plumbing under
    `sdl image …`).
11. **Tooling, layered; no Cobra.** Layer 0 now: `sdl highlight <vim|vscode>`
    GENERATED from `sdl/token` (+ the section vocabulary), golden-tested so
    it cannot drift from the grammar — never a hand-written static file;
    `sdl completion <shell>` introspecting the cmd/go-style dispatcher
    (`base.Commands` + `FlagSet.VisitAll`). Layer 1: a BASIC LSP (positioned
    diagnostics from the real scanner/parser/linker; simple completion);
    higher fidelity deferred. Velocity over purity; deps swappable later.
12. **Phase glossary.** Build acts: GENERATE → BUILD → LINK → EMIT (front
    half generates the program, toolchain builds it, the generated back half
    links the solution and emits the image). Enactment acts: PROVISION →
    DEPLOY. "compile" is reserved for the whole; the `MainCompile` /
    "compile back half" naming is flagged as a door, not renamed now.

## Sequenced phases

**P1 — Anchor & vocabulary** (prerequisite for everything)

- Mockup 7 appended to `solution/sample.sdl` (supersedes 6; file stays an
  unpolished accreting scratchpad).
- Grammar: optional dotted qualifier on sections (`ast.Section` gains it;
  parser + printer + tests). The parser stays compartment-agnostic — no new
  keywords; any section may carry a qualifier syntactically; the linker
  decides which section names allow one.
- Linker: `params { }` routes to slot validation (bare root keys are no
  longer catalogue params — positioned error guides migration); top-level
  fields validate against a closed per-verb table (deploy: {location};
  provision: {} for now); `with <q>` → opaque extensions compartment keyed by
  qualifier; `metadata` unchanged; `on` removed; the default layers
  (verb/type) descend into `params { }` and merge extensions per qualifier,
  preserving merge order (catalogue < default verb < default type <
  instance).
- Image record compartments follow: Params / Deployment (top-level fields) /
  Extensions (map[qualifier]) / Metadata. Ripple: echocmd re-render, imagecmd
  queries, e2e fixtures, examples/sample migration, onelayer .sdl fixtures
  (mechanical only — onelayer stays a superseded corpus, do not redesign).
- Code-tree compilation-architecture doc + the phase glossary (cmd/sdl
  overview doc; solution/doc.go aligned; README pointer).
- Docs ripple: new ADR (next free D number) deciding the mockup-7 anatomy +
  `with` rename + advisory stanzas + closed top-level fields, refines A-09;
  D-07's open body-compartments path resolved to a reference; A-09 gains the
  refined-by edge; family READMEs updated (docs/CLAUDE.md rules apply).

**P2 — Milestone M1** (deployable ping-pong)

- Reification packages: read an image into typed records/symbols/refs;
  PROVISION→DEPLOY ordering from the reference DAG; site binding for extern
  symbols (smallest honest mechanism, e.g. a tiny site file or flags).
- Host SDK: bind image values into `Make()`d services via
  `application/loader`; run all instances under `application.Group`; clean
  SIGTERM shutdown. Mode P shape = these packages linked by a platform
  binary; Mode T = `sdl run <dir>` generating a tailored host main via the
  existing generate/build machinery (A-13 dev-loop: one command, sources →
  running solution).
- `ProvisionType` gains its driver value (attach-only v0) + a
  CheckProvisionType-style citizenship harness; dummy attach type in
  examples returning hard-coded outputs.
- Typed scalar outputs in the image + link-time reference checks.
- Governance block in the image (+ imagecmd query surface).
- ADRs: host-agnostic enactment with two host modes and PROVISION/DEPLOY
  phases; driver-on-type (typed outputs may fold into it).

**P3 — with-stanza scheme mechanism** — discovered scheme values, advisory
unknowns, compile-time check when importable; A-13 BMA building-blocks
update; D-09 citizenship classification decided here.

**P4 — Programmatic IR** — public builder API over `solution/image`;
frontend-layering recorded (A-09/A-14 extension); round-trip guarantees.

**P5 — Tooling** — `sdl highlight` (generated), `sdl completion` (static),
`sdl vet` (source lint skeleton with a few real checks), basic LSP
(`sdl lsp`: diagnostics + simple completion).

Order: P1 → P2 strictly (image shape first). P5 highlight/completion may
start once P1 grammar settles. P3/P4 after the M1 user-feedback gate — gate
WAIVED for the 2026-07-13 night shift (see Status); decisions land as
proposed ADRs the user reviews in the morning.

## Deferred (recorded, non-blocking)

- Composite/structured values and structured outputs (parseValue needs an
  LBRACE arm).
- IR encoding format (JSON now; buf/protobuf door open — repo codegen OK).
- D-09: with-schemes as citizens vs a parallel discovered namespace.
- Override-audit record home (A-10/A-13 open questions stand).
- LSP fidelity beyond basic; catalogue-aware dynamic completion.
- onelayer-scale deployment; real NATS slice driver (acceptance spec above);
  Kafka attach-only corpus edit.
- Slice lifecycle (create/mutate/destroy, delete protection, prune).
- `MainCompile` / "compile back half" rename.
- `params` block name vs A-03's "parameters" wording (naming door).

## Key code facts (verified 2026-07-12/13 — saves re-investigation)

- The parser is compartment-agnostic: `on`/`metadata`/`params`/`with` are NOT
  keywords; a Section is a generic (name, body) pair; `params { }` parses
  TODAY. Legal section names are a hard-coded {on, metadata} test in the
  linker (solution/compile.go ~1128). [P1 replaced this with vocabulary
  routing; kept as the pre-P1 record.]
- `ast.Section` has a single Name slot; `with k8s.pod {` needs a qualifier
  field (parser.go ~382). [Landed in P1.]
- `parseValue` (parser.go ~269) has no LBRACE arm — composite values are a
  parse error; composites exist only as flattened dotted keys.
- Provision params bind per-statement to a fresh throwaway flag surface at
  compile; canonical values land in `image.Record.Params`. No globals
  anywhere; the "single config per host" worry was unfounded.
- Outputs today are name-only (`image.SymbolRef{Symbol,Output}`, empty
  value; `Output{Name,Doc,Sensitive}` untyped). [P2 added Type.]
- No enactment code exists in the authoritative line (CLI verbs: build,
  echo, fmt, image edit/records/symbols). `onelayer/` is superseded prior
  art: its host runs a real dependency-ordered provision→deploy loop, but
  its driver echoes a site file (touches no infrastructure). [P2 landed
  solution/enact + solution/host + sdl run.]
- CORRECTED 2026-07-13: there is NO `application/loader` package and NO
  `application.Group`. The wet-binding block is
  `application/loader/loaderflags` (NewFlagSet/Parse/Parser, priority
  sources, called only by onelayer/host), and the N-app concurrent runner is
  `application.Runtime` (RuntimeWithContext/Run/Wait/Shutdown(ctx
  verbatim)/Cancel; no signal handling in the library — callers own it).
- `cmd/sdl` is a hand-rolled cmd/go-style dispatcher (`base.Command`); no
  cobra; go.mod deps: x/sync, x/mod, x/tools only. [Remaster added
  go-internal/testscript as a test dependency.]
- The image pins its catalogue (`CompileConfig.Catalogue`);
  `work.go checkPrototype` already warns on tool-vs-library skew at build.
- `sdl/token` exports the whole lexical vocabulary (keywords iterable;
  IsKeyword/IsLiteral/IsOperator) — the highlighter generator's source of
  truth.
- Module-less builds generate a probe program then the compiler into the
  same solmain.go; units are parsed twice by design (CLI fast-fail +
  verbatim embedding).
- Lexical gotcha: hyphenated bare tokens (`aws-us-east-1`, `k8s-pod`) do NOT
  lex — `-` only introduces signed numbers. Use dotted idents or strings.

## Observation ledger (2026-07-12 batch → outcome)

| # | Observation | Outcome |
| - | ----------- | ------- |
| 1 | Kafka attach-only | revision; deferred with onelayer milestone |
| 2 | NATS slice = create-if-absent | planned (dismissed); later driver's acceptance spec |
| 3 | provision flags → globals | premise corrected (per-statement surfaces); obviated by driver-on-type |
| 4 | execution tied to declaration | decided → driver-on-type + typed outputs (locked #5, #6) |
| 5 | see provisioning work | M1 dummy attach now; real driver later |
| 6 | params block | locked #1 (mockup 7) |
| 7 | `on k8s` extensions | locked as `with`-stanzas, discovered schemes (P3) |
| 8 | compilation architecture doc | P1 |
| 9 | phase names | P1 glossary + PROVISION/DEPLOY split |
| 10 | shell completion | P5 (static, no cobra) |
| 11 | syntax highlighting | P5, GENERATED from grammar, golden-tested |
| + | LSP | P5 basic |
