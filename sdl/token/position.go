// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package token

import (
	"fmt"
	"sort"
	"strconv"
)

// Position describes an arbitrary source position including the file,
// line, and column location. A Position is valid if the line number is
// > 0.
type Position struct {
	Filename string // filename, if any
	Offset   int    // offset, starting at 0
	Line     int    // line number, starting at 1
	Column   int    // column number, starting at 1 (byte count)
}

// IsValid reports whether the position is valid.
func (pos *Position) IsValid() bool { return pos.Line > 0 }

// String returns a string in one of several forms:
//
//	file:line:column    valid position with file name
//	line:column         valid position without file name
//	file                invalid position with file name
//	-                   invalid position without file name
func (pos Position) String() string {
	s := pos.Filename
	if pos.IsValid() {
		if s != "" {
			s += ":"
		}
		s += strconv.Itoa(pos.Line)
		if pos.Column != 0 {
			s += fmt.Sprintf(":%d", pos.Column)
		}
	}
	if s == "" {
		s = "-"
	}
	return s
}

// A File is a handle for one unit of SDL source text. It translates byte
// offsets into Positions by way of a line-offset table that the scanner
// populates through [File.AddLine].
//
// There is deliberately no file set: a solution unit is parsed alone, and
// every node stores materialized Positions. Growing a FileSet later only
// requires allocating Files from it; the File API would not change.
type File struct {
	name  string
	size  int
	lines []int // offsets of the first character of each line; lines[0] == 0
}

// NewFile returns a new File with the given filename and size (the length
// of the source text in bytes).
func NewFile(filename string, size int) *File {
	return &File{name: filename, size: size, lines: []int{0}}
}

// Name returns the file name of file f as registered with NewFile.
func (f *File) Name() string { return f.name }

// Size returns the size of file f as registered with NewFile.
func (f *File) Size() int { return f.size }

// LineCount returns the number of lines in file f.
func (f *File) LineCount() int { return len(f.lines) }

// AddLine adds the line offset for a new line. The line offset must be
// larger than the offset for the previous line and smaller than the file
// size; otherwise the line offset is ignored.
func (f *File) AddLine(offset int) {
	if i := len(f.lines); (i == 0 || f.lines[i-1] < offset) && offset < f.size {
		f.lines = append(f.lines, offset)
	}
}

// Position returns the Position value for the given file offset. The
// offset is clamped to [0, f.Size()].
func (f *File) Position(offset int) Position {
	if offset < 0 {
		offset = 0
	}
	if offset > f.size {
		offset = f.size
	}
	// Find the index i of the largest line offset <= offset.
	i := sort.Search(len(f.lines), func(i int) bool { return f.lines[i] > offset }) - 1
	return Position{
		Filename: f.name,
		Offset:   offset,
		Line:     i + 1,
		Column:   offset - f.lines[i] + 1,
	}
}
