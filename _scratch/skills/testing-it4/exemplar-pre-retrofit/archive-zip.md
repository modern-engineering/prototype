# archive/zip — buying a whole interface contract with one verifier call

Source: `archive/zip/reader_test.go` (Go development tree, 1.27 dev, 2026-06).
Verbatim; BSD-3-Clause, © The Go Authors.

Surface exercised — `testing/fstest/testfs.go` lines 20-39:

```go
// TestFS tests a file system implementation.
// It walks the entire tree of files in fsys,
// opening and checking that each file behaves correctly.
// Symbolic links are not followed,
// but their Lstat values are checked
// if the file system implements [fs.ReadLinkFS].
// It also checks that the file system contains at least the expected files.
// As a special case, if no expected files are listed, fsys must be empty.
// Otherwise, fsys must contain at least the listed files; it can also contain others.
// The contents of fsys must not change concurrently with TestFS.
//
// If TestFS finds any misbehaviors, it returns either the first error or a
// list of errors. Use [errors.Is] or [errors.AsType] to inspect.
//
// Typical usage inside a test is:
//
//	if err := fstest.TestFS(myFS, "file/that/should/be/present"); err != nil {
//		t.Fatal(err)
//	}
func TestFS(fsys fs.FS, expected ...string) error {
```

reader_test.go lines 1203-1229:

```go
func TestFS(t *testing.T) {
	for _, test := range []struct {
		file string
		want []string
	}{
		{
			"testdata/unix.zip",
			[]string{"hello", "dir/bar", "readonly"},
		},
		{
			"testdata/subdir.zip",
			[]string{"a/b/c"},
		},
	} {
		t.Run(test.file, func(t *testing.T) {
			t.Parallel()
			z, err := OpenReader(test.file)
			if err != nil {
				t.Fatal(err)
			}
			defer z.Close()
			if err := fstest.TestFS(z, test.want...); err != nil {
				t.Error(err)
			}
		})
	}
}
```

zip implements `fs.FS`, so one `fstest.TestFS` call per archive buys the ENTIRE
documented file-system contract — every Open, ReadDir, Stat clause, checked over
the whole tree. The verifier shape matters: it takes no `*testing.T`, returns
joined errors, and the caller decides severity. When you own an interface, ship
a verifier in this shape; when you implement one, call the owner's verifier
before writing any bespoke test. The bespoke tests that follow in this file
cover only what `TestFS` cannot know: zip-specific contents and errors.
