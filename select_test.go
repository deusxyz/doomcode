package main

import (
	"image"
	"testing"

	"github.com/rjkroege/edwood/draw"
)

func wantSel(t *testing.T, what string, text *Text, q0, q1 int) {
	t.Helper()
	if text.q0 != q0 || text.q1 != q1 {
		t.Errorf("%s: selection %d,%d; want %d,%d", what, text.q0, text.q1, q0, q1)
	}
}

func TestAnchorExtendsSelection(t *testing.T) {
	// "abcdef\nxyz\n123456"
	text := makeKeyTestBody("abcdef\nxyz\n123456", 2, 2)
	text.keyAnchor()
	text.Type(draw.KeyRight)
	text.Type(draw.KeyRight)
	wantSel(t, "anchor, Right Right", text, 2, 4)
	text.Type(draw.KeyLeft)
	wantSel(t, "then Left", text, 2, 3)
	text.Type(draw.KeyLeft)
	text.Type(draw.KeyLeft)
	text.Type(draw.KeyLeft)
	wantSel(t, "past the anchor", text, 0, 2) // caret at 0, anchor at 2
	text.Type(draw.KeyEnd)
	wantSel(t, "End", text, 2, 6)
	text.Type(draw.KeyDown)
	wantSel(t, "Down keeps column 6 -> clamps to end of xyz", text, 2, 10)
	text.Type(draw.KeyHome)
	wantSel(t, "Home", text, 2, 7)
	// Esc ends anchor mode and leaves the caret.
	text.Type(0x1b)
	wantSel(t, "Esc", text, 7, 7)
	if text.anchorOn {
		t.Error("anchor still on after Esc")
	}
	// Plain movement again collapses as usual.
	text.Type(draw.KeyRight)
	wantSel(t, "Right after Esc", text, 8, 8)
}

func TestAnchorEndsOnTypingAndCopy(t *testing.T) {
	text := makeKeyTestBody("hello world", 0, 0)
	text.keyAnchor()
	text.Type(draw.KeyRight)
	text.Type(draw.KeyRight)
	wantSel(t, "two right", text, 0, 2)
	text.Type(0x03) // ^C snarf ends anchor mode, keeps selection
	if text.anchorOn {
		t.Error("anchor still on after ^C")
	}
	wantSel(t, "after ^C", text, 0, 2)
	text.keyAnchor()
	text.Type(draw.KeyRight)
	wantSel(t, "anchor at q0 with selection: caret is q1", text, 0, 3)
	text.Type('X') // typing replaces the selection and ends anchor mode
	if text.anchorOn {
		t.Error("anchor still on after typing")
	}
	if got := text.file.String(); got != "Xlo world" {
		t.Errorf("buffer %q; want %q", got, "Xlo world")
	}
}

func TestSelectLine(t *testing.T) {
	text := makeKeyTestBody("one\ntwo\nthree", 5, 5)
	text.Type(0x0c) // ^L
	wantSel(t, "^L", text, 4, 8)
	text.Type(0x0c)
	wantSel(t, "^L again adds the next line", text, 4, 13)
	text.Type(0x0c) // nothing more to add
	wantSel(t, "^L at end", text, 4, 13)
}

func TestSelectWordAndNext(t *testing.T) {
	text := makeKeyTestBody("foo bar foo baz foo", 1, 1)
	text.Type(0x04) // ^D selects the word under the cursor
	wantSel(t, "^D word", text, 0, 3)
	text.Type(0x04) // ^D again: next occurrence
	wantSel(t, "^D next", text, 8, 11)
	text.Type(0x04)
	wantSel(t, "^D next again", text, 16, 19)
	text.Type(0x04) // wraps
	wantSel(t, "^D wraps", text, 0, 3)
}

func TestSelectBlock(t *testing.T) {
	//            0123456789012345
	text := makeKeyTestBody("f(a, g(b, c), d)", 8, 8) // caret inside "b, c"
	text.keySelectBlock()
	wantSel(t, "inside inner", text, 7, 11)
	text.keySelectBlock()
	wantSel(t, "inner with brackets", text, 6, 12)
	text.keySelectBlock()
	wantSel(t, "inside outer", text, 2, 15)
	text.keySelectBlock()
	wantSel(t, "outer with brackets", text, 1, 16)
	text.keySelectBlock() // nothing encloses: unchanged
	wantSel(t, "no enclosing block", text, 1, 16)

	text.SetSelect(2, 2) // right after "(": the inside of that bracket
	text.keySelectBlock()
	wantSel(t, "after opener", text, 2, 15)
	text.SetSelect(0, 0)
	text.keySelectBlock()
	wantSel(t, "outside any block", text, 0, 0)
}

