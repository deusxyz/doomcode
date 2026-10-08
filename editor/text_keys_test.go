package main

import (
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/dumpfile"
)

// makeKeyTestBody builds a single window whose body holds buf with the
// selection at [q0,q1) and returns the body Text.
func makeKeyTestBody(buf string, q0, q1 int) *Text {
	MakeWindowScaffold(&dumpfile.Content{
		Columns: []dumpfile.Column{{}},
		Windows: []*dumpfile.Window{
			{
				Column: 0,
				Tag:    dumpfile.Text{Buffer: ""},
				Body:   dumpfile.Text{Buffer: buf, Q0: q0, Q1: q1},
			},
		},
	})
	return &global.row.col[0].w[0].body
}

func TestLineStartEnd(t *testing.T) {
	text := makeKeyTestBody("ab\n\ncdef\nxyz", 0, 0)
	for _, tc := range []struct {
		q, start, end int
	}{
		{0, 0, 2},   // first line
		{1, 0, 2},   // inside first line
		{2, 0, 2},   // on the newline of the first line
		{3, 3, 3},   // empty line
		{5, 4, 8},   // inside "cdef"
		{9, 9, 12},  // last line without trailing newline
		{12, 9, 12}, // end of buffer
	} {
		if got := text.lineStart(tc.q); got != tc.start {
			t.Errorf("lineStart(%d) = %d; want %d", tc.q, got, tc.start)
		}
		if got := text.lineEnd(tc.q); got != tc.end {
			t.Errorf("lineEnd(%d) = %d; want %d", tc.q, got, tc.end)
		}
	}
}

func TestTypeUpDownKeepsColumn(t *testing.T) {
	// Lines: "abcdef" (0-6), "xy" (7-9), "" (10), "123456" (11-17)
	text := makeKeyTestBody("abcdef\nxy\n\n123456", 4, 4)

	text.Type(draw.KeyDown) // "xy" is short: clamp to its end
	if text.q0 != 9 || text.q1 != 9 {
		t.Fatalf("after Down: q0,q1 = %d,%d; want 9,9", text.q0, text.q1)
	}
	text.Type(draw.KeyDown) // empty line
	if text.q0 != 10 {
		t.Fatalf("after 2nd Down: q0 = %d; want 10", text.q0)
	}
	text.Type(draw.KeyDown) // long line again: sticky column 4 restored
	if text.q0 != 15 {
		t.Fatalf("after 3rd Down: q0 = %d; want 15", text.q0)
	}
	text.Type(draw.KeyDown) // last line: go to end of buffer
	if text.q0 != 17 {
		t.Fatalf("Down on last line: q0 = %d; want 17", text.q0)
	}
	text.Type(draw.KeyUp) // column was reset by the end-of-buffer move: 6 -> clamp to 10
	if text.q0 != 10 {
		t.Fatalf("after Up: q0 = %d; want 10", text.q0)
	}
	text.Type(draw.KeyUp)
	text.Type(draw.KeyUp)
	if text.q0 != 6 {
		t.Fatalf("after Up to first line: q0 = %d; want 6", text.q0)
	}
	text.Type(draw.KeyUp) // first line: go to start of buffer
	if text.q0 != 0 {
		t.Fatalf("Up on first line: q0 = %d; want 0", text.q0)
	}
}

func TestTypeUpDownCollapsesSelection(t *testing.T) {
	text := makeKeyTestBody("abcdef\nxyzuvw\n", 1, 4)
	text.Type(draw.KeyDown) // collapses to q1=4, then moves down keeping column 4
	if text.q0 != 11 || text.q1 != 11 {
		t.Fatalf("Down with selection: q0,q1 = %d,%d; want 11,11", text.q0, text.q1)
	}
	text.SetSelect(8, 10)
	text.Type(draw.KeyUp) // collapses to q0=8 (column 1), then up
	if text.q0 != 1 || text.q1 != 1 {
		t.Fatalf("Up with selection: q0,q1 = %d,%d; want 1,1", text.q0, text.q1)
	}
}

func TestTypeHomeEnd(t *testing.T) {
	text := makeKeyTestBody("abcdef\nxyz\n", 9, 9)
	text.Type(draw.KeyHome)
	if text.q0 != 7 || text.q1 != 7 {
		t.Fatalf("Home: q0,q1 = %d,%d; want 7,7", text.q0, text.q1)
	}
	text.Type(draw.KeyEnd)
	if text.q0 != 10 || text.q1 != 10 {
		t.Fatalf("End: q0,q1 = %d,%d; want 10,10", text.q0, text.q1)
	}
}

func TestTypeSelectAll(t *testing.T) {
	text := makeKeyTestBody("abc\ndef", 2, 2)
	text.Type(0x01)
	if text.q0 != 0 || text.q1 != 7 {
		t.Fatalf("^A: q0,q1 = %d,%d; want 0,7", text.q0, text.q1)
	}
}

func TestTypeKillLine(t *testing.T) {
	text := makeKeyTestBody("abcdef\n\nxyz", 2, 2)
	text.Type(0x0b)
	if got, want := text.file.String(), "ab\n\nxyz"; got != want {
		t.Fatalf("^K: buffer = %q; want %q", got, want)
	}
	if text.q0 != 2 || text.q1 != 2 {
		t.Fatalf("^K: q0,q1 = %d,%d; want 2,2", text.q0, text.q1)
	}
	text.Type(0x0b) // at end of line: delete the newline
	if got, want := text.file.String(), "ab\nxyz"; got != want {
		t.Fatalf("^K on newline: buffer = %q; want %q", got, want)
	}
	text.SetSelect(3, 6)
	text.Type(0x0b) // with a selection: delete the selection
	if got, want := text.file.String(), "ab\n"; got != want {
		t.Fatalf("^K with selection: buffer = %q; want %q", got, want)
	}
	text.Type(0x0b) // at end of buffer: nothing happens
	if got, want := text.file.String(), "ab\n"; got != want {
		t.Fatalf("^K at end: buffer = %q; want %q", got, want)
	}
}
