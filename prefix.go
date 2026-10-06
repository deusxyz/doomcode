package main

import (
	"image"
	"strconv"

	"github.com/rjkroege/edwood/draw"
)

// The Ctrl-B prefix, in the manner of tmux: Ctrl-B followed by one key runs
// an action from the prefix keymap on the focused text. Esc or an unbound
// key cancels. There is no timeout. Unlike tmux, Ctrl-B Ctrl-B does not type
// a literal Ctrl-B: devdraw delivers key auto-repeat as plain keystrokes, so
// holding Ctrl-B a little too long would otherwise insert control characters
// and leave the prefix in a random state. A repeated Ctrl-B keeps the prefix
// armed.
//
// Keyboard focus is explicit: the last text clicked or typed into, which is
// what globals.barttext already records for the -b flag. Prefix navigation
// moves the focus and warps the mouse pointer there, so that the mouse
// chords and the "typing goes under the mouse" rule keep working.
//
// See docs/03-keyboard-spec.md section 5.2 in the justcode project.

// prefixKey is the rune devdraw delivers for Ctrl-B.
const prefixKey = 0x02

// prefixState is the two-state machine fed by keyboardthread.
type prefixState struct {
	armed bool
}

// prefixResult says what keyboardthread should do with a rune.
type prefixResult int

const (
	prefixPass   prefixResult = iota // not for us: deliver r as ordinary typing
	prefixArmed                      // r was Ctrl-B: wait for the next key
	prefixCancel                     // the prefix was dropped without an action
	prefixRun                        // run the returned action on the focus
)

// feed advances the machine with r and reports what to do.
func (p *prefixState) feed(r rune, km Keymap) (prefixResult, *Action) {
	if !p.armed {
		if r == prefixKey {
			p.armed = true
			return prefixArmed, nil
		}
		return prefixPass, nil
	}
	switch r {
	case prefixKey: // auto-repeat of the prefix key: stay armed
		return prefixArmed, nil
	case 0x1b: // Esc
		p.armed = false
		return prefixCancel, nil
	}
	p.armed = false
	if a := km.Lookup(r); a != nil {
		return prefixRun, a
	}
	return prefixCancel, nil
}

// prefixcursor is shown while the prefix is armed: a hollow diamond, as
// distinct from the box cursor used for dragging.
var prefixcursor = draw.Cursor{
	Point: image.Point{-7, -7},
	Clr: [32]byte{0x01, 0x80, 0x03, 0xC0, 0x07, 0xE0, 0x0F, 0xF0,
		0x1F, 0xF8, 0x3F, 0xFC, 0x7F, 0xFE, 0xFF, 0xFF,
		0xFF, 0xFF, 0x7F, 0xFE, 0x3F, 0xFC, 0x1F, 0xF8,
		0x0F, 0xF0, 0x07, 0xE0, 0x03, 0xC0, 0x01, 0x80},
	Set: [32]byte{0x00, 0x00, 0x01, 0x80, 0x03, 0xC0, 0x06, 0x60,
		0x0C, 0x30, 0x18, 0x18, 0x30, 0x0C, 0x60, 0x06,
		0x60, 0x06, 0x30, 0x0C, 0x18, 0x18, 0x0C, 0x30,
		0x06, 0x60, 0x03, 0xC0, 0x01, 0x80, 0x00, 0x00},
}

// defaultPrefixBindings follow tmux where tmux has the same idea.
var defaultPrefixBindings = []struct{ key, action string }{
	{"Left", "focus-left"},
	{"Right", "focus-right"},
	{"Up", "focus-up"},
	{"Down", "focus-down"},
	{"o", "focus-next"},
	{";", "focus-prev"},
	{"0", "focus-0"},
	{"1", "focus-1"},
	{"2", "focus-2"},
	{"3", "focus-3"},
	{"4", "focus-4"},
	{"5", "focus-5"},
	{"6", "focus-6"},
	{"7", "focus-7"},
	{"8", "focus-8"},
	{"9", "focus-9"},
	{"t", "tag-toggle"},
	{":", "command-line"},
	{"c", "new"},
	{"%", "newcol"},
	{"x", "del"},
	{"&", "delcol"},
	{"z", "zoom"},
	{"Z", "maximize"},
	{"+", "grow"},
	{"s", "sort"},
	{"S", "putall"},
	{"g", "file-start"},
	{"G", "file-end"},
	{"Enter", "execute"},
	{"/", "comment-toggle"},
	{"l", "look"},
	{"Space", "anchor"},
	{"b", "select-block"},
	{"Tab", "outdent"},
	{"n", "find-next"},
	{"w", "window-list"},
	{"{", "move-up"},
	{"}", "move-down"},
	{"[", "move-left"},
	{"]", "move-right"},
	{"q", "show-numbers"},
	{"-", "shrink"},
}

