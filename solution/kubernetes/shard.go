// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

// Package kubernetes is the static-emitter controller for the
// Kubernetes archetype: it consumes a solution's desired-state image
// dry — no live catalogue, no cluster client, never an apply — and
// derives the artifacts a Kubernetes environment deploys statically
// (D-16). The pod is the enactment boundary: whatever this package
// emits, the process inside the pod is an ordinary image-enacting
// host, so plans keep loading where D-13 put them and nothing here
// re-implements the wet half.
//
// [Shard] is the package's foundation: one deploy record's slice of
// the desired state, derived as an image in its own right, so a pod
// receives exactly the state it enacts and nothing beyond its reach.
package kubernetes

import (
	"fmt"
	"strconv"
	"time"

	"github.com/modern-engineering/prototype/solution/image"
)

// Shard derives the sub-image of one deploy record: the record
// itself, the provision records its parameters reach transitively,
// and the symbols that closure references — over the full pinned
// catalogue and the image's provenance, carried verbatim. The result
// is a desired-state image in its own right: a composed frontend
// document owing the composer's calls, which Shard runs before
// returning, so a shard always validates. Sharding the same image the
// same way yields byte-identical encodings.
//
// rebind rewrites reached var symbols to site-supplied values, parsed
// under the kind of the var's compile-bound default — the symbol
// table stays the one place a site override lands (A-10), and the
// pod's host reads the rebound default like any other. Extern symbols
// are deliberately untouched: they stay late-bound in the shard, and
// their site values travel outside the image. A rebind naming a var
// the closure never reaches is ignored; judging a site's keys against
// the whole solution is the renderer's gate, not the shard's.
//
// The record's advisory compartments (deployment intent, extension
// stanzas, metadata) ride the shard untouched, opaque to the host
// that enacts it — the advisory contract, unchanged by transport.
func Shard(img *image.Image, instance string, rebind map[string]string) (*image.Image, error) {
	if err := img.Validate(); err != nil {
		return nil, err
	}

	var rec *image.Record
	for i := range img.Records {
		if img.Records[i].Name == instance {
			rec = &img.Records[i]
			break
		}
	}
	if rec == nil {
		return nil, fmt.Errorf("kubernetes: image deploys no instance %s", instance)
	}
	if rec.Verb != image.VerbDeploy {
		return nil, fmt.Errorf("kubernetes: instance %s is a %s record; only deploy records shard into pods", instance, rec.Verb)
	}

	needs := closure(img, rec)

	sub := &image.Image{
		Format:     img.Format,
		Solution:   img.Solution,
		Generation: img.Generation,
		Build:      img.Build,
		Catalogue:  img.Catalogue,
	}
	for _, def := range img.Symbols {
		switch def.Class {
		case image.ClassExtern:
			if !needs.externs[def.Name] {
				continue
			}
		case image.ClassVar:
			if !needs.vars[def.Name] {
				continue
			}
			if raw, ok := rebind[def.Name]; ok {
				val, err := rebindValue(def, raw)
				if err != nil {
					return nil, err
				}
				def.Value = val
			}
		}
		sub.Symbols = append(sub.Symbols, def)
	}
	// Records keep image order — statement order is meaning the image
	// must keep, and the closure is a selection, not a rearrangement.
	for i := range img.Records {
		r := &img.Records[i]
		if r.Name == instance || needs.provisions[r.Name] {
			sub.Records = append(sub.Records, *r)
		}
	}

	sub.Canonicalize()
	if err := sub.Validate(); err != nil {
		return nil, fmt.Errorf("kubernetes: shard %s: %w", instance, err)
	}
	return sub, nil
}

// A needs is the closure of what one deploy record consumes: the
// symbols its bindings reference and the provision records those
// bindings and the provisions' own parameters reach, walked to a
// fixpoint. The linker guarantees the reference graph acyclic, so the
// walk terminates; outputs record which of a provision's outputs the
// closure actually consumes, for consumers that meter delivery.
type needs struct {
	externs    map[string]bool
	vars       map[string]bool
	provisions map[string]bool
	outputs    map[string]map[string]bool
}

// closure walks rec's reference graph over a validated image.
func closure(img *image.Image, rec *image.Record) needs {
	symbols := make(map[string]string, len(img.Symbols))
	for _, def := range img.Symbols {
		symbols[def.Name] = def.Class
	}
	records := make(map[string]*image.Record, len(img.Records))
	for i := range img.Records {
		records[img.Records[i].Name] = &img.Records[i]
	}

	n := needs{
		externs:    make(map[string]bool),
		vars:       make(map[string]bool),
		provisions: make(map[string]bool),
		outputs:    make(map[string]map[string]bool),
	}
	frontier := []*image.Record{rec}
	for len(frontier) > 0 {
		r := frontier[0]
		frontier = frontier[1:]
		for _, b := range r.Params {
			if b.Ref == nil {
				continue
			}
			if b.Ref.Output == "" {
				switch symbols[b.Ref.Symbol] {
				case image.ClassExtern:
					n.externs[b.Ref.Symbol] = true
				case image.ClassVar:
					n.vars[b.Ref.Symbol] = true
				}
				continue
			}
			if n.outputs[b.Ref.Symbol] == nil {
				n.outputs[b.Ref.Symbol] = make(map[string]bool)
			}
			n.outputs[b.Ref.Symbol][b.Ref.Output] = true
			if !n.provisions[b.Ref.Symbol] {
				n.provisions[b.Ref.Symbol] = true
				frontier = append(frontier, records[b.Ref.Symbol])
			}
		}
	}
	return n
}

// rebindValue parses one site-supplied var override under the kind of
// the var's compile-bound default, so a rebound symbol table stays
// typed the way the compiler left it and flag surfaces meet the same
// spellings wet binding renders.
func rebindValue(def image.SymbolDef, raw string) (*image.Value, error) {
	switch def.Value.Kind {
	case image.KindString:
		return image.String(raw), nil
	case image.KindInt:
		i, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("kubernetes: rebind var %s: %q is not an int", def.Name, raw)
		}
		return image.Int(i), nil
	case image.KindBool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("kubernetes: rebind var %s: %q is not a bool", def.Name, raw)
		}
		return image.Bool(b), nil
	case image.KindDuration:
		d, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("kubernetes: rebind var %s: %q is not a duration", def.Name, raw)
		}
		return image.Duration(d), nil
	}
	return nil, fmt.Errorf("kubernetes: rebind var %s: defaults of kind %q cannot be rebound", def.Name, def.Value.Kind)
}
