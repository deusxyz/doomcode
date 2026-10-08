package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// needUnixTools skips tests that run sh, tr and false as formatters.
func needUnixTools(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("formatter tests run Unix tools")
	}
	for _, tool := range []string{"sh", "tr", "false"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("no %s: %v", tool, err)
		}
	}
}

func TestReplaceAllKeepsSelection(t *testing.T) {
	// Change in the middle: selection before it stays, after it shifts.
	text := makeKeyTestBody("aaa\nbbb\nccc\n", 1, 2)
	text.replaceAll("aaa\nBBBBB\nccc\n")
	if got := text.file.String(); got != "aaa\nBBBBB\nccc\n" {
		t.Fatalf("text %q", got)
	}
	wantSel(t, "selection before the change", text, 1, 2)
	text.SetSelect(9, 10) // "c" after the change
	text.replaceAll("aaa\nbb\nccc\n")
	wantSel(t, "selection after the change shifts", text, 6, 7)
	text.SetSelect(4, 6) // exactly "bb", which gets replaced
	text.replaceAll("aaa\nzzzz\nccc\n")
	wantSel(t, "selection covering the change grows to the new text", text, 4, 8)
	text.SetSelect(5, 5) // strictly inside the span that changes
	text.replaceAll("aaa\nyy\nccc\n")
	wantSel(t, "caret inside the change goes to its end", text, 6, 6)
	// Identical text is a no-op.
	text.SetSelect(2, 3)
	text.replaceAll("aaa\nzzzz\nccc\n")
	wantSel(t, "no change", text, 2, 3)
}

func TestFormatBodyAndPut(t *testing.T) {
	needUnixTools(t)
	dir := t.TempDir()
	name := filepath.Join(dir, "shout.txt")
	text := makeKeyTestBody("hello\nworld\n", 6, 11) // "world"
	text.w.body.file.SetName(name)
	old := global.fmtRules
	defer func() { global.fmtRules = old }()
	global.fmtRules = defaultFmtTable()
	global.fmtRules.set(".txt", []string{"tr", "a-z", "A-Z"})

	if !text.w.formatBody([]string{"tr", "a-z", "A-Z"}) {
		t.Fatal("formatBody reported no change")
	}
	if got := text.file.String(); got != "HELLO\nWORLD\n" {
		t.Fatalf("formatted body %q", got)
	}
	wantSel(t, "selection inside the change", text, 11, 11)

	// Put formats again (idempotent here) and writes the file.
	text.file.InsertAt(0, []rune("x"))
	put(text, nil, nil, false, false, "")
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "XHELLO\nWORLD\n" {
		t.Errorf("file after Put: %q", b)
	}
	if got := text.file.String(); got != "XHELLO\nWORLD\n" {
		t.Errorf("body after Put: %q", got)
	}

	// A failing formatter leaves the text alone and does not stop Put.
	global.fmtRules.set(".txt", []string{"false"})
	text.file.InsertAt(0, []rune("y"))
	put(text, nil, nil, false, false, "")
	b, _ = os.ReadFile(name)
	if string(b) != "yXHELLO\nWORLD\n" {
		t.Errorf("file after Put with a failing formatter: %q", b)
	}

	// put off: Put does not format.
	global.fmtRules.set(".txt", []string{"tr", "a-z", "A-Z"})
	global.fmtRules.onPut = false
	text.file.InsertAt(0, []rune("z"))
	put(text, nil, nil, false, false, "")
	b, _ = os.ReadFile(name)
	if string(b) != "zyXHELLO\nWORLD\n" {
		t.Errorf("file after Put with put off: %q", b)
	}
}

func TestFormatterErrorNamesTheFile(t *testing.T) {
	needUnixTools(t)
	text := makeKeyTestBody("x\n", 0, 0)
	name := "/tmp/a.go"
	text.w.body.file.SetName(name)
	warnings = nil
	defer func() { warnings = nil }()
	bad := []string{"sh", "-c", "echo '<standard input>:2:1: expected declaration' >&2; echo '<stdin>:3: other' >&2; exit 2"}
	if text.w.formatBody(bad) {
		t.Fatal("a failing formatter changed the body")
	}
	var got string
	for _, w := range warnings {
		got += w.buf.String()
	}
	for _, want := range []string{
		"Fmt: " + name + ": sh failed; text left unchanged\n",
		name + ":2:1: expected declaration\n",
		name + ":3: other\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("warning lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "standard input") || strings.Contains(got, "exit status") {
		t.Errorf("warning still names stdin or the exit status:\n%s", got)
	}

	// A formatter that cannot start says why.
	warnings = nil
	text.w.formatBody([]string{"/nonexistent/fmt"})
	if len(warnings) == 0 || !strings.Contains(warnings[0].buf.String(), "failed: ") {
		t.Errorf("missing reason for a formatter that did not start: %v", warnings)
	}
}

func TestExpandFmtArgs(t *testing.T) {
	got := expandFmtArgs([]string{"%f", "--stdin-filepath", "%f", "--x=%f.bak", "100%%", "%d", "%"}, "/p/a.ts")
	want := []string{"%f", "--stdin-filepath", "/p/a.ts", "--x=/p/a.ts.bak", "100%", "%d", "%"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("expandFmtArgs = %q; want %q", got, want)
	}

	// The formatter sees the path.
	needUnixTools(t)
	text := makeKeyTestBody("x\n", 0, 0)
	text.w.body.file.SetName("/tmp/b.ts")
	if !text.w.formatBody([]string{"sh", "-c", `cat; echo "$0"`, "%f"}) {
		t.Fatal("no change")
	}
	if got := text.file.String(); got != "x\n/tmp/b.ts\n" {
		t.Errorf("body %q; want the path appended", got)
	}
}
