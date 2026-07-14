// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package kubernetes

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/modern-engineering/prototype/solution/image"
)

// A Site carries one environment's bindings, as data: rendering is a
// site-binding act, and this is the site's whole surface. The extern
// values are MUST-bind — the render gate refuses a site that leaves
// any reached extern unbound — and the var values MAY-rebind image
// defaults (A-10's two classes at their deployment moment). Where a
// site's bindings live and what document records them stays the open
// door it is (D-13); a Site is deliberately format-free.
type Site struct {
	// Externs are the site's extern bindings, keyed by symbol name.
	Externs map[string]string

	// Vars are the site's var overrides, keyed by symbol name; a var
	// the site does not name keeps its image default.
	Vars map[string]string
}

// A Profile carries the conventions a platform team owns: everything
// about the rendered set that is the environment's business rather
// than the solution's. The zero Profile renders the basic contract —
// per-record shards enacted by a stock prebuilt host — and every
// field widens it toward a house shape without forking the renderer.
type Profile struct {
	// Namespace names the objects' namespace; empty leaves the
	// manifests namespace-free, for the apply to choose.
	Namespace string

	// Labels are extra labels for every object, beside the app and
	// solution pair the renderer always writes.
	Labels map[string]string

	// Payload renders one instance's configuration payload. Nil
	// renders the basic contract: the instance's shard as image.json
	// plus its extern bindings as externs.conf, split by taint.
	Payload func(in Instance) (Payload, error)

	// Container fills the pod's one container. Nil renders the basic
	// contract: a prebuilt-host image token and the arguments that
	// feed it the mounted payload.
	Container func(in Instance) (Container, error)
}

// A Payload is one instance's configuration, split by custody and by
// delivery: files mount into the pod, extra data rides the ConfigMap
// for indirections (an env var fed by a key, say) without mounting.
type Payload struct {
	// Config holds the ConfigMap-mounted files, name to content.
	Config map[string]string

	// Extra holds ConfigMap data entries that are not mounted files.
	Extra map[string]string

	// Secret holds the Secret-mounted files: the sensitive custody,
	// never a ConfigMap and never the environment.
	Secret map[string]string
}

// A Container is the pod's one container, as far as the rendered
// shape reaches today; fields grow as profiles need them.
type Container struct {
	// Image is the container image reference. Rendering never builds
	// or names real images, so this is usually a pipeline token.
	Image string

	// Args are the container arguments.
	Args []string

	// Env are the container's environment variables.
	Env []EnvVar

	// Ports are the named container ports.
	Ports []Port

	// Readiness and Liveness probe the container over HTTP; nil emits
	// no probe. The basic contract emits none: the stock host serves
	// no health surface yet, and a probe against nothing is a lie.
	Readiness, Liveness *HTTPProbe
}

// An EnvVar is one container environment variable: a literal Value,
// or — with ConfigMapKey set — a reference into the instance's own
// ConfigMap.
type EnvVar struct {
	Name         string
	Value        string
	ConfigMapKey string
}

// A Port is one named container port.
type Port struct {
	Name          string
	ContainerPort int
}

// An HTTPProbe is one httpGet probe.
type HTTPProbe struct {
	Path                string
	Port                string // a port name or number
	InitialDelaySeconds int
	PeriodSeconds       int
	FailureThreshold    int
}

// An Instance is one deploy record resolved for rendering: what a
// Profile reads to compose its house shape. Everything here derives
// from the image and the site alone.
type Instance struct {
	// Solution is the solution name, shared by the whole render.
	Solution string

	// Record is the deploy record, verbatim.
	Record image.Record

	// Name is the DNS-1123 object name derived from the instance
	// name.
	Name string

	// Shard is the instance's sub-image: the record, the provisions
	// its parameters reach, the symbols that closure references, over
	// the full pinned catalogue — vars already rebound to the site.
	Shard *image.Image

	// Externs are the instance's reached externs with the site's
	// values, sorted by name, taint carried.
	Externs []ExternBinding

	// Pod and Workload are the record's advisory k8s stanzas, read
	// and defaulted.
	Pod      Pod
	Workload Workload
}

// An ExternBinding is one reached extern resolved against the site.
type ExternBinding struct {
	Name  string
	Value string

	// Sensitive carries the symbol type's taint: a sensitive binding
	// belongs to the Secret custody and must never be logged.
	Sensitive bool
}

