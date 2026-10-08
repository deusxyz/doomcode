package main

import (
	"github.com/deusxyz/doomcode/editor/draw"
	"github.com/deusxyz/doomcode/editor/frame"
	"github.com/deusxyz/doomcode/editor/theme"
)

// Styles on screen: the body's file.StyleTable names spans, the theme's
// StyleSet turns names into frame indices, and the frame paints them.
// Tags, column tags and the row tag never carry styles.
//
// See docs/05-style-spec.md in the justcode project.

// styleSet returns the display's style set, creating it on first use.
func (g *globals) styleSet(display draw.Display) *theme.StyleSet {
	if g.styles == nil {
		g.styles = theme.NewStyleSet(display, g.palette.Styles)
	}
	return g.styles
}

// hasStyles reports whether t is a body with any styled spans.
func (t *Text) hasStyles() bool {
	return t.what == Body && t.file != nil && t.file.Styles().Len() > 0
}

// styleIndices returns one frame style index per rune of [q0,q0+n) in t's
// file, or nil when the range has no styles (so plain Insert can be
// used). It also makes sure fr has the current style table.
func (t *Text) styleIndices(fr frame.SelectScrollUpdater, q0, n int) []uint8 {
	if n <= 0 || !t.hasStyles() {
		return nil
	}
	spans := t.file.Styles().Range(q0, q0+n)
	if len(spans) == 0 {
		return nil
	}
	ss := global.styleSet(t.display)
	idx := make([]uint8, n)
	for _, sp := range spans {
		i := ss.Index(sp.Name)
		for q := sp.Q0; q < sp.Q1; q++ {
			idx[q-q0] = i
		}
	}
	fr.SetStyleTable(ss.Table())
	return idx
}

// Restyle repaints the visible part of [q0,q1) of t's file in the styles
// now in its StyleTable. Writers of the style file call it on every Text
// showing the file.
func (t *Text) Restyle(q0, q1 int) {
	if t.what != Body || t.fr == nil {
		return
	}
	nchars := t.fr.GetFrameFillStatus().Nchars
	if q0 < t.org {
		q0 = t.org
	}
	if end := t.org + nchars; q1 > end {
		q1 = end
	}
	if q0 >= q1 {
		return
	}
	idx := t.styleIndices(t.fr, q0, q1-q0)
	if idx == nil {
		idx = make([]uint8, q1-q0) // everything plain now
		if global.styles != nil {
			t.fr.SetStyleTable(global.styles.Table())
		}
	}
	t.fr.Restyle(q0-t.org, q1-t.org, idx)
}
