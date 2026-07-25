# Findings — cmd/sdl test corpus (e2e + internal packages)

Ranked. Paths relative to the module root.

## Layering

1. The e2e re-pins the linker's diagnostic wordings the solution suite already owns: cmd/sdl/e2e_test.go:854-866 (TestProvisionDiagnostics) and :677-686 duplicate solution/compile_test.go:1802/1827/1844/1852/1860/1597 verbatim; the binary's only claim is that one fault crosses the process boundary positioned at exit 1 — one representative line proves it, seven make a minutes-slow change-detector on library message text.
2. TestRunPingpong re-pins seven verbatim host audit lines that solution/host/compose_test.go:153-161 already pins: cmd/sdl/run_e2e_test.go:76-95 should assert the wiring (first delivery, SIGTERM relay, exit 0, empty stdout) and one audit line proving the audit reached stderr, not the host runtime's transcript format.
3. cmd/sdl/internal/base has no test file at all: Name/LongName derivation from UsageLine (base.go:76-98), Lookup, AtExit/Exit ordering, and the three error types every sibling suite type-asserts are load-bearing (completion generation walks Command.Name()), yet pinned only through neighbors.
4. work.Detect's three-mode classification and Context.Env's GOFLAGS/GOWORK pinning — the package doc's central promise — have no direct test; they are exercised only through full toolchain e2e builds, when a temp-dir table against `go env` would pin them in milliseconds (cmd/sdl/internal/work/context.go).
5. TestBuildSample and TestImagePlumbing pin the records table's alignment and headers through the binary (cmd/sdl/e2e_test.go:1353-1354, :1225-1248) when imagecmd's TestRecordsTable pins the layout byte-for-byte (cmd/sdl/internal/imagecmd/imagecmd_test.go:118-130); the e2e wants presence of names, not presentation owned below.
6. The lsp suite re-pins vet's finding wordings verbatim (cmd/sdl/internal/lspcmd/lsp_test.go:184-187 vs vetcmd's TestVetFindings); the lsp contract is "publishes exactly CheckSource's findings" — assert the count and anchors, or one wording, not two copies of the text.

## Comment and scaffold violations

7. Nearly every non-trivial doc comment in the corpus opens with its own test's name — cmd/sdl/e2e_test.go:101, run_e2e_test.go:21, invoke_test.go:11, internal/load/load_test.go:28, internal/work/work_test.go:12, internal/vetcmd/vet_test.go:43, internal/lspcmd/lsp_test.go:116, internal/echocmd/echo_test.go:48 — the content is good rationale; the first word is the defect, every opener needs rephrasing.
8. The "(a)".."(f)" plan letters committed into e2e doc comments (cmd/sdl/e2e_test.go:101, :193, :314, :482, :1529, :2036) point at a dev-time acceptance list that exists nowhere in the repository; a maintainer cannot dereference them — the property statements stand alone, drop the letters.
9. echo_test's pkg helper claims a "schema-less catalogue pin: echo reads only Path and Name" yet builds an element X (cmd/sdl/internal/echocmd/echo_test.go:31-33); whatever X satisfies (decode validation, the not-pinned check) the comment must say, because right now it contradicts the body.

## Structure and fixtures

10. gen's goldens are inline consts with a ~-for-backquote substitution hack (cmd/sdl/internal/gen/source_test.go:16-76, host_test.go:15-65); the house pattern is testdata files with -update (completioncmd, highlightcmd, the e2e itself) — moving them kills the ReplaceAll hack and gains refreshability.
11. Six e2e round trips repeat the same ~40-line build→echo→fmt -l→rebuild→Equal→bytes-differ block verbatim (cmd/sdl/e2e_test.go:489, :578, :734, :875, :1004, :1399); one roundTrip helper returning both images would leave each test only its vertical-specific assertions — one composition point, the harness shape.
12. Prose sub-test names that could never be Go identifiers run through the internal suites — "malformed import does not panic" (load_test.go:43), "prototype as main module" (work_test.go:74), "signalled wind-down" (exec_test.go:40), "hangup without shutdown" (lsp_test.go:309) — rephrase identifier-ish or Field=Value, or inline.
13. TestInvokeExitCodes wraps every row in t.Run when the failure message already prints invoke(%q) (cmd/sdl/invoke_test.go:37-43); a plain loop beats the ceremony, and the prose names ("bare command group") duplicate what the args show.
14. assertBashCompleter duplicates completioncmd's bash-simulation driver heredoc-for-heredoc (cmd/sdl/completion_e2e_test.go:72-89 vs internal/completioncmd/completion_test.go:234-250); either the e2e trusts the golden bytes plus the internal simulation, or the driver becomes one shared harness.
15. The -short skip boilerplate repeats 24 times in e2e_test.go alone (26 hits of testing.Short); one skipShort(t) helper, or a gate in TestMain, states the suite's contract once.

## Testability signals (owner's call, reported not fixed)

16. TestEchoFaults greps 22 substrings of fmt.Errorf faults echo itself mints (cmd/sdl/internal/echocmd/echo_test.go:394, echo.go:145-246), and imagecmd's fault tables do the same (imagecmd_test.go:156, :206) — grep-only tests indict the API; a typed render-fault checked via errors.AsType would carry these.
17. TestVetPathFaults pins err.Error() == "1 error" (cmd/sdl/internal/vetcmd/vet_test.go:415, vet.go:155) — a hand-rolled counter error asserted by string; the fault-vs-diagnostics split deserves a type, not a spelling.
18. completionScript swaps os.Stdout through a pipe to capture the dispatch route (cmd/sdl/completion_test.go:56-77) because invoke and runCompletion hardwire os.Stdout (main.go:139, completioncmd/completion.go:94); the missing writer seam is the design gap the contortion reveals.

## Minor

19. runSDLEnv duplicates runSDL except for the env append (cmd/sdl/e2e_test.go:82-99 vs :1670-1688); runSDL(t, dir, args...) can delegate to runSDLEnv(t, dir, nil, args...).
20. load_test hand-rolls equal over string slices (cmd/sdl/internal/load/load_test.go:114-124) in a Go 1.25 module whose sibling suites already use slices.Equal.
21. runSDL type-asserts *exec.ExitError (cmd/sdl/e2e_test.go:92); errors.As is the house idiom everywhere else in the corpus.
22. TestRunPingpong's watchdog kill (cmd/sdl/run_e2e_test.go:45) is orchestration to dodge a hang; defensible for salvaging a child-process transcript, but note the skill's default — a hang is a valid test failure.
