package main

import (
	"strings"
	"testing"

	"github.com/deusxyz/doomcode/editor/draw"
)

func TestParseKey(t *testing.T) {
	for _, tc := range []struct {
		name string
		want rune
	}{
		{"x", 'x'},
		{"%", '%'},
		{"C-a", 0x01},
		{"C-A", 0x01},
		{"C-z", 0x1a},
		{"C-[", 0x1b},
		{"C-_", 0x1f},
		{"C-/", 0x1f}, // what Ctrl+/ sends
		{"Cmd-/", draw.KeyCmd + '/'},
		{"Cmd-s", draw.KeyCmd + 's'},
		{"Cmd-Z", draw.KeyCmd + 'Z'},
		{"Left", draw.KeyLeft},
		{"PgDn", draw.KeyPageDown},
		{"Enter", '\n'},
		{"Tab", '\t'},
		{"Space", ' '},
		{"Esc", 0x1b},
		{"F3", KF | 3},
		{"F12", KF | 12},
		{"0xF800", 0xF800},
	} {
		got, err := ParseKey(tc.name)
		if err != nil {
			t.Errorf("ParseKey(%q): %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseKey(%q) = %#x; want %#x", tc.name, got, tc.want)
		}
	}
	for _, bad := range []string{"", "C-", "C-ab", "C-1", "Cmd-", "Cmd-ab", "F13", "F0", "0xZZ", "Nope"} {
		if r, err := ParseKey(bad); err == nil {
			t.Errorf("ParseKey(%q) = %#x; want error", bad, r)
		}
	}
}

func TestKeyNameRoundTrip(t *testing.T) {
	for _, name := range []string{"x", "C-a", "C-z", "C-]", "C-/", "Cmd-/", "Cmd-s", "Cmd-Z", "Left", "Home", "PgUp", "Enter", "Tab", "Esc", "Space", "F5", "0xF800"} {
		r, err := ParseKey(name)
		if err != nil {
			t.Fatalf("ParseKey(%q): %v", name, err)
		}
		if got := KeyName(r); got != name {
			t.Errorf("KeyName(ParseKey(%q)) = %q", name, got)
		}
	}
}

func TestDefaultKeymap(t *testing.T) {
	km := DefaultKeymap()
	for _, tc := range []struct{ key, action string }{
		{"C-e", "execute"},
		{"Cmd-e", "execute"},
		{"C-o", "look"},
		{"C-s", "put"},
		{"C-c", "snarf"},
		{"C-x", "cut"},
		{"C-v", "paste"},
		{"C-z", "undo"},
		{"C-y", "redo"},
		{"Cmd-Z", "redo"},
		{"Up", "cursor-up"},
		{"Home", "line-start"},
	} {
		r, _ := ParseKey(tc.key)
		a := km.Lookup(r)
		if a == nil || a.Name != tc.action {
			t.Errorf("%s bound to %v; want %s", tc.key, a, tc.action)
		}
	}
	// Ordinary typing and Acme's own editing keys are not actions.
	for _, key := range []string{"a", "Space", "Enter", "Tab", "Backspace", "C-u", "C-w", "C-h", "Esc", "Ins"} {
		r, _ := ParseKey(key)
		if a := km.Lookup(r); a != nil {
			t.Errorf("%s unexpectedly bound to %s", key, a.Name)
		}
	}
	// Every action in the table has a function and a doc string.
	for name, a := range actionTable {
		if a.Fn == nil || a.Doc == "" || a.Name != name {
			t.Errorf("action %q is incomplete: %+v", name, a)
		}
	}
}

func TestKeymapBindUnbindString(t *testing.T) {
	km := Keymap{}
	if err := km.Bind("C-q", "put"); err != nil {
		t.Fatal(err)
	}
	if err := km.Bind("F2", "execute"); err != nil {
		t.Fatal(err)
	}
	if err := km.Bind("C-q", "no-such-action"); err == nil {
		t.Error("Bind with unknown action succeeded")
	}
	if err := km.Bind("Bogus", "put"); err == nil {
		t.Error("Bind with unknown key succeeded")
	}
	want := "C-q          put\nF2           execute\n"
	if got := km.String(); got != want {
		t.Errorf("String() = %q; want %q", got, want)
	}
	if err := km.Unbind("C-q"); err != nil {
		t.Fatal(err)
	}
	if km.Lookup(0x11) != nil {
		t.Error("C-q still bound after Unbind")
	}
	if !strings.HasPrefix(km.String(), "F2") {
		t.Errorf("String() after Unbind = %q", km.String())
	}
}

func TestTypeUsesKeymap(t *testing.T) {
	text := makeKeyTestBody("abc\ndef", 1, 1)
	// Rebinding changes what Type does.
	global.keymap = Keymap{}
	defer func() { global.keymap = DefaultKeymap() }()
	if err := global.keymap.Bind("F2", "line-end"); err != nil {
		t.Fatal(err)
	}
	text.Type(KF | 2)
	if text.q0 != 3 || text.q1 != 3 {
		t.Fatalf("F2 bound to line-end: q0,q1 = %d,%d; want 3,3", text.q0, text.q1)
	}
	// With Home unbound it is typed as text? No: unknown runes outside
	// the printable range are still inserted by Acme, so just check that
	// the binding is gone.
	if global.keymap.Lookup(draw.KeyHome) != nil {
		t.Error("Home should be unbound in an empty keymap")
	}
}
