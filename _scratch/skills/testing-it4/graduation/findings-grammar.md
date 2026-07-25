# Findings — sdl/token, sdl/scanner, sdl/parser, sdl/printer test suites

Ranked. Reviewer never edits source or prose; drift findings go to the owner.

## Major

1. Not one Example function in any of the four packages (`grep "func Example" sdl/` is empty): the mandatory representative example is missing everywhere — ParseFile-then-Fprint is the whole toolchain's anticipated flow and no pkgsite reader ever sees it; Fprint itself is never even called by a test (printer_test.go exercises Source only).
2. token doc drift silently absorbed: the Token.String doc promises the constant name for non-punctuation, non-keyword tokens ("for IDENT, the string is 'IDENT'"), yet NEWLINE and COMMENT print "newline"/"comment" (token.go:58-59) and token_test.go:19-20 pins the implementation's value — a test authored from the code, siding against the prose; owner must fix doc or code, not the test.
3. scanner.ErrorList is ParseFile's error spine (parser.go:48 returns `p.errors.Err()`), yet nothing tests Err's nil-on-empty (the classic typed-nil trap), Error's "(and N more errors)" rendering, or Sort's cross-position ordering; Scanner.ErrorCount, an exported documented field, is asserted nowhere.
4. TestParseErrors hand-counts 18 line:col pairs at helper-call distance (parser_test.go:345-484) — the go/parser canon puts `/* ERROR "rx" */` markers inline in fixtures so the marker's own position IS the expectation; at this corpus size the inline harness pays for itself and stops the silent position-arithmetic errors.
5. printer_test.go:328-660 hand-rolls a 330-line shadow of the entire ast node hierarchy (cmpFile/cmpDecl/cmpBody/cmpValue) just to compare trees position-insensitively — a design signal: ast should grow a compare or walk facility, or every new node type must be mirrored here or silently pass unchecked.

## Moderate

6. Self-naming test doc comments throughout: "TestKeywords pins..." (token_test.go:75), "TestParseDottedKeys drives..." (parser_test.go:238), "TestParseQualifiedSections drives..." (parser_test.go:274), "TestParseDepthLimit feeds..." (parser_test.go:486), and printer_test.go:56, 129, 143, 181, 229, 297 — no test comment ever opens with its own function's name.
7. TestParseMockup6's failure messages say "mockup 5" three times (parser_test.go:631, 637, 701) while the doc comment and fixtures say mockup 6 — the fixture swap never re-read the test; a failing run would point the reader at the wrong corpus.
8. The parser doc dedicates a bullet to the permissive path — an import-free unit defers unqualified TypeRefs to the linker — and no test parses a bare-name, import-free unit cleanly; every fixture (mockup6_main.sdl:23) carries imports, so the documented deferral has zero coverage.
9. The comment-displacement corner ("if both the '(' line and the ')' line carry a trailing comment...") is promised in the parser's own doc but pinned only by printer round-trips (printer_test.go:90-92); no parser test asserts where the displaced group actually lands (following statement's Before or enclosing After).
10. Prose t.Run names that could never be Go identifiers — "unterminated string at EOF" (scanner_test.go:199), "duplicate parameter key" (parser_test.go:353) — where a plain loop quoting the src in the failure message beats the ceremony, as strings' runIndexTests shows.
11. Scanner continuation coverage stops at ':' and '(': the doc also promises no NEWLINE after '.' or the verb keywords (deploy, import, ...), and neither is exercised (scanner_test.go:88-114) — the exact clauses that let authors break long statements.

## Minor

12. TestGolden's failure dumps the whole got and never shows want or a diff (printer_test.go:79) — the gofmt discipline is a unified diff, and debris beside the fixture beats a wall of bytes.
13. TestScanNumbersAndDurations computes expected positions at run time (`fmt.Sprintf("%d:4", i+1)`, scanner_test.go:162) — that math belongs at coding time, written as results in the rows.
14. token_test.go:143 pins `{Line: 3}` printing "3", a fifth form the Position.String doc does not list — either the doc gains the column-0 form or the case is testing undocumented behavior; owner's call.
15. parser_test.go:360-362 claims in a comment that empty and comment-only units parse without a solution clause — a contract stated nowhere in the docs and tested nowhere; ask the owner, then pin or drop the claim.
16. token_test.go:97 explains the iter.Seq convention ("An early break must stop the iteration, the iter.Seq contract") — nothing good to say about a convention, say nothing.
