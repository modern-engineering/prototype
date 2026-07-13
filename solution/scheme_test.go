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

// TestMainCompileSchemeChecking pins the pre-fold stanza checks: keys
// against the discovered scheme's declared set, literal values
// through its dry surface, in instance bodies and both default
// flavors alike, with faults aggregating rather than short-circuiting.
func TestMainCompileSchemeChecking(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{
			name: "unknown key",
			source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\twith k8s.pod {\n" +
				"\t\tunknownKey: 1\n" +
				"\t}\n" +
				"}\n",
			want: []string{"u.sdl:5:3: unknown key unknownKey in with k8s.pod (scheme k8s.Pod)"},
		},
		{
			name: "invalid literal for a typed key",
			source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\twith k8s.pod {\n" +
				"\t\treplicas: \"three\"\n" +
				"\t}\n" +
				"}\n",
			want: []string{"u.sdl:5:13: invalid value three for k8s.pod key replicas: parse error"},
		},
		{
			name: "faults aggregate within a stanza",
			source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\twith k8s.pod {\n" +
				"\t\tunknownKey: 1\n" +
				"\t\treplicas: \"three\"\n" +
				"\t}\n" +
				"}\n",
			want: []string{
				"u.sdl:5:3: unknown key unknownKey in with k8s.pod (scheme k8s.Pod)",
				"u.sdl:6:13: invalid value three for k8s.pod key replicas: parse error",
			},
		},
		{
			name: "keyless scheme rejects every key",
			source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\twith k8s.workload {\n" +
				"\t\tx: 1\n" +
				"\t}\n" +
				"}\n",
			want: []string{"u.sdl:5:3: unknown key x in with k8s.workload (scheme k8s.Workload)"},
		},
		{
			name: "verb default stanza checked at collection",
			source: "solution sample\n" +
				"default deploy {\n" +
				"\twith k8s.pod {\n" +
				"\t\tunknownKey: 1\n" +
				"\t}\n" +
				"}\n",
			want: []string{"u.sdl:4:3: unknown key unknownKey in with k8s.pod (scheme k8s.Pod)"},
		},
		{
			name: "type default stanza checked at collection",
			source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"default ff.Ping {\n" +
				"\twith k8s.pod {\n" +
				"\t\tunknownKey: 1\n" +
				"\t}\n" +
				"}\n",
			want: []string{"u.sdl:5:3: unknown key unknownKey in with k8s.pod (scheme k8s.Pod)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := compile(t, solution.CompileConfig{
				Solution:  "sample",
				Units:     []solution.Unit{{Name: "u.sdl", Source: tt.source}},
				Catalogue: schemeCatalogue(),
			})
			if code != 1 {
				t.Fatalf("exit code = %d, want 1; stderr:\n%s", code, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if want := strings.Join(tt.want, "\n") + "\n"; stderr != want {
				t.Errorf("stderr = %q, want %q", stderr, want)
			}
		})
	}
}

// TestMainCompileSchemeAdvisoryContract proves the two paths that
// must coexist in one unit: a stanza whose qualifier resolves is
// checked yet binds exactly like an opaque one (same compartment
// shape, token values passing unvalidated — a profile token's
// eventual value is the controller's business, whatever the declared
// key type), while a qualifier no package claims rides the image
// opaquely, verbatim, exactly as before schemes existed.
func TestMainCompileSchemeAdvisoryContract(t *testing.T) {
	source := "solution sample\n" +
		"import ff \"example.com/acme/pingpong\"\n" +
		"deploy ff.Ping as P {\n" +
		"\twith k8s.pod {\n" +
		"\t\treplicas: 3\n" +
		"\t\tpriorityClass: critical\n" +
		"\t}\n" +
		"\twith acme.rollout {\n" +
		"\t\tanything: fast\n" +
		"\t\tcount: 5\n" +
		"\t}\n" +
		"}\n"
	code, stdout, stderr := compile(t, solution.CompileConfig{
		Solution:  "sample",
		Units:     []solution.Unit{{Name: "u.sdl", Source: source}},
		Catalogue: schemeCatalogue(),
	})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
	img, err := image.Decode(strings.NewReader(stdout))
	if err != nil {
		t.Fatalf("decoding the image: %v", err)
	}
	if len(img.Records) != 1 {
		t.Fatalf("records = %d, want 1", len(img.Records))
	}
	want := map[string][]image.Binding{
		"k8s.pod": {
			{Key: "priorityClass", Value: image.Token("critical"), Source: image.SourceInstance},
			{Key: "replicas", Value: image.Int(3), Source: image.SourceInstance},
		},
		"acme.rollout": {
			{Key: "anything", Value: image.Token("fast"), Source: image.SourceInstance},
			{Key: "count", Value: image.Int(5), Source: image.SourceInstance},
		},
	}
	if got := img.Records[0].Extensions; !reflect.DeepEqual(got, want) {
		t.Errorf("extensions = %+v, want %+v", got, want)
	}
}

// TestMainCompileSchemeTokenPassthrough pins the token rule on its
// own: a bare identifier passes a typed key unvalidated even where a
// literal of the wrong kind would fail.
func TestMainCompileSchemeTokenPassthrough(t *testing.T) {
	source := "solution sample\n" +
		"import ff \"example.com/acme/pingpong\"\n" +
		"deploy ff.Ping as P {\n" +
		"\twith k8s.pod {\n" +
		"\t\treplicas: notAnInt\n" +
		"\t}\n" +
		"}\n"
	code, _, stderr := compile(t, solution.CompileConfig{
		Solution:  "sample",
		Units:     []solution.Unit{{Name: "u.sdl", Source: source}},
		Catalogue: schemeCatalogue(),
	})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

// TestMainCompileSchemeParamsPanic drives the recover guards around
// the one place scheme user code runs: a panicking Params aborts with
// exit 2, positioned at the referencing stanza when one exists and
// positionless when only the catalogue pin reaches the element.
func TestMainCompileSchemeParamsPanic(t *testing.T) {
	catalogue := append(testCatalogue(), solution.Package{
		Path: "example.com/acme/boom",
		Name: "boom",
		Elements: []solution.Element{solution.Scheme("Boom", &solution.SchemeType{
			Doc:       "panic on dry instantiation",
			Qualifier: "boom.q",
			Params:    func(*flag.FlagSet) { panic("zap") },
		})},
	})
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name: "referenced by a stanza",
			source: "solution sample\n" +
				"import ff \"example.com/acme/pingpong\"\n" +
				"deploy ff.Ping as P {\n" +
				"\twith boom.q {\n" +
				"\t}\n" +
				"}\n",
			want: "u.sdl:4:7: element boom.Boom: Params panicked: zap\n",
		},
		{
			name:   "unreferenced, pinned at emit",
			source: "solution sample\n",
			want:   "compile: element boom.Boom: Params panicked: zap\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := compile(t, solution.CompileConfig{
				Solution:  "sample",
				Units:     []solution.Unit{{Name: "u.sdl", Source: tt.source}},
				Catalogue: catalogue,
			})
			if code != 2 {
				t.Fatalf("exit code = %d, want 2; stderr:\n%s", code, stderr)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if stderr != tt.want {
				t.Errorf("stderr = %q, want %q", stderr, tt.want)
			}
		})
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
