// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package solution

import (
	"fmt"
	"strconv"
	"time"
)

// An OutputWriter carries one provision instance's outputs across the
// driver/host boundary. The host constructs it over the provision
// type's declared output scheme and hands it to the driver's
// [Provisioner.Attach], which writes each output through the typed
// setter matching its declared [OutputType]; the host then reads the
// results back — typed for programmatic consumers, rendered to the
// flag-ready string for wet parameter binding.
//
// The writer validates at the boundary, the way flag.FlagSet.Set does:
// writing an undeclared name, writing through the wrong type's setter,
// and writing one output twice fail at the write, never downstream.
// Completeness is the host's check on the other side of the call —
// once the driver returns, every declared output must be written, and
// [OutputWriter.Missing] lists the ones that are not.
type OutputWriter struct {
	order  []string
	values map[string]*outputValue
}

// An outputValue is one declared output's slot: its normalized type
// and, once written, the value in the arm the type selects.
type outputValue struct {
	typ     OutputType // normalized: an empty declaration means string
	written bool
	str     string
	num     int64
	boolean bool
	dur     time.Duration
}

// NewOutputWriter builds a writer over a provision type's declared
// output scheme, every output unwritten. Declarations are taken as
// registered — catalogue validation has already vetted names and
// types — with the empty type normalized to [OutputString], the
// permissive default.
func NewOutputWriter(outputs []Output) *OutputWriter {
	w := &OutputWriter{values: make(map[string]*outputValue, len(outputs))}
	for _, out := range outputs {
		if _, ok := w.values[out.Name]; ok {
			// Registration rejects duplicate declarations; against a
			// scheme that skipped validation, the first one wins.
			continue
		}
		typ := out.Type
		if typ == "" {
			typ = OutputString
		}
		w.order = append(w.order, out.Name)
		w.values[out.Name] = &outputValue{typ: typ}
	}
	return w
}

// claim looks up one declared output for writing, holding the write to
// the boundary contract: the name is declared, the setter matches the
// declared type, and no earlier write claimed the output.
func (w *OutputWriter) claim(name string, kind OutputType) (*outputValue, error) {
	v, ok := w.values[name]
	if !ok {
		return nil, fmt.Errorf("undeclared output %s", name)
	}
	if v.typ != kind {
		return nil, fmt.Errorf("output %s is %s, not %s", name, v.typ, kind)
	}
	if v.written {
		return nil, fmt.Errorf("output %s written twice", name)
	}
	v.written = true
	return v, nil
}

// SetString writes a string-typed output. Declarations that omit their
// type are string-typed, so this is their setter too.
func (w *OutputWriter) SetString(name, value string) error {
	v, err := w.claim(name, OutputString)
	if err != nil {
		return err
	}
	v.str = value
	return nil
}

// SetInt writes an int-typed output.
func (w *OutputWriter) SetInt(name string, value int64) error {
	v, err := w.claim(name, OutputInt)
	if err != nil {
		return err
	}
	v.num = value
	return nil
}

// SetBool writes a bool-typed output.
func (w *OutputWriter) SetBool(name string, value bool) error {
	v, err := w.claim(name, OutputBool)
	if err != nil {
		return err
	}
	v.boolean = value
	return nil
}

// SetDuration writes a duration-typed output.
func (w *OutputWriter) SetDuration(name string, value time.Duration) error {
	v, err := w.claim(name, OutputDuration)
	if err != nil {
		return err
	}
	v.dur = value
	return nil
}

// read looks up one declared output for typed reading: the name is
// declared, the getter matches the declared type, and the driver wrote
// the value.
func (w *OutputWriter) read(name string, kind OutputType) (*outputValue, error) {
	v, ok := w.values[name]
	if !ok {
		return nil, fmt.Errorf("undeclared output %s", name)
	}
	if v.typ != kind {
		return nil, fmt.Errorf("output %s is %s, not %s", name, v.typ, kind)
	}
	if !v.written {
		return nil, fmt.Errorf("output %s is unwritten", name)
	}
	return v, nil
}

// GetString reads a written string-typed output back.
func (w *OutputWriter) GetString(name string) (string, error) {
	v, err := w.read(name, OutputString)
	if err != nil {
		return "", err
	}
	return v.str, nil
}

// GetInt reads a written int-typed output back.
func (w *OutputWriter) GetInt(name string) (int64, error) {
	v, err := w.read(name, OutputInt)
	if err != nil {
		return 0, err
	}
	return v.num, nil
}

// GetBool reads a written bool-typed output back.
func (w *OutputWriter) GetBool(name string) (bool, error) {
	v, err := w.read(name, OutputBool)
	if err != nil {
		return false, err
	}
	return v.boolean, nil
}

// GetDuration reads a written duration-typed output back.
func (w *OutputWriter) GetDuration(name string) (time.Duration, error) {
	v, err := w.read(name, OutputDuration)
	if err != nil {
		return 0, err
	}
	return v.dur, nil
}

// Render returns a written output as the flag-ready string a wet
// binding passes to flag.Value.Set: the string itself, base-10 for
// ints, "true"/"false" for bools, and the canonical Go form for
// durations — the same spellings the image's typed literals render
// to, so a slot cannot tell a provision output from an inline value
// and the slot's own Set stays the one validator (A-14).
func (w *OutputWriter) Render(name string) (string, error) {
	v, ok := w.values[name]
	if !ok {
		return "", fmt.Errorf("undeclared output %s", name)
	}
	if !v.written {
		return "", fmt.Errorf("output %s is unwritten", name)
	}
	switch v.typ {
	case OutputInt:
		return strconv.FormatInt(v.num, 10), nil
	case OutputBool:
		return strconv.FormatBool(v.boolean), nil
	case OutputDuration:
		return v.dur.String(), nil
	}
	return v.str, nil
}

// Missing lists the declared outputs no setter has written, in
// declaration order. The host requires completeness once the driver
// returns: every declared output written, so each name still listed
// here is the driver's fault to report.
func (w *OutputWriter) Missing() []string {
	var missing []string
	for _, name := range w.order {
		if !w.values[name].written {
			missing = append(missing, name)
		}
	}
	return missing
}
