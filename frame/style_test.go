package frame

import (
	"image"
	"strings"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/edwoodtest"
)

func boxSummary(f *frameimpl) []string {
	var out []string
	for _, b := range f.box {
		s := string(b.Ptr)
		if b.Nrune < 0 {
			s = string(b.Bc)
		}
		out = append(out, s+"/"+string(rune('0'+b.Style)))
	}
	return out
}

func wantBoxes(t *testing.T, what string, f *frameimpl, want ...string) {
	t.Helper()
	got := boxSummary(f)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("%s: boxes %v; want %v", what, got, want)
	}
}

func TestBxscanSplitsOnStyle(t *testing.T) {
	f := &frameimpl{font: mockFont(), defaultfontheight: 13, rect: image.Rect(0, 0, 500, 100), maxlines: 7, maxtab: 40}
	_, _, nf := f.bxscan([]byte("abcdef"), []uint8{0, 0, 1, 1, 0, 0}, 0, 0)
	wantBoxes(t, "style boundaries", nf, "ab/0", "cd/1", "ef/0")

	_, _, nf = f.bxscan([]byte("ab\tcd\nef"), []uint8{1, 1, 1, 2, 2, 2, 0, 0}, 0, 0)
	wantBoxes(t, "tabs and newlines carry the style", nf, "ab/1", "\t/1", "cd/2", "\n/2", "ef/0")

	_, _, nf = f.bxscan([]byte("abc"), nil, 0, 0)
	wantBoxes(t, "nil styles", nf, "abc/0")

	_, _, nf = f.bxscan([]byte("abcd"), []uint8{3}, 0, 0)
	wantBoxes(t, "short styles pad with 0", nf, "a/3", "bcd/0")
}

func TestCleanKeepsStyleBoundaries(t *testing.T) {
	f := &frameimpl{font: mockFont(), defaultfontheight: 13, rect: image.Rect(0, 0, 500, 100), maxlines: 7, maxtab: 40}
	ab := makeBox("ab")
	cd := makeBox("cd")
	cd.Style = 1
	ef := makeBox("ef")
	ef.Style = 1
	f.box = []*frbox{ab, cd, ef}
	f.clean(f.rect.Min, 0, 3)
	wantBoxes(t, "clean", f, "ab/0", "cdef/1")
}

// styledFrame builds a frame on a mock display with a two-entry style
// table: 1 is Medblue text, 2 is Purpleblue text on a Palegreygreen background,
// underlined. Named mock images, so that draw ops mention the colour.
func styledFrame(t *testing.T) (Frame, draw.Display) {
	t.Helper()
	iv := &invariants{topcorner: image.Pt(20, 10), textarea: image.Rect(20, 10, 400, 100)}
	fr := setupFrame(t, iv)
	display := fr.(*frameimpl).display
	one := image.Rect(0, 0, 1, 1)
	medblue := edwoodtest.NewImage(display, "Medblue", one)
	purpleblue := edwoodtest.NewImage(display, "Purpleblue", one)
	palegreygreen := edwoodtest.NewImage(display, "Palegreygreen", one)
	fr.SetStyleTable([]StyleColours{
		{},
		{Text: medblue},
		{Text: purpleblue, Back: palegreygreen, Underline: true},
	})
	return fr, display
}

func opsContaining(ops []string, subs ...string) []string {
	var out []string
	for _, op := range ops {
		ok := true
		for _, s := range subs {
			if !strings.Contains(op, s) {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, op)
		}
	}
	return out
}

func TestInsertStyledDrawsStyleColours(t *testing.T) {
	fr, display := styledFrame(t)
	gdo := display.(edwoodtest.GettableDrawOps)
	gdo.Clear()

	fr.InsertStyled([]rune("abcd"), []uint8{0, 1, 1, 2}, 0)
	f := fr.(*frameimpl)
	wantBoxes(t, "after InsertStyled", f, "a/0", "bc/1", "d/2")

	ops := gdo.DrawOps()
	if n := len(opsContaining(ops, `string "bc"`, "Medblue")); n != 1 {
		t.Errorf("want one red draw of \"bc\", got %d in %q", n, ops)
	}
	if n := len(opsContaining(ops, `string "a"`, "black")); n != 1 {
		t.Errorf("want one black draw of \"a\", got %d in %q", n, ops)
	}
	if n := len(opsContaining(ops, `string "d"`, "Purpleblue")); n != 1 {
		t.Errorf("want one blue draw of \"d\", got %d in %q", n, ops)
	}
	// Style 2 has a green background box and a 1px underline (height 10
	// font: y 19..20) under "d", which is the 4th 13px glyph from x=20.
	if n := len(opsContaining(ops, "fill (59,10)-(72,20)", "Palegreygreen")); n == 0 {
		// fills print without colour in the short form; accept either form
		if n2 := len(opsContaining(ops, "(59,10)-(72,20)")); n2 == 0 {
			t.Errorf("want a background fill for \"d\" at (59,10)-(72,20) in %q", ops)
		}
	}
	if n := len(opsContaining(ops, "(59,19)-(72,20)")); n == 0 {
		t.Errorf("want an underline at (59,19)-(72,20) in %q", ops)
	}
}

