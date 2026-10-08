package main

import (
	"image"
	"os"
	"runtime"

	"github.com/deusxyz/doomcode/editor/draw"
	"github.com/deusxyz/doomcode/editor/frame"
	"github.com/deusxyz/doomcode/editor/theme"
)

// Switching the palette while running: "Theme NAME", "Theme reload" and,
// later, following the system's light or dark mode. Every frame copied
// its colours when it was made, so they are handed the new ones and the
// whole row is laid out again, as after a window resize.
//
// See docs/09-config-spec.md section 5.1 in the doomcode repository.

// paletteFonts records whether the fonts follow the palette: true when
// neither a flag nor the config file chose them (set at startup).
var paletteFonts struct{ varFont, fixedFont bool }

// namedPalette returns the built-in palette name with the theme file
// applied on top, as at startup.
func namedPalette(name string) (theme.Palette, bool) {
	p, ok := theme.PaletteByName(name)
	if !ok {
		return theme.Palette{}, false
	}
	return startupPalette(p, false, name, true, themeFilePath()), true
}

// switchPalette makes the named palette current and repaints. It reports
// false for an unknown name.
func (g *globals) switchPalette(name string) bool {
	p, ok := namedPalette(name)
	if !ok {
		return false
	}
	*paletteName = name
	display := g.row.display
	g.palette = p
	g.styles = nil // style indices are allocated against the new styles
	g.iconinit(display)
	g.switchFonts(p)

	tagCols := g.palette.Tag.Colors(display)
	textCols := g.palette.Text.Colors(display)
	recolor := func(t *Text, cols [frame.NumColours]draw.Image) {
		if t.fr != nil {
			t.fr.Init(t.fr.Rect(), frame.OptColors(cols))
		}
	}
	recolor(&g.row.tag, tagCols)
	for _, c := range g.row.col {
		recolor(&c.tag, tagCols)
		for _, w := range c.w {
			recolor(&w.tag, tagCols)
			recolor(&w.body, textCols)
		}
	}
	screen := display.ScreenImage()
	screen.Draw(screen.R(), g.palette.TextBack(), nil, image.Point{})
	g.row.Resize(g.row.r)
	display.Flush()
	return true
}

// switchFonts moves every text to the palette's fonts where the fonts
// follow the palette (macOS, nothing set by flag or config).
func (g *globals) switchFonts(p theme.Palette) {
	if runtime.GOOS != "darwin" {
		return
	}
	oldVar, oldFixed := *varfontflag, *fixedfontflag
	newVar, newFixed := oldVar, oldFixed
	if paletteFonts.varFont && p.VarFont != "" {
		newVar = p.VarFont
	}
	if paletteFonts.fixedFont && p.FixedFont != "" {
		newFixed = p.FixedFont
	}
	if newVar == oldVar && newFixed == oldFixed {
		return
	}
	display := g.row.display
	if fontget(newVar, display) == nil || fontget(newFixed, display) == nil {
		return // fontget has warned; keep what works
	}
	*varfontflag, *fixedfontflag = newVar, newFixed
	g.tagfont = newVar
	os.Setenv("font", newVar)
	refont := func(t *Text) {
		var name string
		switch t.font {
		case oldVar:
			name = newVar
		case oldFixed:
			name = newFixed
		default:
			return // a font chosen with Font in this window stays
		}
		t.font = name
		if t.fr != nil {
			t.fr.Init(t.fr.Rect(), frame.OptFont(fontget(name, display)))
		}
		if t.w != nil && t.what == Body && t.file.IsDir() {
			f := fontget(name, display)
			for i, d := range t.w.dirnames {
				t.w.widths[i] = f.StringWidth(d)
			}
			t.all.Min.X++ // force recolumnation, as Font does
		}
	}
	refont(&g.row.tag)
	for _, c := range g.row.col {
		refont(&c.tag)
		for _, w := range c.w {
			refont(&w.tag)
			refont(&w.body)
		}
	}
}