// Render turns one solution image and one site into the environment's
// static manifest set: for every deploy record a file named after the
// instance, carrying a ConfigMap, a Secret where anything sensitive
// flows, and a Deployment, in that order. Rendering is dry and pure —
// the image validates against itself, no live catalogue, no cluster,
// never an apply — and deterministic: the same image, site, and
// profile render byte-identical sets, the property a GitOps diff and
// a drift gate both stand on.
//
// The gate: a site key naming no symbol of its class is refused (a
// typo is never a silent no-op), and every extern any rendered
// instance reaches must be bound — the same MUST-bind rule the pod's
// host re-runs, enforced first where artifacts are made. Faults
// report all at once, one line each.
func Render(img *image.Image, site Site, prof Profile) (map[string][]byte, error) {
	if err := img.Validate(); err != nil {
		return nil, err
	}

	var faults []error
	fault := func(format string, args ...any) {
		faults = append(faults, fmt.Errorf("kubernetes: "+format, args...))
	}

	// The site's keys must land: unbound-by-typo is the failure a
	// format-free site surface would otherwise invite.
	classes := make(map[string]string, len(img.Symbols))
	for _, def := range img.Symbols {
		classes[def.Name] = def.Class
	}
	for _, name := range sortedNames(site.Externs) {
		if classes[name] != image.ClassExtern {
			fault("site binds extern %s but the image declares no such extern", name)
		}
	}
	for _, name := range sortedNames(site.Vars) {
		if classes[name] != image.ClassVar {
			fault("site rebinds var %s but the image declares no such var", name)
		}
	}

	// Sensitivity is pinned on the extern's symbol type; index it so
	// bindings split custody without a live catalogue.
	sensitive := make(map[string]bool)
	pinned := make(map[image.Ref]image.ElementSchema)
	for _, pkg := range img.Catalogue {
		for _, es := range pkg.Elements {
			pinned[image.Ref{Package: pkg.Path, Name: es.Name}] = es
		}
	}
	for _, def := range img.Symbols {
		if def.Class == image.ClassExtern && def.Type != nil {
			sensitive[def.Name] = pinned[*def.Type].Sensitive
		}
	}

	names := make(map[string]string) // object name -> instance
	files := make(map[string][]byte)
	for i := range img.Records {
		rec := &img.Records[i]
		if rec.Verb != image.VerbDeploy {
			continue
		}

		name := kebab(rec.Name)
		if other, taken := names[name]; taken {
			fault("instances %s and %s both render as %s", other, rec.Name, name)
			continue
		}
		names[name] = rec.Name
		if len(name)+len("-secret") > 63 {
			fault("instance %s renders as %s, too long for an object name", rec.Name, name)
			continue
		}

		doc, err := renderInstance(img, rec, name, site, prof, sensitive)
		if err != nil {
			fault("instance %s: %v", rec.Name, err)
			continue
		}
		files[name+".yaml"] = doc
	}
	if len(faults) > 0 {
		return nil, errors.Join(faults...)
	}
	return files, nil
}

