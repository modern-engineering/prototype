# CLI and compiler-layer corpora — expectations beside fixtures

Distilled from cmd/go's script tests, cmd/gofmt, go/parser, and x/tools'
analysistest.

## The script-test anatomy (cmd/go, 925 files)

- One `testdata/script/*.txt` is a txtar archive: the comment section is the
  SCRIPT, the file sections are the FIXTURE TREE, extracted into a fresh
  hermetic $WORK per test (t.Parallel per file).
- Assertions ARE commands: `stdout 'regex'`, `stderr -count=1 'regex'`,
  `cmp file golden` (file may be literally `stdout`), `exists`, `! cmd` for
  must-fail, `[cond]` guards, `#` comments delimit phases whose transcripts
  collapse in failure output.
- Why it beats Go-code e2e for CLIs: the test is a transcript of exactly the
  user-observable contract (argv, cwd, env, exit status, two streams) and
  can assert nothing else — a Go test can quietly reach into internals, a
  script physically cannot. Fixture locality: the broken source file sits
  ten lines above the `stderr 'file.sdl:3:8: …'` assertion, so positions are
  checkable by eye. Marginal cost of case N+1 is one small text file.
- Harness tricks: TestMain re-executes the TEST BINARY ITSELF as the command
  (zero build step, coverage flows); heavyweight infrastructure (module
  proxy) is built ONCE in Go and exposed to scripts via env — scripts
  consume infrastructure, they never build it; the corpus README is
  generated from the engine's self-documenting command structs and enforced
  by a test.
- Public engine: rogpeppe/go-internal/testscript (descendant of cmd/go's;
  industry default — cue, hugo, task). `testscript.Main(m, map[string]func()
  int{...})` needs only a RunMain-shaped entry; UpdateScripts subsumes
  hand-rolled -update; TestWork preserves work trees. A bespoke txtar
  runner starts at ~200 lines but quoting/conds/update-mode creep — cmd/go's
  engine is 2.5k lines for a reason.

## When the Go project still writes Go-code CLI tests

If a shell user can observe it → script. If the assertion needs Go types
(decode + DeepEqual, structural comparisons) or fd/signal choreography →
Go test. If it is setup → Go infrastructure exposed via env. cmd/go keeps
~57 Go-code tests beside 925 scripts — the coexistence is the design.
Do NOT grow CLI surface (a verb that exists only so scripts can assert
something) — that is inverted priorities; keep such assertions in Go.

## Golden pairs and in-fixture expectations

- cmd/gofmt never executes its binary: package-main tests call processFile
  in-process against `testdata/*.input → *.golden` pairs; per-fixture flags
  ride INSIDE the fixture as a magic comment; `-update` rewrites goldens;
  each golden re-runs through the formatter to prove idempotence
  (`runTest(t, out, out)`) — the cheapest strong property a formatter can
  pin.
- go/parser puts expectations inside fixtures as `/* ERROR "rx" */` placed
  immediately after the offending token — the comment's own position defines
  the expected error position.
- analysistest (x/tools): fixtures are real compilable packages; `// want
  "regex"` comments sit ON the offending line; sealing is bidirectional —
  unmatched expectations AND unexpected diagnostics both fail. Suggested
  fixes compare against `.golden` files beside the fixture.
- Doctrine: each compiler-like layer deserves a bespoke few-hundred-line
  harness, written once, that turns that layer's user contract into a corpus
  with expectations beside fixtures. Parser → ERROR comments; analyzer →
  want comments; CLI → script transcripts.

## Layer hygiene

Library packages assert their Go API to sibling users; the script corpus
asserts the shell-level contract; internal verb packages get in-process
tests through their seams (an injected writer/stream struct kills os.Pipe
hijacks and lets exit taxonomies be pinned without the artifact). Never pin
one wording in several suites: the layer that OWNS the wording holds the
matrix; other layers keep one representative case proving the plumbing.
