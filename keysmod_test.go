package main

import (
	"testing"

	"github.com/rjkroege/edwood/draw"
)

func TestParseModifiedSpecialKeys(t *testing.T) {
	for name, want := range map[string]rune{
		"S-Left":      modKey(modShift, 1),
		"C-Right":     modKey(modCtl, 2),
		"C-S-Left":    modKey(modCtl|modShift, 1),
		"S-C-Left":    modKey(modCtl|modShift, 1), // order does not matter
		"M-Enter":     modKey(modAlt, 13),
		"Cmd-Up":      modKey(modCmd, 3),
		"C-Backspace": modKey(modCtl, 11),
		"S-Tab":       modKey(modShift, 12),
		"C-Home":      modKey(modCtl, 5),
		"Left":        draw.KeyLeft, // unmodified stays as before
		"C-a":         0x01,         // control letters stay as before
		"Cmd-a":       draw.KeyCmd + 'a',
	} {
		got, err := ParseKey(name)
		if err != nil || got != want {
			t.Errorf("ParseKey(%q) = %#x,%v; want %#x", name, got, err, want)
		}
	}
	for _, bad := range []string{"S-", "S-a", "S-C-", "X-Left", "S-Nope", "S-F3"} {
		if r, err := ParseKey(bad); err == nil {
			t.Errorf("ParseKey(%q) = %#x; want error", bad, r)
		}
	}
	for _, name := range []string{"S-Left", "C-S-Right", "M-Enter", "Cmd-Down", "C-Backspace", "S-Tab", "C-M-Esc"} {
		r, err := ParseKey(name)
		if err != nil {
			t.Fatal(err)
		}
		if got := KeyName(r); got != name {
			t.Errorf("KeyName(ParseKey(%q)) = %q", name, got)
		}
	}
}

func TestModifiedKeysSelectAndMove(t *testing.T) {
	text := makeKeyTestBody("alpha beta\ngamma delta\n", 0, 0)
	sRight, _ := ParseKey("S-Right")
	sDown, _ := ParseKey("S-Down")
	cRight, _ := ParseKey("C-Right")
	cLeft, _ := ParseKey("C-Left")
	csRight, _ := ParseKey("C-S-Right")
	cEnd, _ := ParseKey("C-End")
	cHome, _ := ParseKey("C-Home")
	sHome, _ := ParseKey("S-Home")

	text.Type(sRight)
	text.Type(sRight)
	wantSel(t, "Shift-Right twice", text, 0, 2)
	text.Type(sDown)
	wantSel(t, "Shift-Down keeps the column", text, 0, 13)
	text.Type(0x1b) // Esc ends the selection
	wantSel(t, "Esc", text, 13, 13)

	text.Type(cRight) // to the end of "gamma"
	wantSel(t, "Ctrl-Right", text, 16, 16)
	text.Type(cRight) // over the space to the end of "delta"
	wantSel(t, "Ctrl-Right again", text, 22, 22)
	text.Type(cLeft) // back to the start of "delta"
	wantSel(t, "Ctrl-Left", text, 17, 17)
	text.Type(csRight) // select "delta"
	wantSel(t, "Ctrl-Shift-Right", text, 17, 22)
	text.Type(sHome) // extend back to the line start
	wantSel(t, "Shift-Home from a selection", text, 11, 17)

	text.Type(cEnd)
	wantSel(t, "Ctrl-End", text, 23, 23)
	text.Type(cHome)
	wantSel(t, "Ctrl-Home", text, 0, 0)
}

func TestModifiedKeysDeleteWords(t *testing.T) {
	text := makeKeyTestBody("one two three", 7, 7) // after "one two"
	cBs, _ := ParseKey("C-Backspace")
	cDel, _ := ParseKey("C-Del")
	text.Type(cBs)
	if got := text.file.String(); got != "one  three" {
		t.Fatalf("Ctrl-Backspace: %q", got)
	}
	wantSel(t, "after Ctrl-Backspace", text, 4, 4)
	text.Type(cDel) // deletes the space and "three"? no: one word forward: " three" -> the space then the word
	if got := text.file.String(); got != "one " {
		t.Fatalf("Ctrl-Del: %q", got)
	}
}

func TestModifiedKeysExecuteBindings(t *testing.T) {
	km := DefaultKeymap()
	for _, tc := range []struct{ key, action string }{
		{"C-Enter", "execute"}, {"Cmd-Enter", "execute"}, {"M-Enter", "look"},
		{"S-Tab", "outdent"}, {"C-Up", "scroll-up"}, {"C-Down", "scroll-down"},
		{"Cmd-Left", "line-start"}, {"Cmd-Right", "line-end"}, {"Cmd-Up", "file-start"}, {"Cmd-Down", "file-end"},
		{"S-Left", "select-left"}, {"S-End", "select-line-end"}, {"C-S-Left", "select-word-left"},
	} {
		r, err := ParseKey(tc.key)
		if err != nil {
			t.Fatal(err)
		}
		if a := km.Lookup(r); a == nil || a.Name != tc.action {
			t.Errorf("%s bound to %v; want %s", tc.key, a, tc.action)
		}
	}
}
