package main

import (
	"strings"
	"testing"
)

func TestFindNext(t *testing.T) {
	text := makeKeyTestBody("foo bar foo baz foo", 0, 0)
	global.lastsearch = nil
	defer func() { global.lastsearch = nil }()

	text.keyFindNext() // nothing to search for yet
	wantSel(t, "no last search", text, 0, 0)

	text.SetSelect(0, 3) // "foo": search for the selection
	text.keyFindNext()
	wantSel(t, "selection", text, 8, 11)
	if string(global.lastsearch) != "foo" {
		t.Errorf("lastsearch = %q", global.lastsearch)
	}

	text.SetSelect(12, 12) // collapsed: repeats the last search from here
	text.keyFindNext()
	wantSel(t, "repeat", text, 16, 19)
	text.keyFindNext() // wraps
	wantSel(t, "wrap", text, 0, 3)
}

func TestMoveWindowInColumn(t *testing.T) {
	cols, w := makeFocusScaffold()
	c := cols[0] // w0, w1
	if !c.MoveWindow(w[0], 1) || c.w[0] != w[1] || c.w[1] != w[0] {
		t.Errorf("move down: order %v", c.w)
	}
	if c.MoveWindow(w[0], 1) { // now last: cannot move further down
		t.Error("moved past the bottom")
	}
	if !c.MoveWindow(w[0], -1) || c.w[0] != w[0] {
		t.Errorf("move up: order %v", c.w)
	}
	if c.MoveWindow(w[0], -1) {
		t.Error("moved past the top")
	}
	if c.MoveWindow(w[2], 1) { // not in this column
		t.Error("moved a window from another column")
	}
	// Through the globals wrapper with a focus text.
	global.moveWindow(&w[1].body, -1)
	if c.w[0] != w[1] {
		t.Errorf("moveWindow via focus: order %v", c.w)
	}
}

func TestWindowListText(t *testing.T) {
	_, w := makeFocusScaffold()
	got := global.windowListText(nil)
	for _, want := range []string{"\t/a/w0\n", "\t/a/w1\n", "\t/a/w2\n", "\t/a/w3\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("list lacks %q:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "\n\n") {
		t.Errorf("columns not separated by a blank line:\n%s", got)
	}
	got = global.windowListText(w[2])
	if strings.Contains(got, "/a/w2") {
		t.Errorf("skipped window still listed:\n%s", got)
	}
	if name := global.windowListName(); !strings.HasSuffix(name, "+windows") {
		t.Errorf("list window name %q", name)
	}
}

func TestShrinkNeedsNeighbour(t *testing.T) {
	cols, w := makeFocusScaffold()
	cols[1].w = cols[1].w[:1]       // a column with a single window
	global.shrinkFocus(&w[2].body)  // must not panic or call Grow
	global.shrinkFocus(nil)         // nor with no focus
	global.moveWindowColumn(nil, 1) // nor these
	global.moveWindow(nil, 1)
}
