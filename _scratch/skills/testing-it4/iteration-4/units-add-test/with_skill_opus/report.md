Added `ExampleParseDurationOrDefault` to `internal/units/units_test.go`, mirroring its sibling `ExampleParseSizeOrDefault` so both `OrDefault` helpers document the same way and the leniency renders on pkgsite.
The runnable example shows all three documented paths you named: a well-formed `"250ms"` parses, and both `""` and `"garbage"` fall back to the `30s` default with no error, as your doc comment promises.
Your doc comment and implementation agree exactly (empty and unparseable both return `def`), so there was no drift to flag; I reached for a runnable example rather than a table since there was no existing duration test to extend.
Chose an example over a `Test` because its sibling is one and the value is user-facing contract, not a bug hunt.
Final status: `go build ./...` clean, `go test ./...` PASS.
