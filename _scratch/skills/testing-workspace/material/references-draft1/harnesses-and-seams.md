# Harnesses, seams, and synctest — making contracts statable

Distilled from `net/http/httptest`, `testing/{fstest,iotest,synctest,
slogtest}`, `internal/{txtar,nettest,testenv}`, and their users.

## Design moves that make a package testable

- **One-interface behavior slots** consumed identically by users and tests:
  http.Handler, http.RoundTripper, fs.FS, io.Reader/Writer, slog.Handler,
  driver.Connector (database/sql runs an entire fake in-memory DB behind it).
- **Inject the boundary object, not a flag**: http.Server.Serve takes any
  net.Listener; Transport takes any dial func — so httptest swaps the whole
  network without the production package knowing.
- **Lifecycle hooks as seams**: Server.ConnState lets httptest track conns
  for graceful Close; test-only hooks assigned in `_test` init.
- **No clock injection through APIs** — the stdlib rejected threading clock
  interfaces and virtualized time in the runtime instead (synctest).
- **Rule of disk**: bufio consumes io.Reader and never touches disk in tests
  (strings.Reader + iotest's adversarial wrapper matrix); os touches real
  disk because the filesystem IS its contract — and still verifies os.DirFS
  with the shared fstest.TestFS verifier. Disk in tests ⇔ disk in contract.

## Harness API shape (httptest as canon)

- Entry point takes `testing.TB`, registers `t.Cleanup`, converts panics to
  test failures, routes internal logs to `t.Logf`.
- Mutation-then-freeze configuration: "must not be changed after the first
  call to Client, Start…" — enforced by panic. Helper constructors may panic
  on bad args ("for ease of use in testing").
- Offer two altitudes: in-memory end-to-end (Server) and pure in-process
  (ResponseRecorder + NewRequest) — the user picks.
- **Harnesses are themselves tested**, and the self-test doubles as usage
  documentation (TestNewClientServerTest pattern).
- Matrix runners keep ONE test body across variants: run(t, f, modes) over
  HTTP/1/2/3; options as `...any` with a type switch; requireFeature-style
  filters. This is how test count stays low while variant coverage grows.
- Human escape hatches cost little and pay off: a flag that keeps the server
  running for manual poking; a flag preserving extracted work trees.

## Contract-verifier doctrine (fstest/iotest/slogtest shape)

- `func TestX(impl, expectations...) error` — takes NO *testing.T, joins all
  violations with errors.Join; the caller t.Fatal(err)s.
- Verifier + fake pair in one package: TestFS + MapFS; TestReader +
  adversarial wrappers. Fakes verify consumers; verifiers verify producers.
- Probe optional capabilities by type assertion; test only if present.
- Versioning: verifiers only TIGHTEN (a tightened verifier failing a lagging
  implementation IS the flagged breaking change); fakes only grow richer;
  exported harness fields never get repurposed — deprecate in place, add a
  new accessor beside the old.

## testing/synctest patterns (stable, Go 1.25)

- `synctest.Test(t, f)` — bubble with a fake clock (2000-01-01 UTC); time
  advances only when ALL goroutines are durably blocked (bubble channels,
  select, Cond.Wait, WaitGroup.Wait, time.Sleep — NOT mutexes, IO,
  syscalls). Deadlock panics. `Wait()` flushes; `Sleep(d)` = sleep + Wait.
- Wait-before-read accessors make async assertions deterministic one-liners:
  `func (db) numFreeConns() int { synctest.Wait(); lock; read }`.
- `synctest.Sleep(10*time.Second)` crosses idle/lifetime/grace thresholds
  instantly — assert the BUDGET (deadline ≈ configured grace), not mere
  deadline presence.
- Real sockets are forbidden in bubbles (IO-blocked goroutines are not
  durably blocked) — fakes must block on channels, never mutexes; pattern:
  a lock implemented as a 1-buffered channel pair.
- One-line composition wrappers: runSynctest(t, f, opts) composing the
  matrix runner with synctest.Test per mode.
- Comment idiom replacing poll loops: "Connections close asynchronously;
  wait for them to finish doing so." then `synctest.Wait()`.

## txtar — fixture trees and multi-view goldens

- Format: optional comment, then `-- name --` file sections. No possible
  syntax errors; hand-editable; diffs cleanly; public form in x/tools.
- One case = ONE reviewable file holding input AND expectations; archives
  are programmatically rewritable, enabling -update flows that rewrite
  golden sections in place while inputs stay hand-authored.
- Multi-view golden pattern (go/doc/comment): archive comment = config;
  file "input" = source; remaining files = named expected outputs all
  rendered from ONE parse — one fixture exercises the parser and every
  renderer together. Exactly "fewer tests, more API per test".
