package main

import (
	"image"
	"testing"

	"github.com/rjkroege/edwood/draw"
	"github.com/rjkroege/edwood/dumpfile"
)

func TestPrefixStateFeed(t *testing.T) {
	km := DefaultPrefixKeymap()
	var p prefixState

	if res, _ := p.feed('a', km); res != prefixPass {
		t.Fatalf("plain key: %v; want pass", res)
	}
	if res, _ := p.feed(prefixKey, km); res != prefixArmed || !p.armed {
		t.Fatalf("Ctrl-B: %v armed=%v; want armed", res, p.armed)
	}
	res, a := p.feed(draw.KeyLeft, km)
	if res != prefixRun || a == nil || a.Name != "focus-left" || p.armed {
		t.Fatalf("Ctrl-B Left: %v %v armed=%v; want run focus-left", res, a, p.armed)
	}
	p.feed(prefixKey, km)
	if res, _ := p.feed(prefixKey, km); res != prefixArmed || !p.armed {
		t.Fatalf("Ctrl-B Ctrl-B (auto-repeat): %v armed=%v; want still armed", res, p.armed)
	}
	// ...and the action after the repeats still runs.
	if res, a := p.feed('o', km); res != prefixRun || a == nil || a.Name != "focus-next" {
		t.Fatalf("Ctrl-B Ctrl-B o: %v %v; want run focus-next", res, a)
	}
	p.feed(prefixKey, km)
	if res, _ := p.feed(0x1b, km); res != prefixCancel || p.armed {
		t.Fatalf("Ctrl-B Esc: %v; want cancel", res)
	}
	p.feed(prefixKey, km)
	if res, _ := p.feed('~', km); res != prefixCancel || p.armed {
		t.Fatalf("Ctrl-B unbound: %v; want cancel", res)
	}
	// After a cancel, the next key is ordinary again.
	if res, _ := p.feed('x', km); res != prefixPass {
		t.Fatalf("after cancel: %v; want pass", res)
	}
}

func TestDefaultPrefixKeymap(t *testing.T) {
	km := DefaultPrefixKeymap()
	for _, tc := range []struct{ key, action string }{
		{"o", "focus-next"}, {";", "focus-prev"}, {"0", "focus-0"}, {"9", "focus-9"},
		{"c", "new"}, {"%", "newcol"}, {"x", "del"}, {"&", "delcol"},
		{"z", "zoom"}, {"Z", "maximize"}, {":", "command-line"}, {"t", "tag-toggle"},
		{"Enter", "execute"}, {"/", "look"}, {"g", "file-start"}, {"G", "file-end"},
	} {
		r, _ := ParseKey(tc.key)
		if a := km.Lookup(r); a == nil || a.Name != tc.action {
			t.Errorf("prefix %s = %v; want %s", tc.key, a, tc.action)
		}
	}
	for name, a := range actionTable {
		if a.Fn == nil || a.Doc == "" || a.Name != name {
			t.Errorf("action %q incomplete", name)
		}
	}
}

// makeFocusScaffold builds two columns with windows at known screen
// positions:
//
//	column 0 (x 0-100):   w0 y 0-100, w1 y 100-200
//	column 1 (x 100-200): w2 y 0-50,  w3 y 50-200
func makeFocusScaffold() (cols []*Column, wins []*Window) {
	MakeWindowScaffold(&dumpfile.Content{
		Columns: []dumpfile.Column{{}, {}},
		Windows: []*dumpfile.Window{
			{Column: 0, Tag: dumpfile.Text{Buffer: "/a/w0 Del"}, Body: dumpfile.Text{Buffer: "w0"}},
			{Column: 0, Tag: dumpfile.Text{Buffer: "/a/w1 Del"}, Body: dumpfile.Text{Buffer: "w1"}},
			{Column: 1, Tag: dumpfile.Text{Buffer: "/a/w2 Del"}, Body: dumpfile.Text{Buffer: "w2"}},
			{Column: 1, Tag: dumpfile.Text{Buffer: "/a/w3 Del"}, Body: dumpfile.Text{Buffer: "w3"}},
		},
	})
	cols = global.row.col
	for _, c := range cols {
		c.tag.col = c // as Column.Init does; the scaffold leaves it unset
	}
	cols[0].r = image.Rect(0, 0, 100, 200)
	cols[1].r = image.Rect(100, 0, 200, 200)
	wins = append(wins, cols[0].w...)
	wins = append(wins, cols[1].w...)
	wins[0].r = image.Rect(0, 0, 100, 100)
	wins[1].r = image.Rect(0, 100, 100, 200)
	wins[2].r = image.Rect(100, 0, 200, 50)
	wins[3].r = image.Rect(100, 50, 200, 200)
	global.barttext = nil
	global.prevfocus = nil
	global.focusSticky = false
	return cols, wins
}

