// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package scanner

import (
	"fmt"
	"sort"

	"github.com/modern-engineering/prototype/sdl/token"
)

// In an [ErrorList], an error is represented by an *Error. The position
// Pos, if valid, points to the beginning of the offending token, and the
// error condition is described by Msg.
type Error struct {
	Pos token.Position
	Msg string
}

// Error implements the error interface.
func (e Error) Error() string {
	if e.Pos.Filename != "" || e.Pos.IsValid() {
		// Print the file name, line, and column.
		return e.Pos.String() + ": " + e.Msg
	}
	return e.Msg
}

// ErrorList is a list of *Errors. The zero value for an ErrorList is an
// empty ErrorList ready to use.
type ErrorList []*Error

// Add adds an [Error] with given position and error message to an
// [ErrorList].
func (p *ErrorList) Add(pos token.Position, msg string) {
	*p = append(*p, &Error{pos, msg})
}

// ErrorList implements the sort Interface.
func (p ErrorList) Len() int      { return len(p) }
func (p ErrorList) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p ErrorList) Less(i, j int) bool {
	e := &p[i].Pos
	f := &p[j].Pos
	if e.Filename != f.Filename {
		return e.Filename < f.Filename
	}
	if e.Line != f.Line {
		return e.Line < f.Line
	}
	if e.Column != f.Column {
		return e.Column < f.Column
	}
	return p[i].Msg < p[j].Msg
}

// Sort sorts an [ErrorList]. *[Error] entries are sorted by position,
// other errors are sorted by error message, and before any *[Error]
// entry.
func (p ErrorList) Sort() {
	sort.Sort(p)
}

// An ErrorList implements the error interface.
func (p ErrorList) Error() string {
	switch len(p) {
	case 0:
		return "no errors"
	case 1:
		return p[0].Error()
	}
	return fmt.Sprintf("%s (and %d more errors)", p[0], len(p)-1)
}

// Err returns an error equivalent to this error list. If the list is
// empty, Err returns nil.
func (p ErrorList) Err() error {
	if len(p) == 0 {
		return nil
	}
	return p
}