func TestRestyle(t *testing.T) {
	fr, display := styledFrame(t)
	gdo := display.(edwoodtest.GettableDrawOps)
	fr.Insert([]rune("abcdef"), 0)
	f := fr.(*frameimpl)
	wantBoxes(t, "plain insert", f, "abcdef/0")

	gdo.Clear()
	fr.Restyle(1, 4, []uint8{1, 1, 2})
	wantBoxes(t, "restyle middle", f, "a/0", "bc/1", "d/2", "ef/0")
	ops := gdo.DrawOps()
	if n := len(opsContaining(ops, `string "bc"`, "Medblue")); n != 1 {
		t.Errorf("Restyle: want one red draw of \"bc\", got %d in %q", n, ops)
	}
	if n := len(opsContaining(ops, `string "d"`, "Purpleblue")); n != 1 {
		t.Errorf("Restyle: want one blue draw of \"d\", got %d in %q", n, ops)
	}
	if n := len(opsContaining(ops, `string "a"`)); n != 0 {
		t.Errorf("Restyle: \"a\" outside the range was repainted: %q", ops)
	}

	// Restyling to the same styles is a no-op.
	gdo.Clear()
	fr.Restyle(1, 4, []uint8{1, 1, 2})
	if ops := gdo.DrawOps(); len(ops) != 0 {
		t.Errorf("Restyle with unchanged styles drew %q", ops)
	}

	// Back to plain: boxes keep their boundaries until clean merges them,
	// but all are style 0 and drawn black.
	gdo.Clear()
	fr.Restyle(0, 6, nil)
	for _, b := range f.box {
		if b.Style != 0 {
			t.Errorf("box %q still has style %d after Restyle to nil", b.Ptr, b.Style)
		}
	}
	if n := len(opsContaining(gdo.DrawOps(), "Medblue")); n != 0 {
		t.Errorf("red remains after restyle to plain: %q", gdo.DrawOps())
	}

	// Out-of-range and empty ranges are harmless.
	fr.Restyle(5, 50, []uint8{1})
	fr.Restyle(3, 3, []uint8{1})
	wantBoxes(t, "clamped restyle", f, "a/0", "bc/0", "d/0", "e/0", "f/1")
}

func TestRestyleUnderSelection(t *testing.T) {
	fr, display := styledFrame(t)
	gdo := display.(edwoodtest.GettableDrawOps)
	fr.Insert([]rune("abcdef"), 0)
	// Highlight "bcd".
	fr.DrawSel(fr.Ptofchar(1), 1, 4, true)
	gdo.Clear()
	fr.Restyle(0, 6, []uint8{1, 1, 1, 1, 1, 1})
	ops := gdo.DrawOps()
	// The whole run is one box and is painted in the style colour first;
	// the selected part is then repainted in the highlight colours.
	if n := len(opsContaining(ops, `string "abcdef"`, "Medblue")); n != 1 {
		t.Errorf("text not drawn in the style colour: %q", ops)
	}
	if n := len(opsContaining(ops, `string "bcd"`, "black")); n != 1 {
		t.Errorf("selected text not repainted in highlight colours: %q", ops)
	}
	if n := len(opsContaining(ops, `string "bcd"`, "Medblue")); n != 0 {
		t.Errorf("selected text drawn in style colour: %q", ops)
	}
	f := fr.(*frameimpl)
	if f.sp0 != 1 || f.sp1 != 4 || !f.highlighton {
		t.Errorf("selection lost by Restyle: %d,%d highlight=%v", f.sp0, f.sp1, f.highlighton)
	}
}

func TestInsertStyledThroughProxy(t *testing.T) {
	fr, _ := styledFrame(t)
	f := fr.(*frameimpl)
	up := (*selectscrollupdaterimpl)(f)
	up.InsertStyled([]rune("xy"), []uint8{2, 2}, 0)
	wantBoxes(t, "proxy", f, "xy/2")
}
