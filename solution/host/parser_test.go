// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package host

import (
	"testing"

	"github.com/modern-engineering/prototype/application/loader/loaderflags"
)

// The record source feeding every hosted instance's wet binding honors
// loaderflags' priority contract — never Set a flag an earlier source
// claimed — even though it parses alone today; the contract is what
// keeps the door open for sources ahead of the record (an args or
// environment override, say). In-package because the source is an
// implementation detail; the harness call is the whole test.
func TestRecordSourceHonorsPriority(t *testing.T) {
	src := &recordSource{
		values:  map[string]string{"claimed": "from-record"},
		details: map[string]string{"claimed": "literal"},
		applied: map[string]string{},
	}
	loaderflags.CheckParser(t, src, "claimed")
}
