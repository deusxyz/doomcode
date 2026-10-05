package main

import (
	"testing"
)

func TestParseDiagnostics(t *testing.T) {
	body := "/a/b.go:12.5,12.9: undefined: foo\n" +
		"/a/b.go:3.1,3.1: warning: x is unused\n" +
		"/a/c.go:1.1,2.4: deprecated: use y\n" +
		"not a diagnostic line\n" +
		"/a/c.go:7.2,7.2: hint: consider z\n"
	got := parseDiagnostics(body, "error")
	if len(got) != 2 || len(got["/a/b.go"]) != 2 || len(got["/a/c.go"]) != 2 {
		t.Fatalf("groups: %+v", got)
	}
	d := got["/a/b.go"][0]
	if d.Line0 != 12 || d.Col0 != 5 || d.Line1 != 12 || d.Col1 != 9 || d.Message != "undefined: foo" || d.Style != "error" {
		t.Errorf("d0 = %+v", d)
	}
	if got["/a/b.go"][1].Style != "warning" {
		t.Errorf("warning not detected: %+v", got["/a/b.go"][1])
	}
	if got["/a/c.go"][0].Style != "warning" || got["/a/c.go"][1].Style != "hint" {
		t.Errorf("c.go styles: %+v", got["/a/c.go"])
	}
	if got := parseDiagnostics("/a/b.go:1.1,1.2: m\n", "info"); got["/a/b.go"][0].Style != "info" {
		t.Errorf("default style not used: %+v", got)
	}
}

func TestStyleWrite(t *testing.T) {
	// Cyrillic first line: offsets must be in runes.
	body := []byte("// привет\nfunc main() {\n\tfoo()\n}\n")
	// Line 3 col 2..5 is "foo"; line 2 col 6 is the start of "main".
	diags := []diagnostic{
		{Line0: 3, Col0: 2, Line1: 3, Col1: 5, Style: "error"},
		{Line0: 2, Col0: 6, Line1: 2, Col1: 6, Style: "warning"}, // empty range: widened to the word
		{Line0: 99, Col0: 1, Line1: 99, Col1: 1, Style: "error"}, // no such line: dropped
		{Line0: 4, Col0: 50, Line1: 4, Col1: 60, Style: "hint"},  // past the line end: clamped to 1 rune? no: empty after clamp -> one rune
	}
	got := styleWrite(body, diags)
	want := "clear 0 33 error warning info hint\n" +
		"32 33 hint\n" + // line 4 is "}" at rune 31; col 50 clamps to its end, widened to one rune
		"15 19 warning\n" + // "main"
		"25 28 error\n" // "foo"
	if got != want {
		t.Errorf("styleWrite =\n%s\nwant\n%s", got, want)
	}
	// No diagnostics: just the clear.
	if got := styleWrite(body, nil); got != "clear 0 33 error warning info hint\n" {
		t.Errorf("empty = %q", got)
	}
}

func TestOffset(t *testing.T) {
	body := []byte("ab\ncde\n")
	lines := lineStarts(body)
	for _, tc := range []struct {
		line, col, want int
		ok              bool
	}{
		{1, 1, 0, true}, {1, 3, 2, true}, {1, 9, 2, true}, {2, 1, 3, true}, {2, 4, 6, true}, {3, 1, 7, true}, {4, 1, 0, false}, {0, 1, 0, false},
	} {
		q, ok := offset(lines, body, tc.line, tc.col)
		if q != tc.want || ok != tc.ok {
			t.Errorf("offset(%d,%d) = %d,%v; want %d,%v", tc.line, tc.col, q, ok, tc.want, tc.ok)
		}
	}
}
