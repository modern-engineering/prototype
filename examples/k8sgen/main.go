// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Command k8sgen is the committed static-emitter demonstration: the
// D-16 shape, rendering ONE solution image plus ONE site's bindings
// into the static manifest set a Kubernetes environment applies.
// Nothing here talks to a cluster — the output is files, `kubectl
// apply` is the environment's own act, and each rendered pod runs an
// ordinary prebuilt host (examples/host) against its mounted shard.
//
//	k8sgen -image <file> -o <dir> [-extern name=value]... [-var name=value]...
//	       [-namespace name]
//
// The zero profile renders the basic contract; platform teams with
// house conventions write their own command around
// [kubernetes.Render] the way this one is written.
//
// -namespace bakes the objects' namespace at render time, one more
// per-environment knob beside the site bindings. Left unset, the
// manifests stay namespace-free and the apply chooses (kubectl -n, a
// GitOps destination) — the same rendered set then serves any number
// of environments. A namespace is never the solution's to name;
// whether it is the render's or the apply's is the environment
// pipeline's split to pick.
//
// Exit codes follow the taxonomy hosts answer to: 2 for configuration
// faults (bad arguments, an unreadable image, a site the render gate
// refuses — rerunning without changing inputs reproduces it), 1 for
// operational failures (the output directory not taking the files).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/modern-engineering/prototype/solution/image"
	"github.com/modern-engineering/prototype/solution/kubernetes"
)

func main() {
	path := flag.String("image", "", "path to the solution image to render")
	out := flag.String("o", "", "directory to write the manifest set into")
	namespace := flag.String("namespace", "", "bake this namespace into the objects; empty leaves the apply to choose")
	site := kubernetes.Site{Externs: make(map[string]string), Vars: make(map[string]string)}
	bind := func(dst map[string]string) func(string) error {
		return func(arg string) error {
			name, value, ok := strings.Cut(arg, "=")
			if !ok || name == "" {
				return fmt.Errorf("%q is not name=value", arg)
			}
			dst[name] = value
			return nil
		}
	}
	flag.Func("extern", "bind one extern symbol as `name=value` (repeatable)", bind(site.Externs))
	flag.Func("var", "rebind one var symbol as `name=value` (repeatable)", bind(site.Vars))
	flag.Parse() // flag.ExitOnError: bad arguments exit 2 here
	if *path == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "k8sgen: -image and -o are required")
		os.Exit(2)
	}

	f, err := os.Open(*path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "k8sgen: %v\n", err)
		os.Exit(2)
	}
	img, err := image.Decode(f)
	f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "k8sgen: %v\n", err)
		os.Exit(2)
	}

	files, err := kubernetes.Render(img, site, kubernetes.Profile{Namespace: *namespace})
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	if err := os.MkdirAll(*out, 0o777); err != nil {
		fmt.Fprintf(os.Stderr, "k8sgen: %v\n", err)
		os.Exit(1)
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(*out, name), files[name], 0o666); err != nil {
			fmt.Fprintf(os.Stderr, "k8sgen: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(name)
	}
}
