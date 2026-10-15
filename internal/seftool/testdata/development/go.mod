module example.test/local/v2

go 1.27.0

require github.com/modern-engineering/prototype v0.0.0

// This caller-only override must not leak into a generated host.
replace example.test/unrelated => ./missing
