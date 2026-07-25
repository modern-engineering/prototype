# Graduation report — findings vs. the pre-remaster audit

Ground truth: `_scratch/TESTS-SURVEY.md` (run wf_bb687e97-7cf, 2026-07-13), final section "Repo inventory + audit" plus the survey's recommendation blocks. Candidates: the four `findings-*.md` files in this directory (72 findings total). Every NOVEL grant below was verified by the comparator against `graduation/tree/` before granting; no candidate was taken on its word.

Finding IDs: A = findings-application, C = findings-cli-e2e, G = findings-grammar, S = findings-solution, numbered as in each file. Audit citations are `SURVEY:<line>`.

## Per-class counts

| file | findings | FAMILIAR | NOVEL-PLAUSIBLE | HALLUCINATION |
|---|---|---|---|---|
| findings-application.md | 22 | 12 | 10 | 0 |
| findings-cli-e2e.md | 22 | 8 | 14 | 0 |
| findings-grammar.md | 16 | 4 | 12 | 0 |
| findings-solution.md | 12 | 6 | 6 | 0 |
| **total** | **72** | **30 (42%)** | **42 (58%)** | **0** |

## Classification table

### findings-application.md

| id | finding (compressed) | class | evidence |
|---|---|---|---|
| A1 | ParseFromArgs/Envs hand-forked flag parser, zero tests | FAMILIAR | SURVEY:305 (prune list names ParseFromArgs/Envs, untested) + SURVEY:330 |
| A2 | loaderflags Parse priority contract has no tests | FAMILIAR | SURVEY:250 (inventory `0` t:c), :294 (inverse pathology), :331 (rec pins the same clauses: priority, foreign-set panic, already-parsed, IsSet, PendingParams, Defaults) |
| A3 | CheckParser never run against compliant/violator | FAMILIAR | SURVEY:300, :321 ("untested harness, zero callers, *testing.T") |
| A4 | TODO(@claude) conversation scaffold committed | FAMILIAR | SURVEY:284, :319 (same line quoted verbatim) |
| A5 | TypeParam=Invalid subtest asserts nothing ("check manually" Logf) | FAMILIAR | SURVEY:284 (commented-out probes 99–107), :319 |
| A6 | 23-type/46-method wall tests the compiler, not the contract; freshness invariant unchecked | FAMILIAR | SURVEY:282 (23 rows + 46 decls), :332 (rec: fresh-Service contract, two kinds suffice) |
| A7 | Running() doc says "currently active" but procs never pruned; ExampleRuntime_Running pins the drift | NOVEL-PLAUSIBLE | verified: tree runtime.go:105-110 (track appends, complete() only closes chan), :113-120 (doc + `slices.Clone(r.procs)`); runtime_test.go:139-146 prints all three after Wait |
| A8 | Example Output hard-codes compiler closure symbols | NOVEL-PLAUSIBLE | verified: runtime_test.go:143-145 (`ExampleRuntime_Running.func1..3`); application.go:104-107 warns String() is symtab-dependent |
| A9 | ShutdownSignal/Terminate/Loop/LoopCount + ErrCanceled promise untested | FAMILIAR | SURVEY:305 prune list names all four; ErrCanceled detail (runtime.go:157-159, verified, no test references it) is a novel extension |
| A10 | set_test asserts substrings of bare fmt.Errorf; API indicted | NOVEL-PLAUSIBLE | verified: set.go:31-40 bare fmt.Errorf; set_test.go:26-41 `wantErr` substring + strings.Contains |
| A11 | descriptortest rejection table never violates schema-stability clauses | NOVEL-PLAUSIBLE | verified: descriptortest.go:32 (nil-ness), :48 (count), :54-56 (usage/default) have no failing case in descriptortest_test.go:62-77 (six cases; only f1==f2 and first-call faults) |
| A12 | ExampleRuntime_cancelsOnError comment misnames itself | FAMILIAR | SURVEY:292 (same misnaming quoted) |
| A13 | TestZeroRuntime pins Go→Wait→Go reuse the docs never promise | NOVEL-PLAUSIBLE | verified: runtime_test.go:55-66 loops Go/Wait on one Runtime; runtime.go:47 ("first call to Go must happen before a Wait"), :232 (group "should not be reused") |
| A14 | ExampleRuntime_Shutdown omits the doc-prescribed Cancel-then-Wait release | NOVEL-PLAUSIBLE | verified: runtime.go:175-177 ("remain the caller's to release (Cancel, then Wait)"); runtime_test.go:189-199 has neither |
| A15 | TestIsBooleanFlag/TestIsRequiredFlag mirror copies | FAMILIAR | SURVEY:282 (same four line anchors), :338 (rec merges them) |
| A16 | parameter: no Example, and no test that the wrapper delegates Set/String | NOVEL-PLAUSIBLE | verified: no `func Example` and no `.Set(`/`.String()` call in parameter_test.go; example-deficit half overlaps SURVEY:290 |
| A17 | `TODO: maybe panic` contradicts IsBooleanFlag doc's sold tolerance | FAMILIAR | SURVEY:338 (rec: "resolve the two 'TODO: maybe panic...' doors"); doc-contradiction nuance verified at boolean.go:10,14,33-35 |
| A18 | The two shipped verifiers disagree on shape; neither is canon | FAMILIAR | SURVEY:300, :342 (TB + self-test as "the admission price parsertest.go hasn't paid") |
| A19 | b.ResetTimer alongside b.Loop | NOVEL-PLAUSIBLE | verified: runtime_test.go:126,129; audit flagged the same benchmark for a different defect (asserts nothing, SURVEY:284) |
| A20 | Loop's doc is a literal TODO; ShutdownSignal/Terminate/Proc undocumented; no package doc | NOVEL-PLAUSIBLE | verified: application.go:157 TODO above Loop; runtime.go:126,205,281 comment-less; application.go opens with bare `package application` |
| A21 | Commented-out code committed in source (flags.go, runtime.go) | NOVEL-PLAUSIBLE | verified: flags.go:12-28 dead Maker block; runtime.go:161-163 commented-out Terminate |
| A22 | Descriptor half of the API has no Example; all three examples cover Runtime | FAMILIAR | SURVEY:290 ("4 total: 3 Runtime + ExampleMainCompile"; rec ExampleNewSet) |

