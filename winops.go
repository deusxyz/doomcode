package main

import (
	"fmt"
	"image"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Window operations reached through the Ctrl-B prefix that the mouse
// does with the layout box or by dragging: moving windows, showing their
// numbers, shrinking, listing them; and find-next.
//
// See docs/03-keyboard-spec.md section 5.2 in the justcode project.

// keyFindNext searches again: for the selection if there is one, else for
// the text of the last successful search.
func (t *Text) keyFindNext() {
	t.TypeCommit()
	t.dropAnchor()
	if t.q0 != t.q1 {
		r := make([]rune, t.q1-t.q0)
		t.file.Read(t.q0, r)
		search(t, r)
		return
	}
	if len(global.lastsearch) > 0 {
		search(t, append([]rune(nil), global.lastsearch...))
	}
}

// moveWindow moves the focused window up or down within its column.
func (g *globals) moveWindow(t *Text, dir int) {
	w, c := focusWindow(t)
	if w == nil || c == nil {
		return
	}
	if c.MoveWindow(w, dir) {
		g.warpTo(&w.body)
	}
}

// moveWindowColumn moves the focused window to the neighbouring column,
// at the same height.
func (g *globals) moveWindowColumn(t *Text, dir int) {
	w, c := focusWindow(t)
	if w == nil || c == nil {
		return
	}
	i := g.columnIndex(c) + dir
	if i < 0 || i >= len(g.row.col) || g.columnIndex(c) < 0 {
		return
	}
	nc := g.row.col[i]
	y := w.r.Min.Y
	c.Close(w, false)
	nc.Add(w, nil, y)
	g.setFocus(&w.body)
}

// shrinkFocus makes the focused window smaller by growing the window
// below it (or above it, for the last one) a little.
func (g *globals) shrinkFocus(t *Text) {
	w, c := focusWindow(t)
	if w == nil || c == nil || len(c.w) < 2 {
		return
	}
	i := -1
	for j, cw := range c.w {
		if cw == w {
			i = j
		}
	}
	if i < 0 {
		return
	}
	other := i + 1
	if other >= len(c.w) {
		other = i - 1
	}
	c.Grow(c.w[other], 1)
	g.warpTo(&w.body)
}

// showNumbers draws each window's number (1 from the top of its column)
// over its layout box for a moment, then restores the boxes.
func (g *globals) showNumbers() {
	display := g.row.display
	if display == nil {
		return
	}
	font := fontget(g.tagfont, display)
	for _, c := range g.row.col {
		for i, w := range c.w {
			pt := w.tag.scrollr.Min.Add(image.Pt(display.ScaleSize(2), 0))
			display.ScreenImage().Bytes(pt, g.palette.TagText(), image.Point{}, font, []byte(strconv.Itoa(i+1)))
		}
	}
	display.Flush()
	time.AfterFunc(1500*time.Millisecond, func() {
		g.row.lk.Lock()
		defer g.row.lk.Unlock()
		for _, c := range g.row.col {
			for _, w := range c.w {
				w.DrawButton()
			}
		}
		display.Flush()
	})
}

// windowListName is the name of the window listing all windows.
func (g *globals) windowListName() string {
	return filepath.Join(g.wdir, "+windows")
}

// windowListText is the body of the +windows window: one line per
// window, column by column, "id<tab>name", so that Look on a name jumps
// to that window.
func (g *globals) windowListText(skip *Window) string {
	var sb strings.Builder
	for ci, c := range g.row.col {
		if ci > 0 {
			sb.WriteByte('\n')
		}
		for _, w := range c.w {
			if w == skip {
				continue
			}
			fmt.Fprintf(&sb, "%d\t%s\n", w.id, w.body.file.Name())
		}
	}
	return sb.String()
}

// windowList opens (or reuses) the +windows window, fills it and moves
// the focus there.
func (g *globals) windowList(t *Text) {
	name := g.windowListName()
	w := lookfile(name)
	if w == nil {
		if len(g.row.col) == 0 {
			return
		}
		c := g.row.col[len(g.row.col)-1]
		if _, fc := focusWindow(t); fc != nil {
			c = fc
		}
		w = c.Add(nil, nil, -1)
		w.filemenu = false
		w.SetName(name)
		xfidlog(w, "new")
	}
	body := &w.body
	body.Delete(0, body.Nc(), true)
	body.Insert(0, []rune(g.windowListText(w)), true)
	body.Show(0, 0, true)
	if body.fr != nil {
		body.ScrDraw(body.fr.GetFrameFillStatus().Nchars)
	}
	body.file.TreatAsClean()
	g.setFocus(body)
}