func focusIs(t *testing.T, what string, want *Text) {
	t.Helper()
	if global.barttext != want {
		t.Errorf("%s: focus is %v; want %v", what, global.barttext, want)
	}
	if !global.focusSticky {
		t.Errorf("%s: focus not sticky after keyboard navigation", what)
	}
}

func TestFocusNavigation(t *testing.T) {
	cols, w := makeFocusScaffold()

	global.setFocus(&w[1].body) // bottom-left, center y 150
	global.focusColumn(&w[1].body, 1)
	focusIs(t, "right from w1", &w[3].body)

	global.focusColumn(&w[3].body, 1) // no column to the right: stays
	focusIs(t, "right at edge", &w[3].body)

	global.focusColumn(&w[2].body, -1) // w2 center y 25 -> w0
	focusIs(t, "left from w2", &w[0].body)

	global.focusWindowInColumn(&w[0].body, 1)
	focusIs(t, "down from w0", &w[1].body)
	global.focusWindowInColumn(&w[1].body, 1) // bottom: stays
	focusIs(t, "down at bottom", &w[1].body)
	global.focusWindowInColumn(&w[0].body, -1) // up from the top window: column tag
	focusIs(t, "up from w0", &cols[0].tag)
	global.focusWindowInColumn(&cols[0].tag, 1) // down from the column tag: first window
	focusIs(t, "down from column tag", &w[0].body)

	global.focusNext(&w[1].body)
	focusIs(t, "next from w1", &w[2].body)
	global.focusNext(&w[3].body) // wraps
	focusIs(t, "next wraps", &w[0].body)

	global.focusNumbered(&w[0].body, 2)
	focusIs(t, "focus-2", &w[1].body)
	global.focusNumbered(&w[1].body, 0)
	focusIs(t, "focus-0", &cols[0].tag)
	global.focusNumbered(&w[1].body, 7) // no such window: stays
	focusIs(t, "focus-7", &cols[0].tag)

	global.setFocus(&w[2].body)
	global.setFocus(&w[3].body)
	global.focusPrev()
	focusIs(t, "prev", &w[2].body)

	global.focusTagToggle(&w[2].body)
	focusIs(t, "tag-toggle to tag", &w[2].tag)
	global.focusTagToggle(&w[2].tag)
	focusIs(t, "tag-toggle to body", &w[2].body)

	global.focusCommandLine(&w[3].body)
	focusIs(t, "command-line", &w[3].tag)
	if n := w[3].tag.file.Nr(); w[3].tag.q0 != n || w[3].tag.q1 != n {
		t.Errorf("command-line: tag selection %d,%d; want %d,%d", w[3].tag.q0, w[3].tag.q1, n, n)
	}
}

func TestFocusFromRowTag(t *testing.T) {
	cols, w := makeFocusScaffold()
	global.focusColumn(&global.row.tag, 1) // from the row tag, right goes to the first column
	focusIs(t, "right from row tag", &w[0].body)
	global.focusColumn(&global.row.tag, -1) // left goes to the last column
	focusIs(t, "left from row tag", &w[2].body)
	_ = cols
}

func TestFocusTextFallsBackToMouse(t *testing.T) {
	_, w := makeFocusScaffold()
	m := draw.Mouse{Point: image.Pt(150, 100)}
	global.mouse = &m
	defer func() { global.mouse = nil }()
	// No explicit focus: the text under the mouse.
	global.barttext = nil
	global.row.col[1].w[1].body.all = w[3].r
	if got := global.focusText(); got != nil && got.w != w[3] {
		t.Errorf("focusText under mouse = %v; want w3", got)
	}
	// A dead window is not a valid focus.
	dead := &Text{w: &Window{}, what: Body}
	global.barttext = dead
	if got := global.focusText(); got == dead {
		t.Error("focusText returned a text whose window has no column")
	}
}

func TestRowTypeHonoursStickyFocus(t *testing.T) {
	_, w := makeFocusScaffold()
	global.setFocus(&w[1].body)
	// The mouse is nowhere near w1, but sticky focus wins.
	global.row.Type('q', image.Pt(150, 10))
	if got := w[1].body.file.String(); got != "qw1" && got != "w1q" {
		t.Errorf("typed into %q; want the key to land in w1", got)
	}
	global.focusSticky = false
}

// rectFrame is a MockFrame with a real on-screen rectangle, so that
// Column.Which can hit-test the body.
type rectFrame struct {
	MockFrame
	r image.Rectangle
}

func (f *rectFrame) Rect() image.Rectangle { return f.r }

func TestRowTypeBeforeFirstClickGoesUnderMouse(t *testing.T) {
	_, w := makeFocusScaffold()
	global.barttext = nil
	global.focusSticky = false
	w[3].body.all = w[3].r
	w[3].body.fr = &rectFrame{r: w[3].r}
	global.row.Type('q', image.Pt(150, 100)) // over w3, nothing clicked yet
	if got := w[3].body.file.String(); got != "qw3" && got != "w3q" {
		t.Errorf("typed into %q; want the key to land in w3 under the mouse", got)
	}
}
