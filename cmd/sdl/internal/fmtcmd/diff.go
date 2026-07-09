// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package fmtcmd

import (
	"bytes"
	"fmt"
	"strings"
)

// diffContext is the number of unchanged lines shown around each change,
// the unified-diff convention.
const diffContext = 3

// An op is one line of the edit script: kept (' '), deleted ('-'), or
// inserted ('+'). The line text includes its trailing newline when the
// input had one.
type op struct {
	tag  byte
	line string
}

// unifiedDiff renders a unified diff between old and new. It replaces
// shelling out to an external diff tool: sdl fmt -d must not depend on
// the host's toolbox. Line numbers are 1-based; a missing final newline
// is marked the way GNU diff marks it.
func unifiedDiff(oldName, newName string, old, new []byte) []byte {
	ops := diffOps(splitLines(old), splitLines(new))

	// Group the changed ops into hunks: two changes belong to one hunk
	// when at most 2*diffContext kept lines separate them, so their
	// context runs would touch or overlap.
	var groups [][2]int // [first, last] changed indices per hunk
	for i, o := range ops {
		if o.tag == ' ' {
			continue
		}
		if n := len(groups); n > 0 && i-groups[n-1][1] <= 2*diffContext+1 {
			groups[n-1][1] = i
			continue
		}
		groups = append(groups, [2]int{i, i})
	}
	if len(groups) == 0 {
		return nil
	}

	// aAt[i] and bAt[i] count the old and new lines before ops[i].
	aAt := make([]int, len(ops)+1)
	bAt := make([]int, len(ops)+1)
	for i, o := range ops {
		aAt[i+1], bAt[i+1] = aAt[i], bAt[i]
		if o.tag != '+' {
			aAt[i+1]++
		}
		if o.tag != '-' {
			bAt[i+1]++
		}
	}

	var out bytes.Buffer
	fmt.Fprintf(&out, "--- %s\n+++ %s\n", oldName, newName)
	for _, g := range groups {
		start := max(g[0]-diffContext, 0)
		end := min(g[1]+diffContext+1, len(ops))
		fmt.Fprintf(&out, "@@ -%s +%s @@\n",
			hunkRange(aAt[start]+1, aAt[end]-aAt[start]),
			hunkRange(bAt[start]+1, bAt[end]-bAt[start]))
		for _, o := range ops[start:end] {
			out.WriteByte(o.tag)
			out.WriteString(o.line)
			if !strings.HasSuffix(o.line, "\n") {
				out.WriteString("\n\\ No newline at end of file\n")
			}
		}
	}
	return out.Bytes()
}

// hunkRange renders one side of a @@ header; unified diff starts an
// empty range at the line before it.
func hunkRange(start, count int) string {
	if count == 0 {
		start--
	}
	return fmt.Sprintf("%d,%d", start, count)
}

// diffOps computes a line-level edit script from a to b via the classic
// longest-common-subsequence table; ties prefer deletion first, the
// conventional diff shape. The inputs fmt feeds it are one file's two
// renderings, so the quadratic table stays small.
func diffOps(a, b []string) []op {
	n, m := len(a), len(b)
	// lcs[i][j] is the LCS length of a[i:] and b[j:].
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}

	ops := make([]op, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			ops = append(ops, op{' ', a[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			ops = append(ops, op{'-', a[i]})
			i++
		default:
			ops = append(ops, op{'+', b[j]})
			j++
		}
	}
	for ; i < n; i++ {
		ops = append(ops, op{'-', a[i]})
	}
	for ; j < m; j++ {
		ops = append(ops, op{'+', b[j]})
	}
	return ops
}

// splitLines splits text into lines that keep their newline, so a
// missing final newline stays visible to the diff.
func splitLines(text []byte) []string {
	lines := strings.SplitAfter(string(text), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
