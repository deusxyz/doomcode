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
