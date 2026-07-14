// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"

	"github.com/modern-engineering/prototype/fieldcase/host"
	"github.com/modern-engineering/prototype/fieldcase/host/site"
	"github.com/modern-engineering/prototype/solution/kubernetes"
)

// generateK8s emits a ConfigMap, optional Secret, and Deployment for each
// deployed instance through the shared Kubernetes renderer. A local Profile
// supplies the scenario's naming, payload, container, and probe conventions.
// Public and sensitive site fragments are split before mounting, and the
// compile-checked k8s.pod stanza supplies resource requests and limits.
//
// The site's values are baked here (unlike host mode): each instance
// receives exactly the fragment its bindings reach — its externs, its
// var overrides, and the driver outputs its provisions feed it.
// Records with output references need no init containers: the pod's
// host process resolves outputs at startup through the same driver
// hook the single-process host uses.
func generateK8s(m *model, st *site.Site) (map[string][]byte, error) {
	ksite := kubernetes.Site{
		Externs: make(map[string]string),
		Vars:    make(map[string]string),
	}
	for _, name := range st.Externs() {
		ksite.Externs[name], _ = st.Extern(name)
	}
	for _, name := range st.Vars() {
		ksite.Vars[name], _ = st.Var(name)
	}

	prof := kubernetes.Profile{
		Payload: func(in kubernetes.Instance) (kubernetes.Payload, error) {
			conf, secret := siteFragments(m, st, in.Record.Name)
			p := kubernetes.Payload{
				Config: map[string]string{"site.conf": conf},
				// The ConfigMap indirections the Deployment's env reads:
				// the enable key selecting exactly this instance out of
				// the shared host image (tri-state enable semantics) and
				// the health listen address.
				Extra: map[string]string{
					in.Record.Name: "true",
					"health":       ":3000",
				},
			}
			if secret != "" {
				p.Secret = map[string]string{"secret.conf": secret}
			}
			return p, nil
		},
		Container: func(in kubernetes.Instance) (kubernetes.Container, error) {
			// One host image serves every instance of the solution; the
			// deployment pipeline substitutes the token. The host reads
			// the site fragments given as args and enforces the extern
			// gate again at startup.
			c := kubernetes.Container{
				Image: "${" + host.EnvName(m.cfg.Solution) + "_HOST_IMAGE}",
				Args:  []string{kubernetes.ConfigMount + "/site.conf"},
				Env: []kubernetes.EnvVar{
					{Name: host.EnvName(in.Record.Name), ConfigMapKey: in.Record.Name},
					{Name: "HEALTH", ConfigMapKey: "health"},
				},
				Ports: []kubernetes.Port{{Name: "health", ContainerPort: 3000}},
				Readiness: &kubernetes.HTTPProbe{
					Path: "/readyz", Port: "health",
					PeriodSeconds: 5, FailureThreshold: 3,
				},
				Liveness: &kubernetes.HTTPProbe{
					Path: "/healthz", Port: "health",
					InitialDelaySeconds: 10, PeriodSeconds: 6, FailureThreshold: 10,
				},
			}
			if _, secret := siteFragments(m, st, in.Record.Name); secret != "" {
				c.Args = append(c.Args, kubernetes.SecretMount+"/secret.conf")
			}
			return c, nil
		},
	}
	return kubernetes.Render(m.img, ksite, prof)
}

// siteFragments assembles one instance's split site fragments:
// everything the instance's closure consumes, one "key = value" line
// each, sensitive lines in the secret half (empty when nothing tainted
// reaches the instance). Vars ride along only when the site overrides
// them — image defaults are baked in the host binary already.
func siteFragments(m *model, st *site.Site, instance string) (conf, secret string) {
	needs := m.cfg.Needs([]string{instance})

	var plain, tainted []string
	for _, sym := range m.cfg.Symbols {
		switch sym.Class {
		case host.ClassExtern:
			if !needs.Externs[sym.Name] {
				continue
			}
			v, _ := st.Extern(sym.Name) // generation gate guaranteed presence
			line := fmt.Sprintf("extern.%s = %s", sym.Name, v)
			if sym.Sensitive {
				tainted = append(tainted, line)
			} else {
				plain = append(plain, line)
			}
		case host.ClassVar:
			if !needs.Vars[sym.Name] {
				continue
			}
			if v, ok := st.Var(sym.Name); ok {
				plain = append(plain, fmt.Sprintf("var.%s = %s", sym.Name, v))
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
				tainted = append(tainted, line)
			} else {
				plain = append(plain, line)
			}
		}
	}
	if len(plain) == 0 {
		plain = []string{"# nothing site-bound reaches this instance"}
	}
	conf = strings.Join(plain, "\n") + "\n"
	if len(tainted) > 0 {
		secret = strings.Join(tainted, "\n") + "\n"
	}
	return conf, secret
}
