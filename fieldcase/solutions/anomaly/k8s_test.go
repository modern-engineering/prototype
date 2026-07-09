// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package anomaly

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The expected descriptor files are keyed by whether the instance carries
// sensitive bindings and therefore requires a Secret document.
var k8sFiles = map[string]bool{
	"network-observation-collector.yaml": true,  // controller credentials + token-cache credentials
	"observation-translator.yaml":      false, // topics and broker only
	"digitaltwin.yaml":                false, // topics and broker only
	"digital-twin-graph.yaml":              true,  // twinDB credentials
	"identity-anomaly-detector.yaml":          true,  // identityState credentials
	"detection-alerts-journal.yaml":   true,  // alertsDB credentials
	"detection-syslog-exporter.yaml":  false, // SIEM endpoint is not sensitive
}

// TestK8sDescriptors asserts the structural invariants of the
// committed manifests with plain string and regexp checks — the
// stdlib-only, honest substitute for a YAML schema load. kubectl
// or a cluster would be the field case parse; nothing here needs one.
func TestK8sDescriptors(t *testing.T) {
	dir := filepath.Join("gen", "k8s")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(k8sFiles) {
		t.Errorf("gen/k8s holds %d files, want %d (one per deploy record)", len(entries), len(k8sFiles))
	}

	checksum := regexp.MustCompile(`checksum/config: [0-9a-f]{64}`)
	for name, wantSecret := range k8sFiles {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			app := strings.TrimSuffix(name, ".yaml")

			if strings.Contains(text, "\t") {
				t.Error("YAML must not contain tabs")
			}
			docs := strings.Split(text, "\n---\n")
			wantDocs := 2
			if wantSecret {
				wantDocs = 3
			}
			if len(docs) != wantDocs {
				t.Fatalf("%d documents, want %d (ConfigMap%s, Deployment)",
					len(docs), wantDocs, map[bool]string{true: ", Secret", false: ""}[wantSecret])
			}

			cm := docs[0]
			dep := docs[len(docs)-1]
			if !strings.Contains(cm, "kind: ConfigMap") || !strings.Contains(cm, "name: "+app+"-cm") {
				t.Errorf("first document is not the %s-cm ConfigMap", app)
			}
			if !strings.Contains(cm, "site.conf: |") {
				t.Error("ConfigMap does not embed the site fragment")
			}
			if !strings.Contains(dep, "kind: Deployment") || !strings.Contains(dep, "name: "+app) {
				t.Errorf("last document is not the %s Deployment", app)
			}
			if wantSecret {
				sec := docs[1]
				if !strings.Contains(sec, "kind: Secret") || !strings.Contains(sec, "name: "+app+"-secret") {
					t.Errorf("middle document is not the %s-secret Secret", app)
				}
				if !strings.Contains(sec, "secret.conf: |") {
					t.Error("Secret does not embed the sensitive site fragment")
				}
				if !strings.Contains(dep, "- /etc/solution/secret/secret.conf") {
					t.Error("Deployment args do not pass the secret site path")
				}
			} else if strings.Contains(text, "kind: Secret") {
				t.Error("instance without sensitive bindings must not get a Secret")
			}

			// The Deployment invariants of the digest §4(b) checklist.
			for _, want := range []string{
				"replicas: 1",
				"image: ${ANOMALY_HOST_IMAGE}",
				"- /etc/solution/config/site.conf",
				"path: /readyz",
				"path: /healthz",
				"port: health",
				"configMapKeyRef:",
				"key: health",
				"mountPath: /etc/solution/config",
				"readOnly: true",
				"solution: anomaly",
			} {
				if !strings.Contains(dep, want) {
					t.Errorf("Deployment is missing %q", want)
				}
			}
			if !checksum.MatchString(dep) {
				t.Error("Deployment has no checksum/config rollout annotation")
			}

			// requests==limits: the resources block repeats the pair.
			cpus := regexp.MustCompile(`cpu: "[^"]*"`).FindAllString(dep, -1)
			mems := regexp.MustCompile(`memory: "[^"]*"`).FindAllString(dep, -1)
			if len(cpus) != 2 || cpus[0] != cpus[1] || len(mems) != 2 || mems[0] != mems[1] {
				t.Errorf("resources are not requests==limits: cpu %v, memory %v", cpus, mems)
			}

			// Selector and pod labels agree.
			if !strings.Contains(dep, "matchLabels:\n      app: "+app) ||
				!strings.Contains(dep, "labels:\n        app: "+app) {
				t.Error("selector and pod template labels do not both pin app=" + app)
			}

			// No sensitive value may land outside the Secret document.
			for _, secret := range devSecrets {
				if strings.Contains(cm, secret) || strings.Contains(dep, secret) {
					t.Errorf("sensitive value %q escaped the Secret document", secret)
				}
			}
		})
	}
}

// TestK8sEnableEnv pins the per-pod instance selection: each
// Deployment sets exactly its own enable env var (F1's uppercase
// mapping) wired to the ConfigMap's enable key.
func TestK8sEnableEnv(t *testing.T) {
	pairs := map[string][2]string{
		"identity-anomaly-detector.yaml":          {"IDENTITYANOMALYDETECTOR", "identityAnomalyDetector"},
		"digitaltwin.yaml":                {"DIGITALTWIN", "digitaltwin"},
		"network-observation-collector.yaml": {"NETWORKOBSERVATIONCOLLECTOR", "networkObservationCollector"},
	}
	for name, pair := range pairs {
		data, err := os.ReadFile(filepath.Join("gen", "k8s", name))
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		if !strings.Contains(text, "- name: "+pair[0]) {
			t.Errorf("%s: missing enable env %s", name, pair[0])
		}
		if !strings.Contains(text, "key: "+pair[1]) {
			t.Errorf("%s: enable env not wired to ConfigMap key %s", name, pair[1])
		}
		if !strings.Contains(text, pair[1]+": \"true\"") {
			t.Errorf("%s: ConfigMap does not carry the enable key", name)
		}
	}
}
