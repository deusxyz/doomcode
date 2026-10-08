package main

import "testing"

func TestCommentPrefix(t *testing.T) {
	for name, want := range map[string]string{
		"/a/b/main.go": "//", "x.RS": "//", "run.sh": "#", "conf.yml": "#", "q.sql": "--",
		"init.el": ";", "paper.tex": "%", "/src/Makefile": "#", "mkfile": "#", "notes.md": "", "noext": "",
	} {
		if got := commentPrefix(name); got != want {
			t.Errorf("commentPrefix(%q) = %q; want %q", name, got, want)
		}
	}
}

func TestCommentToggle(t *testing.T) {
	text := makeKeyTestBody("func f() {\n\tx := 1\n\n\ty := 2\n}\n", 11, 22) // "\tx := 1\n\n\ty" (lines 2-4)
	text.w.body.file.SetName("/tmp/a.go")
	text.commentToggle()
	if got, want := text.file.String(), "func f() {\n\t// x := 1\n\n\t// y := 2\n}\n"; got != want {
		t.Fatalf("comment: %q; want %q", got, want)
	}
	wantSel(t, "after comment, whole lines", text, 11, 34)
	text.commentToggle()
	if got, want := text.file.String(), "func f() {\n\tx := 1\n\n\ty := 2\n}\n"; got != want {
		t.Fatalf("uncomment: %q; want %q", got, want)
	}
	wantSel(t, "after uncomment", text, 11, 28)

	// A plain caret toggles its line and stays put relative to the text.
	text.SetSelect(14, 14) // after "x "
	text.commentToggle()
	if got, want := text.file.String(), "func f() {\n\t// x := 1\n\n\ty := 2\n}\n"; got != want {
		t.Fatalf("caret comment: %q", got)
	}
	wantSel(t, "caret moves with the text", text, 17, 17)
	text.commentToggle()
	wantSel(t, "caret back", text, 14, 14)

	// Mixed lines (one commented) comment everything; the shallowest
	// indentation wins.
	text = makeKeyTestBody("# a\n  b\n", 0, 8)
	text.w.body.file.SetName("/tmp/a.sh")
	text.commentToggle()
	if got, want := text.file.String(), "# # a\n#   b\n"; got != want {
		t.Fatalf("mixed: %q; want %q", got, want)
	}
}

func TestCommentToggleUnknownType(t *testing.T) {
	text := makeKeyTestBody("hello\n", 0, 6)
	text.w.body.file.SetName("/tmp/README")
	text.commentToggle()
	if got := text.file.String(); got != "hello\n" {
		t.Fatalf("unknown type changed the text: %q", got)
	}
}

func TestBlockCommentToggle(t *testing.T) {
	src := "# Title\n\n  some text\n  more\n\nafter\n"
	text := makeKeyTestBody(src, 9, 25) // from "  some text" into "more"
	text.w.body.file.SetName("/tmp/README.md")
	text.commentToggle()
	want := "# Title\n\n  <!-- some text\n  more -->\n\nafter\n"
	if got := text.file.String(); got != want {
		t.Fatalf("comment: %q; want %q", got, want)
	}
	wantSel(t, "whole lines after comment", text, 9, 37)
	text.commentToggle()
	if got := text.file.String(); got != src {
		t.Fatalf("uncomment: %q; want %q", got, src)
	}
	wantSel(t, "whole lines after uncomment", text, 9, 28)

	// A caret toggles its line and moves with the text.
	text.SetSelect(14, 14) // in "some"
	text.commentToggle()
	if got, want := text.file.String(), "# Title\n\n  <!-- some text -->\n  more\n\nafter\n"; got != want {
		t.Fatalf("caret comment: %q; want %q", got, want)
	}
	wantSel(t, "caret after comment", text, 19, 19)
	text.commentToggle()
	if got := text.file.String(); got != src {
		t.Fatalf("caret uncomment: %q", got)
	}
	wantSel(t, "caret after uncomment", text, 14, 14)

	// CSS uses /* */; blank lines alone do nothing.
	css := makeKeyTestBody("a { color: red }\n\n", 0, 0)
	css.w.body.file.SetName("/tmp/x.css")
	css.commentToggle()
	if got := css.file.String(); got != "/* a { color: red } */\n\n" {
		t.Fatalf("css: %q", got)
	}
	css.SetSelect(23, 23) // the blank line
	css.commentToggle()
	if got := css.file.String(); got != "/* a { color: red } */\n\n" {
		t.Fatalf("blank line changed: %q", got)
	}
	if o, c := commentBlock("/x/page.HTML"); o != "<!--" || c != "-->" {
		t.Errorf("commentBlock(.HTML) = %q %q", o, c)
	}
	if o, _ := commentBlock("/x/main.go"); o != "" {
		t.Errorf("Go has a block comment entry: %q", o)
	}
}

func TestCommentToggleKeys(t *testing.T) {
	km := DefaultKeymap()
	for _, key := range []string{"C-/", "Cmd-/"} {
		r, err := ParseKey(key)
		if err != nil {
			t.Fatal(err)
		}
		if a := km.Lookup(r); a == nil || a.Name != "comment-toggle" {
			t.Errorf("%s bound to %v; want comment-toggle", key, a)
		}
	}
	// Typing Ctrl+/ in a Go body comments the line.
	text := makeKeyTestBody("x := 1\n", 0, 0)
	text.w.body.file.SetName("/tmp/a.go")
	text.Type(0x1f)
	if got := text.file.String(); got != "// x := 1\n" {
		t.Errorf("Ctrl+/ gave %q", got)
	}
}
