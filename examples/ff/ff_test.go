// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package ff_test

import (
	"testing"

	"github.com/modern-engineering/prototype/application"
	"github.com/modern-engineering/prototype/examples/ff"
)

// TestCitizenship holds every exported descriptor to the catalogue
// citizenship contract; a package that fails here has no business being
// imported by a solution.
func TestCitizenship(t *testing.T) {
	for name, d := range map[string]*application.Descriptor{
		"Ping": ff.Ping,
		"Pong": ff.Pong,
	} {
		t.Run(name, func(t *testing.T) {
			application.CheckDescriptor(t, d)
		})
	}
}
