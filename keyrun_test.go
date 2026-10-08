package main

import (
	"strings"
	"testing"
)

func TestCommandAction(t *testing.T) {
	if a := commandAction("run   L   def "); a == nil || a.Name != "run L def" {
		t.Errorf("run action = %v", a)
	}
	if a := commandAction("tag L rn"); a == nil || a.Name != "tag L rn" {
		t.Errorf("tag action = %v", a)
	}
	for _, bad := range []string{"run", "run ", "tag  ", "runx", "cursor-left"} {
		if a := commandAction(bad); a != nil {
			t.Errorf("commandAction(%q) = %v; want nil", bad, a)
		}
	}
}

func TestDefaultLspBindings(t *testing.T) {
	km, pkm := DefaultKeymap(), DefaultPrefixKeymap()
	for _, tc := range []struct {
		km          Keymap
		key, action string
	}{
		{km, "F12", "run L def"}, {km, "F2", "tag L rn"}, {km, "M-Esc", "run L comp -e"},
		{pkm, "d", "run L def"}, {pkm, "r", "run L refs"}, {pkm, "R", "tag L rn"},
		{pkm, "h", "run L hov"}, {pkm, "i", "run L impls"}, {pkm, "k", "run L sig"},
	} {
		r, err := ParseKey(tc.key)
		if err != nil {
			t.Fatal(err)
		}
		if a := tc.km.Lookup(r); a == nil || a.Name != tc.action {
			t.Errorf("%s bound to %v; want %s", tc.key, a, tc.action)
		}
	}
}

func TestKeysFileRunAndTag(t *testing.T) {
	km, pkm := DefaultKeymap(), DefaultPrefixKeymap()
	src := "F5 run go test ./...\nprefix e run Edit ,x/foo/\nF6 tag Edit s/x/y/\nF7 run\n"
	n, errs := loadKeysText(km, pkm, strings.NewReader(src))
	if n != 3 || len(errs) != 1 {
		t.Fatalf("applied %d, errors %v; want 3 and one error for the empty run", n, errs)
	}
	if a := km.Lookup(KF | 5); a == nil || a.Name != "run go test ./..." {
		t.Errorf("F5 = %v", a)
	}
	if a := pkm.Lookup('e'); a == nil || a.Name != "run Edit ,x/foo/" {
		t.Errorf("prefix e = %v", a)
	}
	// The listing round-trips through the file format.
	if !strings.Contains(km.String(), "run go test ./...") {
		t.Errorf("listing lacks the run binding:\n%s", km.String())
	}
}

func TestRunBuiltinFromKey(t *testing.T) {
	// A built-in command runs inside Edwood: "Edit ,s/a/b/g" edits the
	// body of the focused window, as button 2 on that text would.
	text := makeKeyTestBody("aaa\n", 0, 0)
	executeCommand(text, "Edit ,s/a/b/g")
	if got := text.file.String(); got != "bbb\n" {
		t.Errorf("body after Edit: %q", got)
	}
}

func TestTagActionTypesIntoTag(t *testing.T) {
	text := makeKeyTestBody("x", 0, 0)
	defer func() { global.barttext = nil; global.focusSticky = false }()
	commandAction("tag L rn").Fn(text)
	tag := &text.w.tag
	if s := tag.file.String(); !strings.HasSuffix(s, "L rn ") {
		t.Errorf("tag %q; want it to end with \"L rn \"", s)
	}
	if global.barttext != tag {
		t.Error("focus did not move to the tag")
	}
}