// DefaultPrefixKeymap returns a fresh copy of the built-in prefix bindings.
func DefaultPrefixKeymap() Keymap {
	km := make(Keymap, len(defaultPrefixBindings))
	for _, b := range defaultPrefixBindings {
		if err := km.Bind(b.key, b.action); err != nil {
			panic("default prefix keymap: " + err.Error())
		}
	}
	return km
}

// prefixActions are the window and focus actions. They are Row actions:
// they tolerate a nil focus text. Text actions from keys.go (file-start,
// execute, ...) may be bound under the prefix as well.
func prefixActions() []*Action {
	as := []*Action{
		{Name: "focus-left", Doc: "focus the window in the column to the left nearest to this one", Row: true, Fn: func(t *Text) { global.focusColumn(t, -1) }},
		{Name: "focus-right", Doc: "focus the window in the column to the right nearest to this one", Row: true, Fn: func(t *Text) { global.focusColumn(t, 1) }},
		{Name: "focus-up", Doc: "focus the window above in this column", Row: true, Fn: func(t *Text) { global.focusWindowInColumn(t, -1) }},
		{Name: "focus-down", Doc: "focus the window below in this column", Row: true, Fn: func(t *Text) { global.focusWindowInColumn(t, 1) }},
		{Name: "focus-next", Doc: "focus the next window, column by column, wrapping around", Row: true, Fn: func(t *Text) { global.focusNext(t) }},
		{Name: "focus-prev", Doc: "focus the previously focused window", Row: true, Fn: func(t *Text) { global.focusPrev() }},
		{Name: "tag-toggle", Doc: "move the focus between the tag and the body of the window", Row: true, Fn: func(t *Text) { global.focusTagToggle(t) }},
		{Name: "command-line", Doc: "put the cursor at the end of the window's tag, ready to type a command", Row: true, Fn: func(t *Text) { global.focusCommandLine(t) }},
		{Name: "new", Doc: "New: a new window in this column", Row: true, Fn: func(t *Text) {
			if t != nil {
				newx(t, nil, nil, false, false, "")
			}
		}},
		{Name: "newcol", Doc: "Newcol: a new column", Row: true, Fn: func(t *Text) {
			if t == nil {
				t = &global.row.tag
			}
			if t.row == nil {
				t.row = &global.row
			}
			newcol(t, nil, nil, false, false, "")
		}},
		{Name: "del", Doc: "Del: close the window if it is clean; a second Del closes a dirty one", Row: true, Fn: func(t *Text) {
			if t != nil && t.w != nil {
				del(t, nil, nil, false, false, "")
			}
		}},
		{Name: "delcol", Doc: "Delcol: close the column if all its windows are clean", Row: true, Fn: func(t *Text) {
			if t != nil && t.col != nil {
				delcol(t, nil, nil, false, false, "")
			}
		}},
		{Name: "zoom", Doc: "the window takes the whole column, like button 3 in the layout box", Row: true, Fn: func(t *Text) { global.growFocus(t, 3) }},
		{Name: "maximize", Doc: "the window grows as much as it can leaving other tags visible, like button 2 in the layout box", Row: true, Fn: func(t *Text) { global.growFocus(t, 2) }},
		{Name: "grow", Doc: "the window grows a little, like button 1 in the layout box", Row: true, Fn: func(t *Text) { global.growFocus(t, 1) }},
		{Name: "sort", Doc: "Sort: order the windows of the column by name", Row: true, Fn: func(t *Text) {
			if t != nil {
				sortx(t, nil, nil, false, false, "")
			}
		}},
		{Name: "putall", Doc: "Putall: write all dirty windows that name existing files", Row: true, Fn: func(t *Text) { putall(t, nil, nil, false, false, "") }},
		{Name: "find-next", Doc: "search again: for the selection, or for the last thing searched", Fn: (*Text).keyFindNext},
		{Name: "window-list", Doc: "open a +windows window listing every window; Look (^O) on a name jumps there", Row: true, Fn: func(t *Text) { global.windowList(t) }},
		{Name: "move-up", Doc: "move the window one place up in its column", Row: true, Fn: func(t *Text) { global.moveWindow(t, -1) }},
		{Name: "move-down", Doc: "move the window one place down in its column", Row: true, Fn: func(t *Text) { global.moveWindow(t, 1) }},
		{Name: "move-left", Doc: "move the window to the column on the left", Row: true, Fn: func(t *Text) { global.moveWindowColumn(t, -1) }},
		{Name: "move-right", Doc: "move the window to the column on the right", Row: true, Fn: func(t *Text) { global.moveWindowColumn(t, 1) }},
		{Name: "show-numbers", Doc: "show each window's number in its layout box for a moment (focus-N uses them)", Row: true, Fn: func(t *Text) { global.showNumbers() }},
		{Name: "shrink", Doc: "make the window a little smaller by growing its neighbour", Row: true, Fn: func(t *Text) { global.shrinkFocus(t) }},
		{Name: "file-start", Doc: "move to the start of the text", Fn: func(t *Text) { t.TypeCommit(); t.Show(0, 0, true) }},
		{Name: "file-end", Doc: "move to the end of the text", Fn: func(t *Text) { t.TypeCommit(); t.Show(t.file.Nr(), t.file.Nr(), true) }},
	}
	for n := 0; n <= 9; n++ {
		n := n
		doc := "focus window " + strconv.Itoa(n) + " of this column, counting from the top"
		if n == 0 {
			doc = "focus the tag of this column"
		}
		as = append(as, &Action{Name: "focus-" + strconv.Itoa(n), Doc: doc, Row: true, Fn: func(t *Text) { global.focusNumbered(t, n) }})
	}
	return as
}

