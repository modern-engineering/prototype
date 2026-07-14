// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package kubernetes

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

// The in-pod mount points of an instance's configuration, split by
// custody: what a ConfigMap may carry and what only a Secret may.
// Profiles compose container arguments against these paths.
const (
	// ConfigMount is where the pod mounts the instance's plain
	// configuration files.
	ConfigMount = "/etc/solution/config"

	// SecretMount is where the pod mounts the instance's sensitive
	// files, fed from a Secret and never from a ConfigMap.
	SecretMount = "/etc/solution/secret"
)

// A kv is one rendered key-value row of a manifest map.
type kv struct{ Key, Value string }

// manifestData feeds the object templates for one instance.
type manifestData struct {
	Name      string // DNS-1123 object name
	Namespace string
	Solution  string
	Labels    []kv // extra object labels beyond app and solution
	Meta      []kv // the record's metadata compartment, passed through

	Extra       []kv // ConfigMap data keys that are not mounted files
	ConfigFiles []kv // ConfigMap-mounted files, content pre-indented
	SecretFiles []kv // Secret-mounted files, content pre-indented

	Checksum      string
	Replicas      int
	PriorityClass string
	CPU, Memory   string

	Image string
	Args  []string
	Env   []EnvVar
	Ports []Port

	Readiness, Liveness *HTTPProbe

	ConfigMount, SecretMount string
}

// render executes one named manifest template.
func render(name string, data manifestData) ([]byte, error) {
	var b strings.Builder
	if err := manifests.ExecuteTemplate(&b, name, data); err != nil {
		return nil, err
	}
	return []byte(b.String()), nil
}

// joinDocs concatenates YAML documents with the --- separator.
func joinDocs(docs [][]byte) []byte {
	var b []byte
	for i, doc := range docs {
		if i > 0 {
			b = append(b, []byte("---\n")...)
		}
		b = append(b, doc...)
	}
	return b
}

// checksum digests one rendered document; the pod-template annotation
// carries it so a configuration change rolls the Deployment while an
// unchanged render leaves the workload untouched.
func checksum(doc []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(doc))
}

// indent prefixes every line of a file's content, producing the body
// of a YAML block scalar.
func indent(content, prefix string) string {
	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	var b strings.Builder
	for i, line := range lines {
		if i > 0 {
			b.WriteString("\n")
		}
		if line != "" {
			b.WriteString(prefix)
			b.WriteString(line)
		}
	}
	return b.String()
}

// sortedKVs renders a map as rows sorted by key, the only order a
// deterministic manifest may use.
func sortedKVs(m map[string]string) []kv {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	rows := make([]kv, 0, len(keys))
	for _, k := range keys {
		rows = append(rows, kv{Key: k, Value: m[k]})
	}
	return rows
}

// kebab converts an identifier instance name to a DNS-1123 object
// name: a '-' lands before every upper-case letter that starts a
// word, and the letter lowers (identityAnomalyDetector -> identity-anomaly-detector,
// networkObservationCollector -> network-observation-collector). Where the
// result differs from a name the environment already knows the thing
// by, the record's metadata carries the foreign spelling; the
// instance name stays the identity.
func kebab(name string) string {
	runes := []rune(name)
	var b strings.Builder
	for i, r := range runes {
		if r >= 'A' && r <= 'Z' {
			prevLower := i > 0 && isLower(runes[i-1])
			nextLower := i+1 < len(runes) && isLower(runes[i+1])
			if prevLower || (i > 0 && nextLower) {
				b.WriteByte('-')
			}
			b.WriteRune(r - 'A' + 'a')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isLower(r rune) bool { return r >= 'a' && r <= 'z' }

// envName maps a name to the environment-variable register:
// ASCII-uppercase, separators to underscores.
func envName(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r - 'a' + 'A'
		case r == '-' || r == '.' || r == '/':
			return '_'
		}
		return r
	}, name)
}

