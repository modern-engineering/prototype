// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package detection_test

import (
	"testing"

	"github.com/modern-engineering/prototype/application"

	"github.com/modern-engineering/prototype/fieldcase/catalog/detection"
)

// TestCitizenship holds every represented descriptor to the catalogue
// citizenship contract.
func TestCitizenship(t *testing.T) {
	for name, d := range map[string]*application.Descriptor{
		"IdentityAnomalyDetector": detection.IdentityAnomalyDetector,
		"Journal":                 detection.Journal,
		"SyslogExporter":          detection.SyslogExporter,
	} {
		t.Run(name, func(t *testing.T) {
			application.CheckDescriptor(t, d)
		})
	}
}