// focusText returns the text keyboard commands act on: the last text
// clicked or typed into if it is still on screen, else the text under the
// mouse.
func (g *globals) focusText() *Text {
	if t := g.barttext; t != nil && g.onScreen(t) {
		return t
	}
	if g.mouse != nil {
		return g.row.Which(g.mouse.Point)
	}
	return nil
}

// onScreen reports whether t still belongs to a live window or column.
func (g *globals) onScreen(t *Text) bool {
	switch {
	case t.w != nil:
		return t.w.col != nil
	case t.what == Rowtag:
		return true
	case t.col != nil:
		for _, c := range g.row.col {
			if c == t.col {
				return true
			}
		}
	}
	return false
}

// setFocus makes t the keyboard focus and moves the mouse pointer onto it.
func (g *globals) setFocus(t *Text) {
	if t == nil {
		return
	}
	if g.barttext != nil && g.barttext != t {
		g.prevfocus = g.barttext
	}
	g.barttext = t
	g.focusSticky = true
	g.warpTo(t)
}

// warpTo moves the mouse pointer to the start of the first visible line of
// t, where Acme itself puts it after a button-3 jump.
func (g *globals) warpTo(t *Text) {
	if t.display == nil || t.fr == nil {
		return
	}
	r := t.fr.Rect()
	h := t.fr.DefaultFontHeight()
	if h == 0 {
		h = 12
	}
	g.lastWarp = r.Min.Add(image.Pt(4, h-4))
	t.display.MoveTo(g.lastWarp)
}

// focusWindow returns the window and column of t, either of which may be
// nil: a column tag has a column but no window, the row tag has neither.
func focusWindow(t *Text) (*Window, *Column) {
	if t == nil {
		return nil, nil
	}
	return t.w, t.col
}

// centerY is the vertical middle of w on screen.
func centerY(w *Window) int {
	return (w.r.Min.Y + w.r.Max.Y) / 2
}