func TestIndentOutdent(t *testing.T) {
	text := makeKeyTestBody("a\nb\nc\nd", 2, 4) // selects "b\n" (lines b)
	text.Type('\t')
	if got := text.file.String(); got != "a\n\tb\nc\nd" {
		t.Fatalf("Tab on one selected line: %q", got)
	}
	wantSel(t, "after indent", text, 2, 5)

	text.SetSelect(0, 7) // a .. c (through "c\n"? positions: a\n\tb\nc\nd -> c at 6)
	text.Type('\t')
	if got := text.file.String(); got != "\ta\n\t\tb\n\tc\nd" {
		t.Fatalf("Tab on three lines: %q", got)
	}
	// Outdent them all back.
	text.SetSelect(0, text.lineEnd(text.q1))
	text.outdentLines()
	if got := text.file.String(); got != "a\n\tb\nc\nd" {
		t.Fatalf("outdent: %q", got)
	}
	text.outdentLines()
	if got := text.file.String(); got != "a\nb\nc\nd" {
		t.Fatalf("outdent again: %q", got)
	}
	text.outdentLines() // nothing left to remove
	if got := text.file.String(); got != "a\nb\nc\nd" {
		t.Fatalf("outdent with nothing to remove: %q", got)
	}

	// Tab with a selection inside one line still replaces it with a tab.
	text.SetSelect(0, 1)
	text.Type('\t')
	if got := text.file.String(); got != "\t\nb\nc\nd" {
		t.Fatalf("Tab within a line: %q", got)
	}
}

func TestIndentWithTabexpand(t *testing.T) {
	text := makeKeyTestBody("a\nb", 0, 3)
	text.tabexpand = true
	text.tabstop = 2
	text.indentLines()
	if got := text.file.String(); got != "  a\n  b" {
		t.Fatalf("indent with spaces: %q", got)
	}
	text.outdentLines()
	if got := text.file.String(); got != "a\nb" {
		t.Fatalf("outdent spaces: %q", got)
	}
}

func TestFindAndGotoTypeIntoTag(t *testing.T) {
	text := makeKeyTestBody("alpha beta", 0, 0)
	defer func() { global.barttext = nil; global.focusSticky = false }()
	tag := &text.w.tag
	before := tag.file.String()
	text.Type(0x06) // ^F with no selection
	got := tag.file.String()
	if want := before + " Look "; got != want && got != before+"Look " {
		t.Fatalf("tag after ^F: %q; want %q", got, want)
	}
	if tag.q0 != tag.file.Nr() || tag.q1 != tag.q0 {
		t.Errorf("caret not at end of tag: %d,%d (len %d)", tag.q0, tag.q1, tag.file.Nr())
	}
	if tag.eq0 != tag.file.Nr()-len("Look ") {
		t.Errorf("eq0 = %d; want start of \"Look \" at %d", tag.eq0, tag.file.Nr()-len("Look "))
	}
	if global.barttext != tag {
		t.Error("focus did not move to the tag")
	}
	// Typing the term and Esc selects "Look term" for ^E.
	for _, r := range "beta" {
		tag.Type(r)
	}
	tag.Type(0x1b)
	r := make([]rune, tag.q1-tag.q0)
	tag.file.Read(tag.q0, r)
	if string(r) != "Look beta" {
		t.Errorf("Esc selected %q; want %q", string(r), "Look beta")
	}

	// ^F with a selection searches instead.
	text.SetSelect(0, 5) // "alpha"
	text.file.InsertAt(10, []rune(" alpha"))
	text.Type(0x06)
	wantSel(t, "^F next occurrence", text, 11, 16)

	// ^G types ":".
	n := tag.file.Nr()
	text.Type(0x07)
	if s := tag.file.String(); s[len(s)-1] != ':' || tag.file.Nr() < n+1 {
		t.Errorf("tag after ^G: %q", s)
	}
}

func TestFindMovesKeyboardFocusToTag(t *testing.T) {
	text := makeKeyTestBody("alpha beta", 0, 0)
	defer func() { global.barttext = nil; global.focusSticky = false }()
	tag := &text.w.tag
	global.barttext = text
	global.focusSticky = true
	// As keyboardthread does: deliver the key, then record where it went,
	// unless the action moved the focus itself.
	if got := global.typeKey(0x06, image.Pt(0, 0)); got != text {
		t.Fatalf("^F went to %v; want the body", got)
	}
	if global.barttext != tag {
		t.Fatal("focus did not stay on the tag after ^F")
	}
	n := tag.file.Nr()
	if got := global.typeKey('x', image.Pt(0, 0)); got != tag {
		t.Fatalf("next key went to %v; want the tag", got)
	}
	if tag.file.Nr() != n+1 || tag.file.String()[n] != 'x' {
		t.Errorf("tag after typing: %q", tag.file.String())
	}
	if global.barttext != tag {
		t.Error("ordinary typing into the tag moved the focus away")
	}
	// Ordinary typing still records the focus.
	global.barttext = text
	global.typeKey('y', image.Pt(0, 0))
	if global.barttext != text {
		t.Error("typing into the body did not record it as the focus")
	}
}
