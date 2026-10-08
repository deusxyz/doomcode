package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// marked returns the text of every rune range styled errorStyle after a
// full highlight of src.
func marked(t *testing.T, suffix, src string) []string {
	t.Helper()
	styles := fullStyles(languages[suffix], []byte(src))
	runes := []rune(src)
	var out []string
	for i := 0; i < len(styles); i++ {
		if styles[i] != errorStyle {
			continue
		}
		j := i
		for j < len(styles) && styles[j] == errorStyle {
			j++
		}
		out = append(out, string(runes[i:j]))
		i = j
	}
	return out
}

func TestNoErrorsInValidCode(t *testing.T) {
	src := "package main\n\nfunc f() int {\n\treturn 1\n}\n"
	if got := marked(t, ".go", src); len(got) != 0 {
		t.Errorf("valid Go marked as errors: %q", got)
	}
}

func TestErrorMarksBadToken(t *testing.T) {
	// A Cyrillic "с" makes "funс" an identifier where a declaration
	// should start: the bad token is marked, not the whole rest.
	src := "package main\n\nfunс f(st string) error {\n\tfor len(st) > 0 {\n\t\tst = st[1:]\n\t}\n\treturn nil\n}\n"
	got := marked(t, ".go", src)
	if len(got) == 0 {
		t.Fatal("no error marked")
	}
	if !strings.Contains(strings.Join(got, "|"), "funс") {
		t.Errorf("marks %q do not include the bad token", got)
	}
	for _, g := range got {
		if strings.Contains(g, "\n") {
			t.Errorf("a mark spans lines: %q", g)
		}
	}
}

func TestErrorMarksMissingToken(t *testing.T) {
	// A missing closing parenthesis: tree-sitter inserts a MISSING ")",
	// which has no width; the token before it is marked instead.
	src := "package main\n\nfunc f() {\n\tg(1, 2\n}\n"
	got := marked(t, ".go", src)
	if len(got) == 0 {
		t.Fatal("no error marked")
	}
	for _, g := range got {
		if utf8.RuneCountInString(g) > maxErrorSpan || strings.Contains(g, "\n") {
			t.Errorf("mark too large: %q", g)
		}
	}
}

func TestSyntaxClearCoversErrors(t *testing.T) {
	if !strings.Contains(clearLine(0, 1), " "+errorStyle) {
		t.Errorf("Syn's clear does not name %s: %q", errorStyle, clearLine(0, 1))
	}
	if strings.Contains(clearLine(0, 1), " error ") || strings.HasSuffix(strings.TrimSpace(clearLine(0, 1)), " error") {
		t.Errorf("Syn's clear names Diag's \"error\": %q", clearLine(0, 1))
	}
}

func TestIncrementalErrorAppearsAndGoes(t *testing.T) {
	src := "package main\n\nfunc f() {\n\tg(1, 2)\n}\n"
	d, err := newDocument(languages[".go"], []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	defer d.close()
	text := []byte(src)
	m := &styleModel{styles: make([]string, utf8.RuneCount(text))}
	m.apply(d.flush(true, false))
	// Break the keyword, then fix it, one rune at a time.
	i := strings.Index(string(text), "func")
	q := utf8.RuneCount(text[:i]) + 3
	step(t, d, &text, m, del(q, q+1))
	step(t, d, &text, m, ins(q, "с")) // Cyrillic
	hasErr := false
	for _, s := range m.styles {
		hasErr = hasErr || s == errorStyle
	}
	if !hasErr {
		t.Error("no error mark after breaking the keyword")
	}
	step(t, d, &text, m, del(q, q+1))
	step(t, d, &text, m, ins(q, "c"))
	for i, s := range m.styles {
		if s == errorStyle {
			t.Fatalf("error mark left at rune %d after the fix", i)
		}
	}
	// Drop the closing parenthesis and put it back.
	i = strings.Index(string(text), ")\n}")
	q = utf8.RuneCount(text[:i])
	step(t, d, &text, m, del(q, q+1))
	step(t, d, &text, m, ins(q, ")"))
}
