// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package main

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"text/template"

	"github.com/modern-engineering/prototype/fieldcase/host"
	"github.com/modern-engineering/prototype/fieldcase/host/site"
)

// The in-pod mount points of the split site fragments.
const (
	configMount = "/etc/solution/config"
	secretMount = "/etc/solution/secret"
)

// generateK8s emits one static YAML per deploy record — ConfigMap,
// optional Secret, Deployment — using a conventional one-component-per-pod profile: one component per pod, the same
// host image everywhere, the instance selected by its enable env var,
// configuration delivered as a ConfigMap-mounted file whose path is
// the container argument, sensitive values split into a Secret, a
// checksum annotation forcing rollout on config change, and probes
// against the host's health endpoint.
//
// The site's values are baked here (unlike host mode): each instance
// receives exactly the fragment its bindings reach — its externs, its
// var overrides, and the driver outputs its provisions feed it —
// split by sensitivity. Records with output references need no init
// containers: the pod's host process resolves outputs at startup
// through the same driver hook the single-process host uses.
func generateK8s(m *model, st *site.Site) (map[string][]byte, error) {
	files := make(map[string][]byte, len(m.cfg.Instances))
	for _, inst := range m.cfg.Instances {
		doc, err := renderInstance(m, st, inst)
		if err != nil {
			return nil, fmt.Errorf("instance %s: %v", inst.Name, err)
		}
		files[kebab(inst.Name)+".yaml"] = doc
	}
	return files, nil
}

// instanceData feeds the manifest templates for one instance.
type instanceData struct {
	Instance   string // instance name: the enable ConfigMap key
	Name       string // kebab-case Kubernetes object name
	Solution   string
	EnvName    string // enable env var (UPPER(instance))
	Image      string // image placeholder token
	Health     string // health listen address baked into the ConfigMap
	HealthPort string
	SiteConf   string // non-sensitive site fragment, pre-indented
	SecretConf string // sensitive site fragment, pre-indented; "" for none
	Checksum   string
	CPU        string
	Memory     string
	ConfigPath string
	SecretPath string
	Metadata   []kv // the record's metadata compartment, passed through
}

type kv struct{ Key, Value string }

// renderInstance renders one instance's manifest document set.
func renderInstance(m *model, st *site.Site, inst host.Instance) ([]byte, error) {
	needs := m.cfg.Needs([]string{inst.Name})

	// The per-instance site fragments: everything this instance's
	// closure consumes, split by sensitivity. Vars ride along only
	// when the site overrides them (image defaults are baked in the
	// host binary already).
	var conf, secret []string
	for _, sym := range m.cfg.Symbols {
		switch sym.Class {
		case host.ClassExtern:
			if !needs.Externs[sym.Name] {
				continue
			}
			v, _ := st.Extern(sym.Name) // generation gate guaranteed presence
			line := fmt.Sprintf("extern.%s = %s", sym.Name, v)
			if sym.Sensitive {
				secret = append(secret, line)
			} else {
				conf = append(conf, line)
			}
		case host.ClassVar:
			if !needs.Vars[sym.Name] {
				continue
			}
			if v, ok := st.Var(sym.Name); ok {
				conf = append(conf, fmt.Sprintf("var.%s = %s", sym.Name, v))
			}
		}
	}
	for _, p := range m.cfg.Provisions {
		if !needs.Provisions[p.Name] {
			continue
		}
		sensitive := make(map[string]bool, len(p.Outputs))
		for _, out := range p.Outputs {
			sensitive[out.Name] = out.Sensitive
		}
		section := st.Driver(p.Name)
		for _, out := range sortedNames(needs.Outputs[p.Name]) {
			line := fmt.Sprintf("driver.%s.%s = %s", p.Name, out, section[out])
			if sensitive[out] {
				secret = append(secret, line)
			} else {
				conf = append(conf, line)
			}
		}
	}
	if len(conf) == 0 {
		conf = []string{"# nothing site-bound reaches this instance"}
	}

	rec := m.deploys[inst.Name]
	data := instanceData{
		Instance:   inst.Name,
		Name:       kebab(inst.Name),
		Solution:   m.cfg.Solution,
		EnvName:    host.EnvName(inst.Name),
		Image:      "${" + host.EnvName(m.cfg.Solution) + "_HOST_IMAGE}",
		Health:     ":3000",
		HealthPort: "3000",
		SiteConf:   indent(conf, "    "),
		ConfigPath: configMount,
		SecretPath: secretMount,
	}
	if len(secret) > 0 {
		data.SecretConf = indent(secret, "    ")
	}
	for _, b := range rec.On {
		if b.Value == nil {
			continue
		}
		v, err := canonical(b.Value)
		if err != nil {
			return nil, fmt.Errorf("on %s: %v", b.Key, err)
		}
		switch b.Key {
		case "cpu":
			data.CPU = v
		case "memory":
			data.Memory = v
		}
	}
	if (data.CPU == "") != (data.Memory == "") {
		return nil, fmt.Errorf("on compartment binds only one of cpu and memory; the generator emits requests==limits for both or neither")
	}
	for _, b := range rec.Metadata {
		if b.Value != nil {
			data.Metadata = append(data.Metadata, kv{Key: b.Key, Value: b.Value.Str})
		}
	}

	// The checksum covers the rendered ConfigMap, so a config change
	// rolls the Deployment (the base-service chart's pattern). Secret
	// rotation intentionally does not roll pods here, as in the field case
	// charts.
	cm, err := render("configmap", data)
	if err != nil {
		return nil, err
	}
	data.Checksum = fmt.Sprintf("%x", sha256.Sum256(cm))

	docs := [][]byte{cm}
	if data.SecretConf != "" {
		sec, err := render("secret", data)
		if err != nil {
			return nil, err
		}
		docs = append(docs, sec)
	}
	dep, err := render("deployment", data)
	if err != nil {
		return nil, err
	}
	docs = append(docs, dep)
	return joinDocs(docs), nil
}