// renderInstance renders one instance's manifest document set.
func renderInstance(img *image.Image, rec *image.Record, name string, site Site, prof Profile, sensitive map[string]bool) ([]byte, error) {
	needs := closure(img, rec)

	var externs []ExternBinding
	var unbound []string
	for _, sym := range sortedNames(needs.externs) {
		value, ok := site.Externs[sym]
		if !ok {
			unbound = append(unbound, sym)
			continue
		}
		externs = append(externs, ExternBinding{Name: sym, Value: value, Sensitive: sensitive[sym]})
	}
	if len(unbound) > 0 {
		return nil, fmt.Errorf("reaches extern %s; the site must bind it", strings.Join(unbound, ", "))
	}

	shard, err := Shard(img, rec.Name, site.Vars)
	if err != nil {
		return nil, err
	}
	pod, err := podStanza(rec)
	if err != nil {
		return nil, err
	}
	workload, err := workloadStanza(rec)
	if err != nil {
		return nil, err
	}
	in := Instance{
		Solution: img.Solution,
		Record:   *rec,
		Name:     name,
		Shard:    shard,
		Externs:  externs,
		Pod:      pod,
		Workload: workload,
	}

	payload, err := payload(in, prof)
	if err != nil {
		return nil, err
	}
	container, err := container(in, payload, prof)
	if err != nil {
		return nil, err
	}

	data := manifestData{
		Name:          name,
		Namespace:     prof.Namespace,
		Solution:      img.Solution,
		Labels:        objectLabels(in, prof),
		Replicas:      pod.Replicas,
		PriorityClass: pod.PriorityClass,
		CPU:           pod.CPU,
		Memory:        pod.Memory,
		Image:         container.Image,
		Args:          container.Args,
		Env:           container.Env,
		Ports:         container.Ports,
		Readiness:     container.Readiness,
		Liveness:      container.Liveness,
		ConfigMount:   ConfigMount,
		SecretMount:   SecretMount,
	}
	for _, row := range sortedKVs(payload.Config) {
		if row.Value == "" {
			return nil, fmt.Errorf("payload file %s is empty", row.Key)
		}
		if _, collides := payload.Extra[row.Key]; collides {
			return nil, fmt.Errorf("payload names %s as both a file and an extra key", row.Key)
		}
		data.ConfigFiles = append(data.ConfigFiles, kv{Key: row.Key, Value: indent(row.Value, "    ")})
	}
	for _, row := range sortedKVs(payload.Secret) {
		if row.Value == "" {
			return nil, fmt.Errorf("payload secret file %s is empty", row.Key)
		}
		data.SecretFiles = append(data.SecretFiles, kv{Key: row.Key, Value: indent(row.Value, "    ")})
	}
	data.Extra = sortedKVs(payload.Extra)
	for _, b := range rec.Metadata {
		if b.Value != nil {
			data.Meta = append(data.Meta, kv{Key: b.Key, Value: b.Value.Str})
		}
	}

	// The checksum covers the rendered ConfigMap, so a configuration
	// change rolls the Deployment; Secret rotation deliberately does
	// not.
	cm, err := render("configmap", data)
	if err != nil {
		return nil, err
	}
	data.Checksum = checksum(cm)

	docs := [][]byte{cm}
	if len(data.SecretFiles) > 0 {
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

// payload resolves the instance's configuration payload: the
// profile's when one is set, the basic contract otherwise — the shard
// as image.json beside the extern bindings as externs.conf, tainted
// values in the Secret half.
func payload(in Instance, prof Profile) (Payload, error) {
	if prof.Payload != nil {
		return prof.Payload(in)
	}

	var img strings.Builder
	if err := in.Shard.Encode(&img); err != nil {
		return Payload{}, err
	}

	var plain, secret []string
	for _, b := range in.Externs {
		// The extern file is line-based (host.ParseExterns); a value
		// spanning lines would corrupt it silently, so refuse loudly.
		// A document-shaped extern wants the secret-reference custody
		// door, not smuggling.
		if strings.ContainsAny(b.Value, "\r\n") {
			return Payload{}, fmt.Errorf("extern %s: the value spans lines; the extern file carries one binding per line", b.Name)
		}
		line := fmt.Sprintf("%s = %s", b.Name, b.Value)
		if b.Sensitive {
			secret = append(secret, line)
		} else {
			plain = append(plain, line)
		}
	}
	if len(plain) == 0 {
		plain = []string{"# no plain externs reach this instance"}
	}

	p := Payload{Config: map[string]string{
		"image.json":   img.String(),
		"externs.conf": strings.Join(plain, "\n") + "\n",
	}}
	if len(secret) > 0 {
		p.Secret = map[string]string{
			"externs.conf": strings.Join(secret, "\n") + "\n",
		}
	}
	return p, nil
}

// container resolves the pod's container: the profile's when one is
// set, the basic contract otherwise — a prebuilt host handed the
// mounted shard and extern files, its image a token the deployment
// pipeline substitutes.
func container(in Instance, p Payload, prof Profile) (Container, error) {
	if prof.Container != nil {
		return prof.Container(in)
	}
	c := Container{
		Image: "${" + envName(in.Solution) + "_HOST_IMAGE}",
		Args: []string{
			"-image=" + ConfigMount + "/image.json",
			"-externs=" + ConfigMount + "/externs.conf",
		},
	}
	if len(p.Secret) > 0 {
		c.Args = append(c.Args, "-externs="+SecretMount+"/externs.conf")
	}
	return c, nil
}

// objectLabels renders the profile's extra labels plus the part-of
// grouping, sorted; the app and solution pair is the template's own.
func objectLabels(in Instance, prof Profile) []kv {
	labels := make(map[string]string, len(prof.Labels)+1)
	for k, v := range prof.Labels {
		labels[k] = v
	}
	if in.Workload.PartOf != "" {
		labels["app.kubernetes.io/part-of"] = in.Workload.PartOf
	}
	return sortedKVs(labels)
}

// sortedNames lists a string-keyed set in order.
func sortedNames[V any](m map[string]V) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
