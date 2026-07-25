# Truncate package: committed tests

Wrote table-driven tests for `Head` and `Tail` against the documented contract (empty/zero/negative n, exact/over-length n, multi-byte runes, emoji, UTF-8 validity of every result), in `outputs/truncate_test.go`.
Finding for the maintainer: the `Head` doc promises "a negative n is treated as zero" but the code lacks Tail's `n <= 0` guard, so `Head(s, -1)` panics (`r[:-1]`); I deliberately kept the two negative-n tests failing to pin the documented contract rather than silently blessing the panic — fix is a one-line guard (or a doc change, but Tail's symmetry argues for the guard).
Minor: the ask said "./truncate package in the utils module", but the module is `example.invalid/truncate` with the package at the module root; worth reconciling before shipping.
Status: `go build ./...` passes; `go test ./...` fails only on `TestHead/negative_n` and `TestHead/negative_n_empty_string` (the drift above); all other subtests pass.
