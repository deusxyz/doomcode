package main

import (
	"strings"
	"testing"
)

func TestLoadFmtText(t *testing.T) {
	tab := defaultFmtTable()
	src := `# comment
.rs rustfmt --emit stdout   # trailing comment
Makefile -
.go -
.py black -q -
put off
bad
put maybe
`
	n, errs := loadFmtText(tab, strings.NewReader(src))
	if n != 5 || len(errs) != 2 {
		t.Fatalf("applied %d lines, %d errors (%v); want 5 and 2", n, len(errs), errs)
	}
	if r := tab.ruleFor("/x/a.go"); r != nil {
		t.Errorf(".go rule still present: %v", r)
	}
	if r := tab.ruleFor("/x/lib.RS"); r == nil || strings.Join(r.argv, " ") != "rustfmt --emit stdout" {
		t.Errorf(".rs rule = %v", r)
	}
	if r := tab.ruleFor("/x/t.py"); r == nil || len(r.argv) != 3 {
		t.Errorf(".py rule = %v", r)
	}
	if tab.onPut {
		t.Error("put off not applied")
	}
	// An exact name wins over a suffix.
	tab.set("go.mod", []string{"true"})
	tab.set(".mod", []string{"false"})
	if r := tab.ruleFor("/x/go.mod"); r == nil || r.argv[0] != "true" {
		t.Errorf("go.mod rule = %v; want the exact-name rule", r)
	}
	if !strings.Contains(tab.doc(), "\t.rs rustfmt --emit stdout\n") || !strings.Contains(tab.doc(), "put off") {
		t.Errorf("doc:\n%s", tab.doc())
	}
}

func TestDefaultFmtTable(t *testing.T) {
	tab := defaultFmtTable()
	if r := tab.ruleFor("main.go"); r == nil || r.argv[0] != "gofmt" || !tab.onPut {
		t.Fatalf("default table: %v onPut=%v", tab.rules, tab.onPut)
	}
}
