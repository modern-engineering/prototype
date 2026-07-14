// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package kubernetes

import (
	"fmt"
	"strconv"

	"github.com/modern-engineering/prototype/solution/image"
)

// The stanza qualifiers this controller recognizes. They are the
// community k8s vocabulary (the conventions catalogue of examples/k8s
// publishes the matching schemes); any package claiming the same
// qualifiers with the same keys satisfies the contract — per D-15 the
// qualifier is the name, never the import. Every other qualifier
// rides the image ignored, the advisory contract.
const (
	podQualifier      = "k8s.pod"
	workloadQualifier = "k8s.workload"
)

// A Pod is the pod-level tuning read off one deploy record's k8s.pod
// stanza, defaults filled where the stanza is silent. A missing
// stanza is never an error — a record tuned for no controller still
// deploys — so the zero stanza and the absent stanza tune alike.
type Pod struct {
	// Replicas is the desired pod count; 1 when unset.
	Replicas int

	// PriorityClass is the scheduling class name; empty means none,
	// and the manifest omits the field.
	PriorityClass string

	// CPU and Memory are the resource pair, in the ecosystem's own
	// quantity grammar, opaque here. Controllers emit requests==limits
	// from the pair, both keys or neither; a stanza binding only one
	// is refused.
	CPU, Memory string
}

// A Workload is the grouping read off one deploy record's
// k8s.workload stanza.
type Workload struct {
	// PartOf names the umbrella workload the instance joins; it lands
	// on the well-known part-of label when set.
	PartOf string
}

// podStanza reads a record's k8s.pod stanza. Values arrive as
// canonical literal text — bare tokens included, their meaning being
// this controller's business exactly as D-12 priced — and only what
// this controller consumes is judged: replicas must read as an int,
// the resource pair must be both-or-neither, unknown keys are the
// linker's business (checked when the scheme's package is imported,
// riding otherwise).
func podStanza(rec *image.Record) (Pod, error) {
	pod := Pod{Replicas: 1}
	for _, b := range rec.Extensions[podQualifier] {
		text, err := valueText(b.Value)
		if err != nil {
			return Pod{}, fmt.Errorf("with %s %s: %v", podQualifier, b.Key, err)
		}
		switch b.Key {
		case "replicas":
			n, err := strconv.Atoi(text)
			if err != nil || n < 0 {
				return Pod{}, fmt.Errorf("with %s replicas: %q is not a replica count", podQualifier, text)
			}
			pod.Replicas = n
		case "priorityClass":
			pod.PriorityClass = text
		case "cpu":
			pod.CPU = text
		case "memory":
			pod.Memory = text
		}
	}
	if (pod.CPU == "") != (pod.Memory == "") {
		return Pod{}, fmt.Errorf("with %s binds only one of cpu and memory; controllers emit requests==limits from both or neither", podQualifier)
	}
	return pod, nil
}

// workloadStanza reads a record's k8s.workload stanza.
func workloadStanza(rec *image.Record) (Workload, error) {
	var w Workload
	for _, b := range rec.Extensions[workloadQualifier] {
		text, err := valueText(b.Value)
		if err != nil {
			return Workload{}, fmt.Errorf("with %s %s: %v", workloadQualifier, b.Key, err)
		}
		if b.Key == "partOf" {
			w.PartOf = text
		}
	}
	return w, nil
}

// valueText renders one literal to the text every consumer surface
// meets: exactly the spelling flag.Value.Set would parse back for the
// typed kinds, and the bare word itself for a token.
func valueText(v *image.Value) (string, error) {
	if v == nil {
		return "", fmt.Errorf("binding carries no literal value")
	}
	switch v.Kind {
	case image.KindString:
		return v.Str, nil
	case image.KindInt:
		return strconv.FormatInt(v.Int, 10), nil
	case image.KindBool:
		return strconv.FormatBool(v.Bool), nil
	case image.KindDuration:
		return v.Dur.String(), nil
	case image.KindToken:
		return v.Tok, nil
	}
	return "", fmt.Errorf("unknown value kind %q", v.Kind)
}