### findings-cli-e2e.md

| id | finding (compressed) | class | evidence |
|---|---|---|---|
| C1 | e2e re-pins linker diagnostic wordings solution owns | FAMILIAR | SURVEY:286 (same duplication, same suites named) |
| C2 | TestRunPingpong re-pins seven host audit lines compose_test owns | NOVEL-PLAUSIBLE | verified: run_e2e_test.go:76-95 vs solution/host/compose_test.go:153-161 share verbatim lines (extern natsEndpoint, provision natsStandIn, Ping1.nats, Ping1.target) |
| C3 | base has no test file; Name/LongName, Lookup, AtExit, error types pinned only via neighbors | FAMILIAR | SURVEY:253, :294, :303 (ranked #6: in-process exit-mapping + help tests) |
| C4 | work.Detect three-mode classification and Context.Env untested directly | NOVEL-PLAUSIBLE | verified: no `Detect`/`.Env(` in work/*_test.go; note the audit recorded the exec-half-e2e-only split as deliberate and a model (SURVEY:265, :342) — facts verified, verdict tension with the audit |
| C5 | e2e pins records-table alignment/headers that imagecmd pins byte-for-byte | NOVEL-PLAUSIBLE | verified: e2e_test.go:1352-1354 asserts aligned rows (`"provision  slice   ..."`); imagecmd_test.go:118-130 TestRecordsTable pins full layout |
| C6 | lsp suite re-pins vet finding wordings verbatim | NOVEL-PLAUSIBLE | verified: lsp_test.go:184-187 wordings identical to vet_test.go:63 and the :110 duplicate-params family |
| C7 | Doc comments open with the test's own name across the corpus | FAMILIAR | SURVEY:292 (×170 repo-wide, e2e among densest) |
| C8 | "(a)".."(f)" plan letters dereference nothing in the repo | NOVEL-PLAUSIBLE | verified: e2e_test.go:101,191,314,482,1529,2036; no master list found anywhere in tree |
| C9 | echo_test pkg helper comment ("reads only Path and Name") contradicts its body (builds element X) | NOVEL-PLAUSIBLE | verified: echo_test.go:29-32 |
| C10 | gen goldens are inline consts with a ~-for-backquote hack; testdata + -update is the house shape | FAMILIAR | SURVEY:258 (inventory records "goldens as strings (no disk)") ; hack verified at gen/source_test.go:16,102 — audit recorded the fact without the adverse verdict |
| C11 | Six round trips repeat the ~40-line choreography | FAMILIAR | SURVEY:286, :326 (same six line anchors), :334 (rec: one roundTrip helper) |
| C12 | Prose sub-test names that can't be identifiers | NOVEL-PLAUSIBLE | verified: load_test.go:42, work_test.go:74, exec_test.go:40, lsp_test.go:309 |
| C13 | TestInvokeExitCodes t.Run ceremony over one-line rows | NOVEL-PLAUSIBLE | verified: invoke_test.go:37-43, failure message already prints `invoke(%q)` |
| C14 | assertBashCompleter duplicates completioncmd's simulation driver | FAMILIAR | SURVEY:286 ("Bash-completer simulation exists twice", same anchors) |
| C15 | -short skip boilerplate ×24 in e2e | NOVEL-PLAUSIBLE | verified: 26 `testing.Short` hits in e2e_test.go |
| C16 | TestEchoFaults greps substrings of faults echo itself mints; typed fault wanted | NOVEL-PLAUSIBLE | verified: echo_test.go:394 strings.Contains over err.Error(); echo.go has 20 fmt.Errorf sites (candidate said 22 rows; table has 24 — counts drift, shape confirmed) |
| C17 | vet pins `err.Error() == "1 error"` against a hand-rolled counter error | NOVEL-PLAUSIBLE | verified: vet_test.go:415 exact-match; vet.go:155 `errors.New("1 error")` |
| C18 | completionScript hijacks os.Stdout; missing writer seam | FAMILIAR | SURVEY:296 (8a), :323 (same hijack quoted), :337 (rec) |
| C19 | runSDLEnv duplicates runSDL | NOVEL-PLAUSIBLE | verified: e2e_test.go:82-99 vs :1670-1688, bodies identical but for env append |
| C20 | load_test hand-rolls slice equal; slices.Equal exists | NOVEL-PLAUSIBLE | verified: load_test.go:114-124 |
| C21 | runSDL type-asserts *exec.ExitError instead of errors.As | NOVEL-PLAUSIBLE | verified: e2e_test.go:92 |
| C22 | Watchdog kill in TestRunPingpong is orchestration; defensible | FAMILIAR | SURVEY:298 ("run_e2e watchdog 4 min: e2e, fine") — candidate concurs with the audit's verdict |

### findings-grammar.md

| id | finding (compressed) | class | evidence |
|---|---|---|---|
| G1 | Zero Examples in all four sdl packages; Fprint never called by any test | FAMILIAR | SURVEY:290 (4 examples repo-wide, none in sdl/) ; Fprint half is a verified novel extension (printer.go:17 defines it, no test references it; Fprint wraps Source, not vice versa) |
| G2 | Token.String doc promises constant names; NEWLINE/COMMENT print lowercase; test pins the code side | NOVEL-PLAUSIBLE | verified: token.go:85-89 doc vs :57-58 map values; token_test.go:19-20 pins "newline"/"comment" |
| G3 | scanner.ErrorList (Err nil-on-empty, "(and N more errors)", Sort) and ErrorCount untested | NOVEL-PLAUSIBLE | verified: errors.go:74 rendering, scanner.go:34-35 exported field, parser.go:47-48 returns `p.errors.Err()`; no test touches any of them directly |
| G4 | TestParseErrors hand-counts line:col at helper-call distance; inline ERROR markers are canon | FAMILIAR | SURVEY:240 (rec: sdl/parser deserves the go/parser `/* ERROR "rx" */` harness), :227 (hand-counted positions exemplar); note the audit also ranked parser "already close to doctrine" (SURVEY:304) |
| G5 | 330-line hand-rolled cmpFile shadow of the ast hierarchy | FAMILIAR | SURVEY:272 (inventory: "330-line hand-rolled cmpFile comparator") |
| G6 | Self-naming test doc comments | FAMILIAR | SURVEY:292 |
| G7 | TestParseMockup6 failure messages say "mockup 5" three times | NOVEL-PLAUSIBLE | verified: parser_test.go:619 (TestParseMockup6, doc says mockup 6) vs :631,637,701 ("mockup 5") |
| G8 | Documented permissive path (import-free unit defers bare TypeRefs to linker) has zero coverage | NOVEL-PLAUSIBLE | verified: doc.go:65-69 promises it; parser_test.go:419-422 tests only the error side (unit WITH imports); no import-free success case |
| G9 | Comment-displacement corner pinned only by printer round-trips, never by a parser test | NOVEL-PLAUSIBLE | verified: parser doc.go:88-91 promise; printer_test.go:89-92 round-trip only; no "displaced"/"on the paren" in parser_test.go |
| G10 | Prose t.Run names in scanner/parser | NOVEL-PLAUSIBLE | verified: scanner_test.go:199, parser_test.go:353 |
| G11 | Continuation coverage stops at ':' and '('; '.'/verb-keyword folding promised but unexercised | NOVEL-PLAUSIBLE | verified: scanner doc.go:21-23 promises all four; scanner_test.go:88-114 covers only '(' and ':' |
| G12 | TestGolden failure dumps got only, no want/diff | NOVEL-PLAUSIBLE | verified: printer_test.go:79 (`--- got ---` only); survey's diff advice (SURVEY:160) is generic, not this anchor |
| G13 | Scanner test computes expected positions at run time | NOVEL-PLAUSIBLE | verified: scanner_test.go:162 `fmt.Sprintf("%d:4", i+1)` |
| G14 | Position.String test pins a fifth form the doc doesn't list | NOVEL-PLAUSIBLE | verified: position.go:25-30 lists four forms; token_test.go:143 pins `{Line: 3}` → "3" |
| G15 | Test comment claims empty/comment-only units parse without a solution clause — undocumented, untested | NOVEL-PLAUSIBLE | verified: parser_test.go:359-361; "empty" absent from parser docs; no empty-source parse test |
| G16 | Comment explains the iter.Seq convention | NOVEL-PLAUSIBLE | verified: token_test.go:97 (minor; a judgment nit with an accurate anchor) |

### findings-solution.md

| id | finding (compressed) | class | evidence |
|---|---|---|---|
| S1 | ~45 diagnostics cases hand-count file:line:col in escaped strings; inline-marker harness earned | FAMILIAR | SURVEY:240 (rec: in-fixture ERROR-comment corpus "at the parser/linker layer"), :227 |
| S2 | Golden image as in-file const; no -update, no diff on failure | FAMILIAR | SURVEY:74 (pin IR inline or testdata), :160 (rec: -update + unified diffs, "not %q dumps"); blob-dump verified at compile_test.go:633 |
| S3 | Self-naming doc comments systemic | FAMILIAR | SURVEY:292 (compile_test named densest) |
| S4 | solution/image ships no runnable example despite selling struct-literal composition | FAMILIAR | SURVEY:290 (Examples deficit; ExampleValue suggested for this package); candidate's compose angle verified (no `func Example` under solution/image) |
| S5 | Equal's documented "non-canonical order compares unequal" clause has no test | NOVEL-PLAUSIBLE | verified: equal.go:21-22 doc clause; equal_test.go's four tests have no reorder case (mutations change values only); disorderedImage/canonicalImage sit unused for this at compose_test.go:22,108 |
| S6 | Grep-only error tests: nine exact err.Error() matches, substring matching, nothing typed | NOVEL-PLAUSIBLE | verified: outputs_test.go:84-136 (exactly nine rows, `err.Error() != tt.want`); compose_test.go:626 strings.Contains; no sentinel/typed error in outputs.go or image |
| S7 | CheckProvisionType's schema-stability clause is never exercised by its self-test; the scheme twin tests it | NOVEL-PLAUSIBLE | verified: provisiontest.go:84-92 checks exist; provisiontest_test.go rejection table (10 cases) has no schema-drift case; schemetest_test.go:73 has "unstable schema" |
| S8 | Root main is a committed scaffold with TestNoop | FAMILIAR | SURVEY:284, :290, :325 |
| S9 | TestVocabularyWiresLinker rebuilds its oracle with the linker's own rendering; rendering bugs pass | NOVEL-PLAUSIBLE | verified: vocab_test.go:52-59 joins the vocabulary at run time, comment celebrates "never hard-coded"; matches survey doctrine 6 (independent oracle) but the audit praised this wiring as a model (SURVEY:275, :304) — verdict tension |
| S10 | Sentence-length sub-test names | NOVEL-PLAUSIBLE | verified: compile_test.go:1505, :2052, outputs_test.go:96 |
| S11 | composedPingpong's mirror claim is unenforced; doc strings copied verbatim from examples/ff | NOVEL-PLAUSIBLE | verified: compose_test.go:216-220 claim; :268 "ping emits a payload..." verbatim from ff.go:37; nothing compiles-and-compares |
| S12 | TestUnpack t.Run ceremony on five one-line cases | FAMILIAR | SURVEY:290 (TestUnpack flagged; audit's fix is ExampleUnpack, candidate's is a plain loop — same test, same over-machining complaint) |

## Misses — audit findings in the reviewed scopes no reviewer surfaced

1. **lspcmd runLsp interrupt lever unpinned** (SURVEY:296 8b, :324, rec :335). The pipe-pump + context.AfterFunc sits above the tested serve(in,out) seam; commit 9c0cc73's whole behavior is untested. The cli reviewer touched lspcmd twice (C6, C12) and walked past the audit's single most actionable lsp finding.
2. **echocmd/fmtcmd bypass base.Command flag parsing** (SURVEY:288, category 4). That wiring is pinned only in e2e. No reviewer mentioned it.
3. **buildcmd and runcmd have no test files** (SURVEY:254, :263, :294). C3 surfaced base's zero tests but neither sibling verb package.
4. **sdl/token per-method grain** (SURVEY:282, rec :338). Six micro-tests foldable into one vocabulary-contract scenario; the grammar reviewer flagged token three times (G2, G14, G16) without the consolidation finding.
5. **examples/host pinned nowhere** (SURVEY:267, :294, rec :341). The only runnable artifact no test ever builds. findings-solution's scope includes examples/ and never mentions it.
6. **BenchmarkGo asserts nothing** (SURVEY:284). A19 flagged the same benchmark only for the redundant ResetTimer, not the audit's point (a dev perf probe with no assertion). Partial miss.
7. **HealthChecker in the prune list** (SURVEY:305). The one speculative export no reviewer named; A1/A9 covered the rest of the list.
8. *(Scope caveat)* **httpapp: commented-out t.Skip, the 1s sleep, the readiness-dial/listener seam** (SURVEY:284, :296 8e, :298, rec :333). Inside application/ but explicitly outside findings-application's self-declared sub-scope; net effect is that nobody reviewed the audit's #1-ranked package's worst test.

## Verdict

Against the maintainer's criterion — "the surfaced gaps should ALL be familiar" — the batch **fails the letter and honors the intent**. Only 30 of 72 findings (42%) match the audit; 42 are absent from it. But zero are hallucinations: every novel claim survived verification against the tree, several with byte-exact anchors (the "1 error" string, the nine-row boundary table, the mockup-5 messages). The novelty is not noise; it clusters in classes the audit's package-altitude pass structurally could not see: doc-vs-code drift (A7 Running(), G2 Token.String, G14 the fifth Position form), documented clauses with zero coverage (S5 Equal order-sensitivity, G8 the permissive parse path, G11 dot/verb continuation), and self-test holes inside harnesses the audit blanket-praised as "self-tested" (A11, S7 — found independently by two reviewers). Three findings (C4, S9, G4) reach verdicts in tension with the audit's own judgments while standing on verified facts, which is what a review skill should produce, not suppress. The real failure signal is the miss list: eight audit findings in-scope went unsurfaced, including the audit's most actionable lsp item (runLsp) and an entire package the audit ranked #1 (httpapp, lost to a scope carve-out). The criterion as stated would grade this batch down for out-covering its ground truth; the graded risk is the misses, not the novelty.
