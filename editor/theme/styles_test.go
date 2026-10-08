package theme

import (
	"image"
	"testing"

	"github.com/deusxyz/doomcode/editor/edwoodtest"
)

func TestStylesResolve(t *testing.T) {
	st := Styles{"keyword": fg(1), "keyword.control": fg(2), "diff.add": bg(3)}
	for _, tc := range []struct {
		name string
		want uint32
		ok   bool
	}{
		{"keyword", 1, true},
		{"keyword.control", 2, true},
		{"keyword.other", 1, true},        // falls back to keyword
		{"keyword.control.flow", 2, true}, // two levels
		{"diff.add", 0, true},             // bg only: Fg is zero
		{"diff", 0, false},
		{"", 0, false},
		{"nothing.here", 0, false},
	} {
		s, ok := st.Resolve(tc.name)
		if ok != tc.ok || uint32(s.Fg.Color) != tc.want {
			t.Errorf("Resolve(%q) = %v,%v; want fg %d,%v", tc.name, s, ok, tc.want, tc.ok)
		}
	}
}

func TestAllPalettesDefineTheStandardNames(t *testing.T) {
	standard := []string{"comment", "keyword", "string", "number", "type", "function", "constant", "preproc",
		"heading", "link", "error", "warning", "info", "hint", "match", "diff.add", "diff.del", "diff.change"}
	for name, p := range palettes {
		if p.Styles == nil {
			t.Errorf("palette %q has no styles", name)
			continue
		}
		for _, n := range standard {
			if _, ok := p.Styles.Resolve(n); !ok {
				t.Errorf("palette %q lacks style %q", name, n)
			}
		}
		for _, n := range []string{"error", "warning", "info", "hint"} {
			s, _ := p.Styles.Resolve(n)
			if !s.Underline || s.Bg.Color == 0 || s.Line.Color == 0 {
				t.Errorf("palette %q: diagnostic %q should have a background, an underline and a line colour: %+v", name, n, s)
			}
		}
	}
}

func TestStyleSetIndices(t *testing.T) {
	display := edwoodtest.NewDisplay(image.Rect(0, 0, 100, 100))
	ss := NewStyleSet(display, Light.Styles)

	if ss.Len() != 1 || ss.Index("") != 0 {
		t.Fatalf("fresh set: len %d, index of \"\" %d", ss.Len(), ss.Index(""))
	}
	k := ss.Index("keyword")
	c := ss.Index("comment")
	if k != 1 || c != 2 || ss.Index("keyword") != 1 {
		t.Errorf("indices: keyword %d comment %d, keyword again %d", k, c, ss.Index("keyword"))
	}
	if ss.Name(k) != "keyword" || ss.Name(0) != "" || ss.Name(200) != "" {
		t.Errorf("Name: %q %q %q", ss.Name(k), ss.Name(0), ss.Name(200))
	}

	// Unknown names get an index and plain colours.
	u := ss.Index("no.such.style")
	tab := ss.Table()
	if u != 3 || len(tab) != 4 {
		t.Fatalf("unknown name: index %d, table len %d", u, len(tab))
	}
	if tab[u].Text != nil || tab[u].Back != nil || tab[u].Underline {
		t.Errorf("unknown style is not plain: %+v", tab[u])
	}
	if tab[k].Text == nil || tab[k].Back != nil {
		t.Errorf("keyword should have a text colour only: %+v", tab[k])
	}
	e := ss.Index("error")
	tab = ss.Table()
	if tab[e].Back == nil || !tab[e].Underline || tab[e].Line == nil {
		t.Errorf("error should have background, underline and line colour: %+v", tab[e])
	}

	// Dotted fallback shares the parent's look but gets its own index.
	kc := ss.Index("keyword.control")
	if kc == k {
		t.Errorf("keyword.control got the same index as keyword")
	}
	if ss.Table()[kc].Text == nil {
		t.Errorf("keyword.control did not inherit keyword's colour")
	}
}

func TestStyleSetExhaustion(t *testing.T) {
	ss := NewStyleSet(nil, Light.Styles)
	for i := 1; i < 256; i++ {
		if got := ss.Index(string(rune('A'+i%26)) + string(rune(i))); got == 0 {
			t.Fatalf("index %d came back 0", i)
		}
	}
	if ss.Len() != 256 {
		t.Fatalf("len %d; want 256", ss.Len())
	}
	if got := ss.Index("one too many"); got != 0 {
		t.Errorf("256th new name should fall back to 0, got %d", got)
	}
}
