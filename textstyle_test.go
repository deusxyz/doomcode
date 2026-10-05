package main

import (
	"testing"

	"github.com/rjkroege/edwood/frame"
)

// recordingFrame is a MockFrame that remembers style calls and reports a
// visible window of nchars runes.
type recordingFrame struct {
	MockFrame
	nchars   int
	table    []frame.StyleColours
	restyles []struct {
		p0, p1 int
		styles []uint8
	}
}

func (f *recordingFrame) GetFrameFillStatus() frame.FrameFillStatus {
	return frame.FrameFillStatus{Nchars: f.nchars, Nlines: 1, Maxlines: 10}
}
func (f *recordingFrame) SetStyleTable(t []frame.StyleColours) { f.table = t }
func (f *recordingFrame) Restyle(p0, p1 int, styles []uint8) {
	f.restyles = append(f.restyles, struct {
		p0, p1 int
		styles []uint8
	}{p0, p1, append([]uint8(nil), styles...)})
}

func TestStyleIndices(t *testing.T) {
	text := makeKeyTestBody("func main() {}", 0, 0)
	fr := &recordingFrame{nchars: 14}
	text.fr = fr
	global.styles = nil
	defer func() { global.styles = nil }()

	if idx := text.styleIndices(fr, 0, 14); idx != nil {
		t.Fatalf("no spans: want nil, got %v", idx)
	}
	text.file.Styles().Set(0, 4, "keyword")
	text.file.Styles().Set(5, 9, "function")
	idx := text.styleIndices(fr, 0, 14)
	k, f := global.styles.Index("keyword"), global.styles.Index("function")
	want := []uint8{k, k, k, k, 0, f, f, f, f, 0, 0, 0, 0, 0}
	if len(idx) != len(want) {
		t.Fatalf("len %d; want %d", len(idx), len(want))
	}
	for i := range want {
		if idx[i] != want[i] {
			t.Fatalf("idx = %v; want %v", idx, want)
		}
	}
	if fr.table == nil || len(fr.table) != global.styles.Len() {
		t.Errorf("frame did not get the style table: %v", fr.table)
	}
	// A sub-range: offsets are relative to q0.
	idx = text.styleIndices(fr, 3, 4) // runes 3..6: "c ma"
	if len(idx) != 4 || idx[0] != k || idx[1] != 0 || idx[2] != f || idx[3] != f {
		t.Errorf("sub-range idx = %v; want [%d 0 %d %d]", idx, k, f, f)
	}
	// Tags never have styles.
	tag := &text.w.tag
	tag.file.Styles().Set(0, 2, "keyword")
	if idx := tag.styleIndices(fr, 0, 2); idx != nil {
		t.Errorf("tag got styles: %v", idx)
	}
}

func TestTextRestyleClampsToVisible(t *testing.T) {
	text := makeKeyTestBody("0123456789abcdefghij", 0, 0)
	fr := &recordingFrame{nchars: 10}
	text.fr = fr
	text.org = 5 // showing runes 5..15
	global.styles = nil
	defer func() { global.styles = nil }()

	text.file.Styles().Set(0, 8, "comment")
	text.Restyle(0, 20)
	if len(fr.restyles) != 1 {
		t.Fatalf("restyles: %v", fr.restyles)
	}
	r := fr.restyles[0]
	c := global.styles.Index("comment")
	if r.p0 != 0 || r.p1 != 10 {
		t.Errorf("frame range %d,%d; want 0,10 (file 5..15)", r.p0, r.p1)
	}
	if len(r.styles) != 10 || r.styles[0] != c || r.styles[2] != c || r.styles[3] != 0 {
		t.Errorf("styles %v; want comment for the first 3 runes", r.styles)
	}

	// Out of view: nothing happens.
	fr.restyles = nil
	text.Restyle(16, 20)
	if len(fr.restyles) != 0 {
		t.Errorf("restyle outside the view: %v", fr.restyles)
	}

	// Styles cleared: Restyle still repaints to plain.
	text.file.Styles().ClearAll()
	text.Restyle(5, 15)
	if len(fr.restyles) != 1 || len(fr.restyles[0].styles) != 10 {
		t.Fatalf("restyle to plain: %v", fr.restyles)
	}
	for _, s := range fr.restyles[0].styles {
		if s != 0 {
			t.Errorf("plain restyle has style %d", s)
		}
	}
}