// manifests holds the three object templates. Names and file keys
// come from identifiers or canonical literals and are YAML-safe;
// free-form strings (labels, annotations, extra data values) are
// quoted.
var manifests = template.Must(template.New("kubernetes").Parse(`
{{- define "labels"}}
    app: {{.Name}}
    solution: {{.Solution}}
{{- range .Labels}}
    {{.Key}}: {{printf "%q" .Value}}
{{- end}}
{{- end}}

{{- define "configmap" -}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.Name}}-cm
{{- with .Namespace}}
  namespace: {{.}}
{{- end}}
  labels:
{{- template "labels" .}}
data:
{{- range .Extra}}
  {{.Key}}: {{printf "%q" .Value}}
{{- end}}
{{- range .ConfigFiles}}
  {{.Key}}: |
{{.Value}}
{{- end}}
{{end}}

{{- define "secret" -}}
apiVersion: v1
kind: Secret
metadata:
  name: {{.Name}}-secret
{{- with .Namespace}}
  namespace: {{.}}
{{- end}}
  labels:
{{- template "labels" .}}
type: Opaque
# The sensitive custody: tainted values mount from here, never from a
# ConfigMap and never through the environment.
stringData:
{{- range .SecretFiles}}
  {{.Key}}: |
{{.Value}}
{{- end}}
{{end}}

{{- define "deployment" -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Name}}
{{- with .Namespace}}
  namespace: {{.}}
{{- end}}
  labels:
{{- template "labels" .}}
{{- if .Meta}}
  annotations:
{{- range .Meta}}
    solution.metadata/{{.Key}}: {{printf "%q" .Value}}
{{- end}}
{{- end}}
spec:
  replicas: {{.Replicas}}
  selector:
    matchLabels:
      app: {{.Name}}
  template:
    metadata:
      labels:
        app: {{.Name}}
        solution: {{.Solution}}
      annotations:
        checksum/config: {{.Checksum}}
    spec:
{{- with .PriorityClass}}
      priorityClassName: {{.}}
{{- end}}
      containers:
        - name: {{.Name}}
          image: {{.Image}}
{{- if .Args}}
          args:
{{- range .Args}}
            - {{.}}
{{- end}}
{{- end}}
{{- if .Env}}
          env:
{{- range .Env}}
            - name: {{.Name}}
{{- if .ConfigMapKey}}
              valueFrom:
                configMapKeyRef:
                  name: {{$.Name}}-cm
                  key: {{.ConfigMapKey}}
{{- else}}
              value: {{printf "%q" .Value}}
{{- end}}
{{- end}}
{{- end}}
{{- if .Ports}}
          ports:
{{- range .Ports}}
            - name: {{.Name}}
              containerPort: {{.ContainerPort}}
{{- end}}
{{- end}}
{{- with .Readiness}}
          readinessProbe:
            httpGet:
              path: {{.Path}}
              port: {{.Port}}
{{- if .InitialDelaySeconds}}
            initialDelaySeconds: {{.InitialDelaySeconds}}
{{- end}}
            periodSeconds: {{.PeriodSeconds}}
            failureThreshold: {{.FailureThreshold}}
{{- end}}
{{- with .Liveness}}
          livenessProbe:
            httpGet:
              path: {{.Path}}
              port: {{.Port}}
{{- if .InitialDelaySeconds}}
            initialDelaySeconds: {{.InitialDelaySeconds}}
{{- end}}
            periodSeconds: {{.PeriodSeconds}}
            failureThreshold: {{.FailureThreshold}}
{{- end}}
{{- if .CPU}}
          resources:
            requests:
              cpu: "{{.CPU}}"
              memory: "{{.Memory}}"
            limits:
              cpu: "{{.CPU}}"
              memory: "{{.Memory}}"
{{- end}}
          volumeMounts:
            - name: solution-config
              mountPath: {{.ConfigMount}}
              readOnly: true
{{- if .SecretFiles}}
            - name: solution-secret
              mountPath: {{.SecretMount}}
              readOnly: true
{{- end}}
      volumes:
        - name: solution-config
          configMap:
            name: {{.Name}}-cm
            items:
{{- range .ConfigFiles}}
              - key: {{.Key}}
                path: {{.Key}}
{{- end}}
{{- if .SecretFiles}}
        - name: solution-secret
          secret:
            secretName: {{.Name}}-secret
{{- end}}
{{end}}
`))
