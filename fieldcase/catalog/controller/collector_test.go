// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package controller_test

import (
	"testing"

	"github.com/modern-engineering/prototype/application"

	"github.com/modern-engineering/prototype/fieldcase/catalog/controller"
)

// TestCitizenship holds the represented descriptor to the catalogue
// citizenship contract.
func TestCitizenship(t *testing.T) {
	application.CheckDescriptor(t, controller.Collector)
}
