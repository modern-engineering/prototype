// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package main_test

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The static-emitter demonstration renders the committed pingpong
// image into one manifest file per deploy record, and the rendered
// pod contract holds end to end: the files a pod would mount —
// extracted here from the ConfigMap exactly as the kubelet would
// project them — drive the prebuilt examples/host to a clean
// completion of the finite instance. A smoke over committed bytes:
// the render semantics are solution/kubernetes's pins, the wet
// semantics solution/host's, and the image bytes TestBuildPingpong's.
func TestRenderedPodContract(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e drives the Go toolchain; skipped in -short mode")
	}
	bin := t.TempDir()
	for _, tool := range []string{"k8sgen", "host"} {
		build := exec.Command("go", "build", "-o", filepath.Join(bin, tool), "./examples/"+tool)
		build.Dir = repoRoot
		if out, err := build.CombinedOutput(); err != nil {
			t.Fatalf("building examples/%s: %v\n%s", tool, err, out)
		}
	}

	outDir := filepath.Join(t.TempDir(), "manifests")
	gen := exec.Command(filepath.Join(bin, "k8sgen"),
		"-image", filepath.Join("testdata", "pingpong.json"),
		"-o", outDir,
		"-extern", "natsEndpoint=nats://e2e.invalid:4222",
	)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("k8sgen: %v\n%s", err, out)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	want := []string{"ping1.yaml", "ping2.yaml", "pong.yaml"}
	if strings.Join(names, " ") != strings.Join(want, " ") {
		t.Fatalf("rendered set = %v, want %v", names, want)
	}

	// The extern's endpoint is not sensitive, so nothing may render a
	// Secret; the whole site rides ConfigMaps.
	for _, name := range want {
		doc, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(doc), "kind: Secret") {
			t.Errorf("%s renders a Secret for a plain-extern solution", name)
		}
	}

	// The namespace knob: unset leaves the set namespace-free for the
	// apply to choose (the default asserted above by omission); set,
	// it bakes into every object at render.
	bakedDir := filepath.Join(t.TempDir(), "baked")
	baked := exec.Command(filepath.Join(bin, "k8sgen"),
		"-image", filepath.Join("testdata", "pingpong.json"),
		"-o", bakedDir,
		"-namespace", "pingpong-dev7",
		"-extern", "natsEndpoint=nats://e2e.invalid:4222",
	)
	if out, err := baked.CombinedOutput(); err != nil {
		t.Fatalf("k8sgen -namespace: %v\n%s", err, out)
	}
	plain, err := os.ReadFile(filepath.Join(outDir, "pong.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), "namespace:") {
		t.Error("the default render names a namespace; the apply's choice was pre-empted")
	}
	bakedDoc, err := os.ReadFile(filepath.Join(bakedDir, "pong.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(bakedDoc), "namespace: pingpong-dev7"); got != 2 {
		t.Errorf("baked render names the namespace on %d objects, want its ConfigMap and Deployment", got)
	}

	// Project ping1's mounted files the way the kubelet would and run
	// the pod's process: the finite Ping1 completes, so a clean exit
	// is the whole verdict — no signal choreography.
	ping1, err := os.ReadFile(filepath.Join(outDir, "ping1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	mount := t.TempDir()
	for _, file := range []string{"image.json", "externs.conf"} {
		content := blockScalar(t, string(ping1), file)
		if err := os.WriteFile(filepath.Join(mount, file), []byte(content), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	pod := exec.Command(filepath.Join(bin, "host"),
		"-image", filepath.Join(mount, "image.json"),
		"-externs", filepath.Join(mount, "externs.conf"),
	)
	out, err := pod.CombinedOutput()
	t.Logf("pod transcript:\n%s", out)
	if err != nil {
		t.Fatalf("the rendered pod contract failed: %v", err)
	}
	for _, wants := range []string{"provision natsStandIn", "deploy Ping1", "running 1 instance(s)"} {
		if !strings.Contains(string(out), wants) {
			t.Errorf("pod transcript misses %q", wants)
		}
	}
}

// blockScalar extracts one ConfigMap file's content from a rendered
// manifest: the lines below "  <name>: |", de-indented by the four
// spaces the renderer indents block scalars with.
func blockScalar(t *testing.T, doc, name string) string {
	t.Helper()
	sc := bufio.NewScanner(strings.NewReader(doc))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	var b strings.Builder
	in := false
	for sc.Scan() {
		line := sc.Text()
		if line == "  "+name+": |" {
			in = true
			continue
		}
		if !in {
			continue
		}
		if strings.HasPrefix(line, "    ") {
			b.WriteString(line[4:])
			b.WriteByte('\n')
			continue
		}
		if line == "" {
			b.WriteByte('\n')
			continue
		}
		break
	}
	if !in {
		t.Fatalf("manifest carries no ConfigMap file %s", name)
	}
	return b.String()
}
