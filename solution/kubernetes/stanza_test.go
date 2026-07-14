// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package kubernetes_test

import (
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/solution/image"
	"github.com/modern-engineering/prototype/solution/kubernetes"
)

// stanzaRecord wraps one stanza into a bare record, the only shape
// the readers consume.
func stanzaRecord(qualifier string, bindings ...image.Binding) *image.Record {
	return &image.Record{Extensions: map[string][]image.Binding{qualifier: bindings}}
}

// bind is one stanza binding with a literal payload.
func bind(key string, v *image.Value) image.Binding {
	return image.Binding{Key: key, Value: v, Source: image.SourceInstance}
}

// TestPodStanza pins the k8s.pod contract this controller honors: a
// silent stanza tunes exactly like an absent one (one replica, no
// class, no resources), bound keys land typed — replicas as a count,
// the rest as canonical text, tokens welcome — and the resource pair
// is both-or-neither, refused when split.
func TestPodStanza(t *testing.T) {
	pod, err := kubernetes.PodStanza(&image.Record{})
	if err != nil || pod.Replicas != 1 || pod.CPU != "" || pod.PriorityClass != "" {
		t.Fatalf("absent stanza = %+v, %v; want one replica and nothing else", pod, err)
	}

	pod, err = kubernetes.PodStanza(stanzaRecord("k8s.pod",
		bind("cpu", image.String("300m")),
		bind("memory", image.String("250Mi")),
		bind("priorityClass", image.Token("standard")),
		bind("replicas", image.Int(3)),
	))
	if err != nil {
		t.Fatalf("PodStanza: %v", err)
	}
	if pod.Replicas != 3 || pod.CPU != "300m" || pod.Memory != "250Mi" || pod.PriorityClass != "standard" {
		t.Fatalf("PodStanza = %+v, want the bound tuning", pod)
	}

	if _, err := kubernetes.PodStanza(stanzaRecord("k8s.pod", bind("cpu", image.String("300m")))); err == nil {
		t.Fatal("a cpu without its memory passed; want the both-or-neither refusal")
	}
	if _, err := kubernetes.PodStanza(stanzaRecord("k8s.pod", bind("replicas", image.String("many")))); err == nil {
		t.Fatal("a non-count replicas passed; want a refusal")
	} else if !strings.Contains(err.Error(), "replicas") {
		t.Fatalf("refusal %q does not name the key", err)
	}
}

// TestPodStanzaIgnoresForeignKeys pins the advisory posture at the
// key level: a key this controller does not consume rides ignored —
// whether it is checked at all belongs to the linker and the imported
// scheme, never to the reader.
func TestPodStanzaIgnoresForeignKeys(t *testing.T) {
	pod, err := kubernetes.PodStanza(stanzaRecord("k8s.pod",
		bind("replicas", image.Int(2)),
		bind("futureKnob", image.Token("on")),
	))
	if err != nil {
		t.Fatalf("PodStanza: %v", err)
	}
	if pod.Replicas != 2 {
		t.Fatalf("PodStanza = %+v, want 2 replicas beside the ignored key", pod)
	}
}

// TestWorkloadStanza pins the grouping read: partOf lands verbatim,
// silence means no umbrella.
func TestWorkloadStanza(t *testing.T) {
	w, err := kubernetes.WorkloadStanza(stanzaRecord("k8s.workload", bind("partOf", image.String("billing"))))
	if err != nil || w.PartOf != "billing" {
		t.Fatalf("WorkloadStanza = %+v, %v; want billing", w, err)
	}
	w, err = kubernetes.WorkloadStanza(&image.Record{})
	if err != nil || w.PartOf != "" {
		t.Fatalf("absent stanza = %+v, %v; want the zero grouping", w, err)
	}
}
