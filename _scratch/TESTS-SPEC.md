# Test Remaster Spec — Committed Tests as User Contracts

(Restored 2026-07-22 from session context after _scratch was accidentally
deleted; original written 2026-07-13. The remaster it ordered is COMPLETE —
53 commits nightshift-complete..2514658 — so this is now a record, not an
order.)

Architect's spec for the test-remaster fleet. Read `_scratch/PLAN.md` (house
rules, repo shape), then `_scratch/TESTS-SURVEY.md` (the four survey digests —
your track's sections are REQUIRED reading; they carry file:line evidence for
every claim below). Baseline tag: `nightshift-complete`. Never push; never
touch `.claude/` or `_scratch/`; onelayer/ is out of scope entirely.

## Doctrine (binding on every track)

1. A committed test exists so a future maintainer can change the package with
   the guarantee that every committed contract with its USERS is upheld;
   breaking changes must fail a test or force a visible test amendment. Tests
   exercise the package AS ITS USER (sibling packages, external module users
   — not the built CLI artifact, except in the CLI's own script corpus).
2. FEWER tests, each covering MORE of the exported surface working together
   (hash-family model: TestGolden exercises Checksum+New+Write+Sum together;
   maphash's TestHashGrouping runs seven write paths to one equality). Names
   claim a scenario or property, never a symbol: `TestSeededHash`,
   `TestUnbalancedDefaultsFold` — not `TestParse`, `TestSetString`.
3. Unit tests assert the contract with callers to enable rapid iteration;
   they are NOT bug hunts. A regression pin is renamed to the property it
   protects; the issue/commit reference demotes to a comment.
4. Part of the contract lives in doc comments — tests execute documented
   sentences (fnv's testIntegrity executes the big-endian doc line). When a
   doc comment promises nothing testable, fix the doc or the API.
5. Trivial assertions become runnable Examples (they are tests AND docs).
   Example doc comments — like ALL test/bench/fuzz/example doc comments —
   NEVER open with the function's own name; open with the contract clause
   ("Shutdown passes the caller's context to every Shutdowner, alive." not
   "TestShutdownHonorsContext pins…"). Underscores appear only in Example
   names for godoc attribution.
6. Development scaffolds (kind sweeps, check-manually probes, commented-out
   skips, assert-nothing benchmarks) do not get committed. While you work,
   DO run uncommitted exercises — play with artifacts as a user/operator —
   and throw them away; the committed suite is only the contract.
7. testing/synctest (stable, Go 1.25) for context/goroutine/time code: fake
   clock, durably-blocking semantics, wait-before-read accessors
   (database/sql's numFreeConns pattern). No sleeps, no poll loops. Real
   time/sockets/signals stay ONLY in explicitly-marked kernel-integration
   pins (one per boundary), never as the default idiom.
8. Harness discipline: in-repo Check* harnesses take `testing.TB`, are
   self-tested (accept + reject via the failer pattern), and live in the
   production package (single-import ergonomics for catalogue citizens —
   the standing convention). A harness that is untested and uncalled is
   worse than none.
9. Goldens come from an oracle or are pinned as lock-ins with intent stated
   in the file header; -update flows rewrite them; failures show diffs.
   Disk appears in tests only when disk IS the contract (fmt/vet/load/work);
   otherwise test through io/fs seams in memory.
10. When a contract is hard to state through the exported surface, that is a
    SIGNAL to redesign the surface (seams: io.Writer injection, signal
    channel, ctx-taking run functions) — not to write a contorted test.
    Production API changes in service of this are in scope and expected;
    grep the whole repo before deleting/altering any export.

## Architect rulings (contested calls, decided)

- R1 **testscript adopted**: `github.com/rogpeppe/go-internal/testscript`
  (v1.14.1) becomes a test dependency of cmd/sdl. invoke() is already
  RunMain-shaped; the test binary re-executes as `sdl`. Reopening trigger
  (record in the harness doc): if the dependency chafes at prototype pace,
  fall back to a bespoke txtar runner — the scripts themselves are
  engine-portable text.
- R2 **e2e dedup**: the linker's diagnostic WORDING matrix lives in
  solution's tables only. cmd/sdl keeps ONE representative positioned-
  diagnostic script per family proving the plumbing (stderr relay, exit
  code, position rendering through the generated compiler). vetcmd's
  shadowing stays (stated anti-drift rationale).
- R3 **application prune**: delete exports with zero in-repo users after a
  repo-wide grep — candidates per survey: ShutdownSignal, ParseFromArgs,
  ParseFromEnvs, ParamParser, ParseTo, Loop, LoopCount, Terminate,
  HealthChecker. A downstream TEST double using one (solution/host) is
  fixed in the same commit. Anything kept gets a test or an Example. Root
  main.go + main_test.go: delete outright.
- R4 **parameter TODO doors** ("maybe panic instead of silently ignoring"):
  resolved to the permissive silent no-op (prototype default); the
  consolidated capability table PINS that choice and the comment records
  the panic door + trigger.
- R5 **host signal seam**: `host.Config.Signals <-chan os.Signal` (nil ⇒
  subscribe real SIGTERM/SIGINT). Shutdown tests move under synctest and
  assert the grace BUDGET (deadline ≈ Grace), keeping exactly ONE
  unix-tagged real-SIGTERM kernel pin.
- R6 **base/invoke output seam**: thread io.Writer(s) through dispatch
  (survey 8a; in-repo precedent solution.CompileConfig.Output/Stderr).
  Kills the os.Pipe stdout hijack; base gains in-process exit-taxonomy and
  help-rendering tests (DiagnosticsError→1, RelayedExit relay — today
  pinned only through the artifact).
- R7 **lspcmd**: factor runLsp into run(ctx, in, out) with the pipe/
  AfterFunc lever inside; synctest test pins cancel-unblocks-pending-read
  (the 9c0cc73 behavior, currently unpinned).
- R8 **httpapp**: readiness dial replaces the 1s sleep; listener injection
  only if the dial proves insufficient (record rationale). BenchmarkGo:
  delete. new_reflect_test.go: replace with the ~20-line two-kind contract
  test; keep the value-vs-pointer-receiver rationale as a real code comment
  (this answers the file's TODO(@claude) — note it in your report).
- R9 **examples/host**: gets a -short-skipped smoke (build, run against the
  committed pingpong image with -extern, expect exit 0 after TERM) in the
  cmd/sdl e2e tree, where the toolchain harness lives.
- R10 **doc-comment sweep**: every track rewords its own tree's test doc
  comments to open with the contract clause (survey counts ~170 sites).
  Also fix ExampleRuntime_cancelsOnError's self-misnaming comment.

## Tracks

**T1 `application-family`** (FIRST, alone — its deletions ripple):
application, application/parameter, application/loader/loaderflags, root
main. Work: R3 prune (grep-verified, fixing downstream references in the
same commits); R8; parameter twin-test consolidation + R4; loaderflags gets
its first contract suite (Parse priority across two sources, foreign-set
panic, already-parsed error, IsSet, PendingParams, Defaults) and CheckParser
converts to testing.TB + failer self-test + a real caller (or dies —
justify); MakeFor/MakeFunc contract test replacing the kind sweep; runtime
tests already model synctest — extend, don't rewrite; ExampleNewSet; R10
sweep over the tree. VERIFY: repo-wide build + full test suite (your
deletions touch others' trees — you own the fallout THIS stage).

**T2 `grammar-tree`**: sdl/token, sdl/scanner, sdl/parser, sdl/printer,
sdl/ast. These suites are models to copy — do NOT shred. Work: token grain
consolidation (String/Lookup/predicates fold into one vocabulary-contract
scenario; Keywords already cross-checks Lookup); R10 sweep; promote a
trivial case to an Example where it teaches (e.g. token vocabulary,
printer canonical form); consider (judgment) the parser ERROR-comment
corpus door from the survey — do NOT build it now, record it as a door in
the parser test file header if you agree.

**T3 `solution-family`**: solution, solution/image, solution/enact,
solution/host. Largely close to doctrine — targeted work only: R5 (signal
seam + synctest grace-budget tests + one real-SIGTERM pin); Examples
promotion (ExampleUnpack, ExampleValue, keep ExampleMainCompile company);
R10 sweep (the densest: compile_test, plan_test, host_test — reword with
craft, each comment states the clause the test holds); image/enact/host
keep their kits; verify harness discipline holds (Check* self-tests
present). Do not touch cmd/sdl.

**T4 `cli-corpus`** (two sequential agents):
- T4a `seams-and-engine`: R6 (writer seam through base + the 10 verb
  packages, mechanical); R7 (lsp run factoring + synctest pin); R1
  (go get go-internal, testscript.Main wiring beside the existing
  TestMain build — the Go-shaped e2e keeps the built binary), plus 2-3
  PILOT scripts (one diagnostics case, TestBuildStdout, one usage-fault)
  proving stdout/stderr/exit/UpdateScripts mechanics end to end.
- T4b `corpus-migration`: migrate the script-shaped families to
  testdata/script/*.txt per the survey's list (diagnostics families ×R2
  dedup — keep one representative per family, DELETE the wording-matrix
  re-assertions that mirror solution's tables; stream/exit tests:
  BuildStdout, BuildDeterminism, FmtExamples, BuildVendored,
  BuildOutputAtomic, BuildOutputDevice, generation usage-fault;
  RunUnboundExtern, RunSample, RunCompileFault, RunNoModule); extract the
  six round-trip choreographies into ONE table-driven roundTrip helper
  (Go — image.Equal is the image package's contract exercised as user);
  dedupe the bash-completer simulation (keep the testTree one, script or
  delete the e2e twin); R9 examples/host smoke; base/buildcmd/runcmd
  coverage lands via the corpus + T4a's in-process base tests; R10 sweep
  over cmd/sdl. Hermetic infra (writeProxy) stays Go, exposed to scripts
  via env (cmd/go pattern).

**T5 `examples`**: examples/ff, substrate, k8s (host smoke is T4b's).
Light: R10 sweep; ensure each example package's tests read as harness-
consumption demonstrations (they already do); add one Example where a
shape-pin reads better as one.

**TV `verify`** (after all): repo + onelayer builds/vet/tests -count=1
non-short; script corpus green + UpdateScripts round-trip (run with -update
on a copy, expect no diff); goldens byte-stable; gofmt/deno; M1 canary
(build sdl, run pingpong with -extern, SIGTERM → exit 0); doctrine
compliance mechanically: grep for test doc comments opening with their own
func name (expect ZERO outside onelayer), underscore-named non-Example
tests (zero), assert-nothing benchmarks (zero); count the corpus: test
functions and test LOC before (at `nightshift-complete`) vs after — the
doctrine PREDICTS fewer, denser tests; report the delta. Fix trivial
breaks only; structural → concerns.

## Report contract (every track)

handoff = what later stages must know (APIs changed, files moved, script
names). report = what you deleted/consolidated/added and WHY, framed as
contract statements; list every export you deleted (R3) and every test you
deleted under R2/doctrine-6 with one-line justification each — the user
will audit the judgment, not just the diff. concerns = judgment calls that
deserve the user's eye.
