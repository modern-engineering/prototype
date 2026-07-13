// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package vetcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVetAcceptsCommittedCorpus sweeps the committed solution corpora
// — every example solution and the migrated fieldcase solutions, the
// unit sets the build and e2e suites keep green — and requires vet to
// come back clean: vet must accept whatever sdl build accepts, since
// its findings claim to predict build errors and author mistakes. A
// failure here means either a check drifted stricter than the linker
// (the drift risk the package doc records; fix the check) or the
// corpus grew a genuine finding (fix the corpus); it is never a golden
// to update.
func TestVetAcceptsCommittedCorpus(t *testing.T) {
	for _, root := range []string{"examples", "fieldcase/solutions"} {
		t.Run(root, func(t *testing.T) {
			dir := filepath.Join("..", "..", "..", "..", filepath.FromSlash(root))
			if _, err := os.Stat(dir); err != nil {
				t.Fatalf("corpus root missing (did the repo layout move?): %v", err)
			}
			var buf strings.Builder
			v := &vetter{stderr: &buf}
			if err := v.run([]string{dir}); err != nil || buf.Len() > 0 {
				t.Errorf("sdl vet %s = %v, want clean\n%s", root, err, buf.String())
			}
			if len(v.visited) == 0 {
				t.Errorf("the %s walk found no units; a clean verdict over nothing proves nothing", root)
			}
		})
	}
}
