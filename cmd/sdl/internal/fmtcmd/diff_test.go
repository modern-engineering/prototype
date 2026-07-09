// Copyright 2026 The prototype authors. Use of this source code is
// governed by the license that can be found in the LICENSE file.

package fmtcmd

import "testing"

func TestUnifiedDiff(t *testing.T) {
	tests := map[string]struct {
		old, new string
		want     string
	}{
		"Identical": {
			old:  "a\nb\n",
			new:  "a\nb\n",
			want: "",
		},
		"ChangeWithContextTruncatedAtEdges": {
			old:  "1\n2\nold\n3\n4\n",
			new:  "1\n2\nnew\n3\n4\n",
			want: "--- old\n+++ new\n@@ -1,5 +1,5 @@\n 1\n 2\n-old\n+new\n 3\n 4\n",
		},
		"InsertOnly": {
			old:  "a\nb\n",
			new:  "a\nmid\nb\n",
			want: "--- old\n+++ new\n@@ -1,2 +1,3 @@\n a\n+mid\n b\n",
		},
		"DeleteOnly": {
			old:  "a\nmid\nb\n",
			new:  "a\nb\n",
			want: "--- old\n+++ new\n@@ -1,3 +1,2 @@\n a\n-mid\n b\n",
		},
		"DistantChangesSplitIntoHunks": {
			old:  "x\n1\n2\n3\n4\n5\n6\n7\n8\n9\ny\n",
			new:  "X\n1\n2\n3\n4\n5\n6\n7\n8\n9\nY\n",
			want: "--- old\n+++ new\n@@ -1,4 +1,4 @@\n-x\n+X\n 1\n 2\n 3\n@@ -8,4 +8,4 @@\n 7\n 8\n 9\n-y\n+Y\n",
		},
		"NearChangesShareOneHunk": {
			old:  "x\n1\n2\n3\ny\n",
			new:  "X\n1\n2\n3\nY\n",
			want: "--- old\n+++ new\n@@ -1,5 +1,5 @@\n-x\n+X\n 1\n 2\n 3\n-y\n+Y\n",
		},
		"MissingFinalNewline": {
			old:  "a\nb",
			new:  "a\nb\n",
			want: "--- old\n+++ new\n@@ -1,2 +1,2 @@\n a\n-b\n\\ No newline at end of file\n+b\n",
		},
		"AppendAtEnd": {
			old:  "a\n",
			new:  "a\nb\n",
			want: "--- old\n+++ new\n@@ -1,1 +1,2 @@\n a\n+b\n",
		},
		"EmptyOld": {
			old:  "",
			new:  "a\n",
			want: "--- old\n+++ new\n@@ -0,0 +1,1 @@\n+a\n",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := string(unifiedDiff("old", "new", []byte(tt.old), []byte(tt.new)))
			if got != tt.want {
				t.Errorf("diff:\n%s--- want ---\n%s", got, tt.want)
			}
		})
	}
}
