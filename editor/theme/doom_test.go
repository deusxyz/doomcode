package theme

import (
	"math"
	"testing"

	"github.com/deusxyz/doomcode/editor/draw"
)

// contrast is the WCAG 2 contrast ratio of two opaque colours.
func contrast(a, b draw.Color) float64 {
	lum := func(c draw.Color) float64 {
		ch := func(v uint32) float64 {
			x := float64(v&0xFF) / 255
			if x <= 0.03928 {
				return x / 12.92
			}
			return math.Pow((x+0.055)/1.055, 2.4)
		}
		return 0.2126*ch(uint32(c)>>24) + 0.7152*ch(uint32(c)>>16) + 0.0722*ch(uint32(c)>>8)
	}
	la, lb := lum(a), lum(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// TestDoomPalettesContrast holds doom-light and doom-dark to the ranges
// of docs/10-theme-style-guide.md.
func TestDoomPalettesContrast(t *testing.T) {
	for name, p := range map[string]Palette{"doom-light": DoomLight, "doom-dark": DoomDark} {
		bg := p.Text.Back.Color
		check := func(what string, c draw.Color, lo, hi float64) {
			t.Helper()
			if r := contrast(c, bg); r < lo || r > hi {
				t.Errorf("%s: %s contrast %.2f, want %.1f–%.1f", name, what, r, lo, hi)
			}
		}
		check("body text", p.Text.Text.Color, 7, 12)
		if r := contrast(p.Tag.Text.Color, p.Tag.Back.Color); r < 7 || r > 12 {
			t.Errorf("%s: tag text contrast %.2f, want 7–12", name, r)
		}
		check("selection", p.Text.High.Color, 1.1, 1.8)
		check("comment", p.Styles["comment"].Fg.Color, 4.5, 6.5)
		for _, s := range []string{"keyword", "string", "number", "constant", "type", "function", "preproc", "heading", "link"} {
			check(s, p.Styles[s].Fg.Color, 4.5, 9)
		}
		for _, s := range []string{"error", "warning", "info", "hint"} {
			st := p.Styles[s]
			if r := contrast(p.Text.Text.Color, st.Bg.Color); r < 7 {
				t.Errorf("%s: text on %s background contrast %.2f, want ≥ 7", name, s, r)
			}
		}
	}
}

func TestDefaultPalette(t *testing.T) {
	if DefaultPaletteName != "doom-light" {
		t.Errorf("default palette %q", DefaultPaletteName)
	}
	for _, n := range []string{"doom-light", "doom-dark", "acme", "vampira", "solarizedlight", "solarizeddark"} {
		if _, ok := PaletteByName(n); !ok {
			t.Errorf("palette %q missing", n)
		}
	}
}