// nearestWindow returns the window of c whose vertical span contains y, or
// failing that the one whose middle is closest to y. nil if c is empty.
func nearestWindow(c *Column, y int) *Window {
	var best *Window
	bestd := 0
	for _, w := range c.w {
		if w.r.Min.Y <= y && y < w.r.Max.Y {
			return w
		}
		d := centerY(w) - y
		if d < 0 {
			d = -d
		}
		if best == nil || d < bestd {
			best, bestd = w, d
		}
	}
	return best
}

// columnIndex returns the position of c in the row, or -1.
func (g *globals) columnIndex(c *Column) int {
	for i, cc := range g.row.col {
		if cc == c {
			return i
		}
	}
	return -1
}

// focusColumn moves the focus dir columns to the left (-1) or right (+1),
// into the window nearest to the current one vertically, or into the tag
// of an empty column. From the row tag it goes to the first or last column.
func (g *globals) focusColumn(t *Text, dir int) {
	_, c := focusWindow(t)
	i := g.columnIndex(c)
	switch {
	case len(g.row.col) == 0:
		return
	case i < 0 && dir < 0:
		i = len(g.row.col)
	case i < 0:
		i = -1
	}
	i += dir
	if i < 0 || i >= len(g.row.col) {
		return
	}
	target := g.row.col[i]
	y := 0
	if w, _ := focusWindow(t); w != nil {
		y = centerY(w)
	} else if c != nil {
		y = c.r.Min.Y
	}
	if w := nearestWindow(target, y); w != nil {
		g.setFocus(&w.body)
		return
	}
	g.setFocus(&target.tag)
}

// focusWindowInColumn moves the focus to the window above (-1) or below
// (+1) in the same column; from the column tag, down goes to the first
// window.
func (g *globals) focusWindowInColumn(t *Text, dir int) {
	w, c := focusWindow(t)
	if c == nil || len(c.w) == 0 {
		return
	}
	i := -1
	for j, cw := range c.w {
		if cw == w {
			i = j
		}
	}
	i += dir
	if i < 0 {
		if w != nil {
			g.setFocus(&c.tag)
		}
		return
	}
	if i >= len(c.w) {
		return
	}
	g.setFocus(&c.w[i].body)
}

// focusNext moves to the next window in reading order (down the column,
// then the next column), wrapping around to the first.
func (g *globals) focusNext(t *Text) {
	var all []*Window
	for _, c := range g.row.col {
		all = append(all, c.w...)
	}
	if len(all) == 0 {
		return
	}
	w, _ := focusWindow(t)
	next := all[0]
	for i, cw := range all {
		if cw == w {
			next = all[(i+1)%len(all)]
			break
		}
	}
	g.setFocus(&next.body)
}

// focusPrev returns to the text that had the focus before the last change.
func (g *globals) focusPrev() {
	if p := g.prevfocus; p != nil && g.onScreen(p) {
		g.setFocus(p)
	}
}

// focusNumbered focuses window n (1-based, from the top) of the current
// column, or the column tag for n == 0.
func (g *globals) focusNumbered(t *Text, n int) {
	_, c := focusWindow(t)
	if c == nil {
		return
	}
	if n == 0 {
		g.setFocus(&c.tag)
		return
	}
	if n <= len(c.w) {
		g.setFocus(&c.w[n-1].body)
	}
}

// focusTagToggle switches the focus between the tag and body of a window.
func (g *globals) focusTagToggle(t *Text) {
	w, _ := focusWindow(t)
	if w == nil {
		return
	}
	if t == &w.tag {
		g.setFocus(&w.body)
	} else {
		g.setFocus(&w.tag)
	}
}

// focusCommandLine puts the cursor at the end of the window's tag, after
// the vertical bar, where commands are typed; for a column or row tag, at
// the end of that tag.
func (g *globals) focusCommandLine(t *Text) {
	if t == nil {
		return
	}
	tag := t
	if w, _ := focusWindow(t); w != nil {
		tag = &w.tag
	}
	n := tag.file.Nr()
	tag.SetSelect(n, n)
	g.setFocus(tag)
}

// growFocus resizes the focused window as a click with button but in its
// layout box would.
func (g *globals) growFocus(t *Text, but int) {
	w, c := focusWindow(t)
	if w == nil || c == nil {
		return
	}
	c.Grow(w, but)
	g.warpTo(&w.body)
}
