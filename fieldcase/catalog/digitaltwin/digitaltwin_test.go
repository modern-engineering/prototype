// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package digitaltwin_test

import (
	"testing"

	"github.com/modern-engineering/prototype/application"

	"github.com/modern-engineering/prototype/fieldcase/catalog/digitaltwin"
)

// TestCitizenship holds both represented descriptors to the catalogue
// citizenship contract.
func TestCitizenship(t *testing.T) {
	for name, d := range map[string]*application.Descriptor{
		"TranslatorComponent": digitaltwin.TranslatorComponent,
		"GraphComponent":      digitaltwin.GraphComponent,
	} {
		t.Run(name, func(t *testing.T) {
			application.CheckDescriptor(t, d)
		})
	}
}

// TestInheritedName pins the consumer-group inheritance fact: the
// digital-twin translator's runtime name must remain the digital twin of a target network's
// old component name, or redeployments reprocess history (scenario
// internal compatibility constraint).
func TestInheritedName(t *testing.T) {
	if got, want := digitaltwin.TranslatorComponent.Name, "digitaltwin"; got != want {
		t.Errorf("TranslatorComponent.Name = %q, want %q (consumer-group identity)", got, want)
	}
}
