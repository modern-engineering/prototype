// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution_test

import (
	"flag"
	"reflect"
	"strings"
	"testing"

	"github.com/modern-engineering/prototype/solution"
	"github.com/modern-engineering/prototype/solution/image"
)

// podSchemeType mirrors the community-conventions shape the worked
// examples adopt: a two-key scheme under a dotted qualifier.
func podSchemeType() *solution.SchemeType {
	return &solution.SchemeType{
		Doc:       "pod-level scheduling conventions",
		Qualifier: "k8s.pod",
		Params: func(fs *flag.FlagSet) {
			fs.Int("replicas", 1, "desired pod replicas")
			fs.String("priorityClass", "", "scheduling priority class")
		},
	}
}

// schemeCatalogue extends the test catalogue with a scheme package: a
// keyed scheme and a keyless one (nil Params claims the qualifier and
// declares nothing).
func schemeCatalogue() []solution.Package {
	return append(testCatalogue(), solution.Package{
		Path: "example.com/acme/k8s",
		Name: "k8s",
		Elements: []solution.Element{
			solution.Scheme("Pod", podSchemeType()),
			solution.Scheme("Workload", &solution.SchemeType{
				Doc:       "workload grouping conventions",
				Qualifier: "k8s.workload",
			}),
		},
	})
}

// TestMainCompileSchemePinning proves registration alone pins a
// scheme into the image's catalogue: kind, doc, the self-declared
// qualifier, and the key schema extracted from a dry surface — no
// unit needs to import the package or write a stanza for the pin to
// exist, since the registration is what the compilation was checked
// against.
func TestMainCompileSchemePinning(t *testing.T) {
	code, stdout, stderr := compile(t, solution.CompileConfig{
		Solution:  "sample",
		Units:     []solution.Unit{{Name: "u.sdl", Source: "solution sample\n"}},
		Catalogue: schemeCatalogue(),
	})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	img, err := image.Decode(strings.NewReader(stdout))
	if err != nil {
		t.Fatalf("decoding the image: %v", err)
	}
	var pkg *image.Package
	for i := range img.Catalogue {
		if img.Catalogue[i].Path == "example.com/acme/k8s" {
			pkg = &img.Catalogue[i]
		}
	}
	if pkg == nil {
		t.Fatal("image pins no example.com/acme/k8s package")
	}
	want := []image.ElementSchema{
		{
			Name:      "Pod",
			Kind:      image.KindScheme,
			Doc:       "pod-level scheduling conventions",
			Qualifier: "k8s.pod",
			Params: []image.ParamSchema{
				{Name: "priorityClass", Usage: "scheduling priority class"},
				{Name: "replicas", Usage: "desired pod replicas", Default: "1"},
			},
		},
		{
			Name:      "Workload",
			Kind:      image.KindScheme,
			Doc:       "workload grouping conventions",
			Qualifier: "k8s.workload",
		},
	}
	if !reflect.DeepEqual(pkg.Elements, want) {
		t.Errorf("pinned elements = %+v, want %+v", pkg.Elements, want)
	}
}

// TestMainCompileSchemeMisuse pins the kind diagnostics: a scheme
// element is neither deployable, provisionable, extern-typable, nor
// defaultable — stanzas are its one attachment point.
func TestMainCompileSchemeMisuse(t *testing.T) {
	tests := []struct {
		name string
		decl string
		want string
	}{
		{
			name: "deploy",
			decl: "deploy k8s.Pod as broken\n",
			want: "u.sdl:3:8: cannot deploy k8s.Pod: element is a scheme type, not a component",
		},
		{
			name: "provision",
			decl: "provision k8s.Pod as broken\n",
			want: "u.sdl:3:11: cannot provision k8s.Pod: element is a scheme type, not a provision type",
		},
		{
			name: "extern",
			decl: "extern broken k8s.Pod\n",
			want: "u.sdl:3:15: element k8s.Pod is a scheme type, not a symbol type",
		},
		{
			name: "default",
			decl: "default k8s.Pod {\n\tparams {\n\t\treplicas: 2\n\t}\n}\n",
			want: "u.sdl:3:9: cannot default k8s.Pod: stanza defaults ride the statement verbs, e.g. default deploy { with k8s.pod { ... } }",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, stderr := compile(t, solution.CompileConfig{
				Solution: "sample",
				Units: []solution.Unit{{Name: "u.sdl", Source: "solution sample\n" +
					"import k8s \"example.com/acme/k8s\"\n" + tt.decl}},
				Catalogue: schemeCatalogue(),
			})
			if code != 1 {
				t.Fatalf("exit code = %d, want 1; stderr:\n%s", code, stderr)
			}
			if stderr != tt.want+"\n" {
				t.Errorf("stderr = %q, want %q", stderr, tt.want+"\n")
			}
		})
	}
}