// render executes one named manifest template.
func render(name string, data instanceData) ([]byte, error) {
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

// indent prefixes every line, producing the body of a YAML block
// scalar.
func indent(lines []string, prefix string) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString(prefix)
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// kebab converts an identifier instance name to a DNS-1123 object
// name: a '-' lands before every upper-case letter that starts a word
// (identityAnomalyDetector -> identity-anomaly-detector, networkObservationCollector ->
// network-observation-collector). This is F1's other half: where the
// result differs in the deployment manifest name, the divergence is exactly
// F2's two-name problem, and the record's metadata carries the chart
// spelling.
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

// manifests holds the three object templates. Field values that came
// from identifiers or canonical literals are YAML-safe; free-form
// strings (metadata values) are quoted.
var manifests = template.Must(template.New("k8s").Parse(`
{{- define "configmap" -}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{.Name}}-cm
  labels:
    app: {{.Name}}
    solution: {{.Solution}}
data:
  # The enable key: the pod's env selects exactly this instance out of
  # the shared host image (tri-state enable semantics).
  {{.Instance}}: "true"
  health: "{{.Health}}"
  # The site fragment this instance's bindings reach; the host reads
  # it at startup (args[0]) and enforces the extern gate again.
  site.conf: |
{{.SiteConf}}
{{end}}

{{- define "secret" -}}
apiVersion: v1
kind: Secret
metadata:
  name: {{.Name}}-secret
  labels:
    app: {{.Name}}
    solution: {{.Solution}}
type: Opaque
stringData:
  # The sensitive half of the site: extern Secrets and tainted driver
  # outputs, mounted as a second site file (path-in-args), never in a
  # ConfigMap and never in env.
  secret.conf: |
{{.SecretConf}}
{{end}}

{{- define "deployment" -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{.Name}}
  labels:
    app: {{.Name}}
    solution: {{.Solution}}
{{- if .Metadata}}
  annotations:
{{- range .Metadata}}
    solution.metadata/{{.Key}}: {{printf "%q" .Value}}
{{- end}}
{{- end}}
spec:
  replicas: 1
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
      containers:
        - name: {{.Name}}
          # One host image serves every instance of the solution; the
          # deployment pipeline substitutes the token.
          image: {{.Image}}
          args:
            - {{.ConfigPath}}/site.conf
{{- if .SecretConf}}
            - {{.SecretPath}}/secret.conf
{{- end}}
          env:
            - name: {{.EnvName}}
              valueFrom:
                configMapKeyRef:
                  name: {{.Name}}-cm
                  key: {{.Instance}}
            - name: HEALTH
              valueFrom:
                configMapKeyRef:
                  name: {{.Name}}-cm
                  key: health
          ports:
            - name: health
              containerPort: {{.HealthPort}}
          readinessProbe:
            httpGet:
              path: /readyz
              port: health
            periodSeconds: 5
            failureThreshold: 3
          livenessProbe:
            httpGet:
              path: /healthz
              port: health
            initialDelaySeconds: 10
            periodSeconds: 6
            failureThreshold: 10
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
            - name: site-config
              mountPath: {{.ConfigPath}}
              readOnly: true
{{- if .SecretConf}}
            - name: site-secret
              mountPath: {{.SecretPath}}
              readOnly: true
{{- end}}
      volumes:
        - name: site-config
          configMap:
            name: {{.Name}}-cm
            items:
              - key: site.conf
                path: site.conf
{{- if .SecretConf}}
        - name: site-secret
          secret:
            secretName: {{.Name}}-secret
{{- end}}
{{end}}
`))
