package main

import (
	"strings"
	"testing"
)

// spanAt reports the style of the first span covering byte offset b.
func spanAt(spans []span, b uint) string {
	style := ""
	for _, sp := range spans {
		if sp.start <= b && b < sp.end {
			style = sp.style // later spans win, as in the editor
		}
	}
	return style
}

func TestHighlightGo(t *testing.T) {
	src := "package main\n\n// greet says hello.\nfunc greet(name string) string {\n\treturn \"hi \" + name + fmt.Sprint(42, nil)\n}\n"
	spans := highlight(languages[".go"], []byte(src), false)
	if len(spans) == 0 {
		t.Fatal("no spans")
	}
	idx := func(s string) uint { return uint(strings.Index(src, s)) }
	for _, tc := range []struct {
		at   string
		want string
	}{
		{"package", "keyword"},
		{"main", "preproc"},
		{"// greet", "comment"},
		{"func", "keyword"},
		{"greet(", "function"},
		{"string)", "type"},
		{"return", "keyword"},
		{"\"hi \"", "string"},
		{"Sprint", "function"},
		{"42", "number"},
		{"nil", "constant"},
	} {
		if got := spanAt(spans, idx(tc.at)); got != tc.want {
			t.Errorf("%q: style %q; want %q", tc.at, got, tc.want)
		}
	}
	// Variables and operators are left out unless -all.
	if got := spanAt(spans, idx("name +")); got != "" {
		t.Errorf("variable styled as %q without -all", got)
	}
	if got := spanAt(highlight(languages[".go"], []byte(src), true), idx("name +")); got != "variable" {
		t.Errorf("variable with -all: %q", got)
	}
}

func TestHighlightMarkdown(t *testing.T) {
	src := "# Title here\n\nSome *emphasis* and **strong** text with `code` and a [link](http://x.y).\n\n- item\n\n```go\nfunc f() {}\n```\n"
	spans := highlight(languages[".md"], []byte(src), false)
	idx := func(s string) uint { return uint(strings.Index(src, s)) }
	for _, tc := range []struct {
		at   string
		want string
	}{
		{"Title", "heading"},
		{"emphasis", "emphasis"},
		{"strong", "emphasis"},
		{"code", "string"},
		{"link]", "link"},
		{"http://x.y", "link"},
		{"func f", "string"}, // fenced code block
	} {
		if got := spanAt(spans, idx(tc.at)); got != tc.want {
			t.Errorf("%q: style %q; want %q", tc.at, got, tc.want)
		}
	}
	if got := spanAt(spans, idx("Some")); got != "" {
		t.Errorf("plain text styled as %q", got)
	}
}

func TestStyleTextRuneOffsets(t *testing.T) {
	// Cyrillic letters are two bytes each: offsets must come out in runes.
	src := []byte("// привет\nx := 1\n")
	spans := []span{{start: 0, end: uint(len("// привет")), style: "comment"}, {start: uint(len("// привет\nx := ")), end: uint(len("// привет\nx := 1")), style: "number"}}
	got := styleText(src, spans)
	want := "clear\n0 9 comment\n15 16 number\n"
	if got != want {
		t.Errorf("styleText = %q; want %q", got, want)
	}
	// Spans beyond the text are clamped, empty ones dropped.
	got = styleText([]byte("ab"), []span{{start: 1, end: 10, style: "x"}, {start: 2, end: 2, style: "y"}})
	if got != "clear\n1 2 x\n" {
		t.Errorf("clamped styleText = %q", got)
	}
}

func TestLanguageFor(t *testing.T) {
	for name, want := range map[string]string{
		"/a/b/main.go": "go", "README.MD": "markdown", "x.markdown": "markdown", "x.txt": "", "Makefile": "", "/dir/": "",
	} {
		l := languageFor(name)
		got := ""
		if l != nil {
			got = l.Name
		}
		if got != want {
			t.Errorf("languageFor(%q) = %q; want %q", name, got, want)
		}
	}
}

func TestStyleFor(t *testing.T) {
	for _, tc := range []struct {
		capture string
		all     bool
		want    string
		ok      bool
	}{
		{"keyword", false, "keyword", true},
		{"function.method", false, "function", true},
		{"constant.builtin", false, "constant", true},
		{"text.title", false, "heading", true},
		{"text.uri", false, "link", true},
		{"variable", false, "variable", false},
		{"variable", true, "variable", true},
		{"punctuation.special", false, "punctuation", false},
		{"none", false, "", false},
	} {
		got, ok := styleFor(tc.capture, tc.all)
		if got != tc.want || ok != tc.ok {
			t.Errorf("styleFor(%q, %v) = %q,%v; want %q,%v", tc.capture, tc.all, got, ok, tc.want, tc.ok)
		}
	}
}

func TestHighlightOtherLanguages(t *testing.T) {
	type check struct{ at, want string }
	for _, tc := range []struct {
		suffix string
		src    string
		checks []check
	}{
		{".c", "#include <stdio.h>\n// comment\nint main(void) { return printf(\"hi %d\", 42); }\n", []check{
			{"#include", "keyword"}, {"<stdio.h>", "string"}, {"// comment", "comment"}, {"int ", "type"},
			{"main", "function"}, {"return", "keyword"}, {"\"hi", "string"}, {"42", "number"}, {"printf", "function"}}},
		{".json", "{\"key\": [1, true, null, \"v\"]}\n", []check{
			{"\"key\"", "type"}, {"1", "number"}, {"true", "constant"}, {"null", "constant"}, {"\"v\"", "string"}}},
		{".sh", "#!/bin/sh\n# c\nfor f in *.go; do echo \"$f\" > out; done\n", []check{
			{"# c", "comment"}, {"for", "keyword"}, {"do", "keyword"}, {"echo", "function"}, {"\"$f\"", "string"}}},
		{".rs", "// c\nfn main() -> u32 { let x: Vec<i32> = Vec::new(); return 7; }\n", []check{
			{"// c", "comment"}, {"fn", "keyword"}, {"main", "function"}, {"u32", "type"}, {"let", "keyword"},
			{"Vec<", "type"}, {"new", "function"}, {"7", "number"}}},
		{".py", "# c\ndef f(x):\n    return str(x) + \"s\" + str(1.5)\nclass Foo: pass\n", []check{
			{"# c", "comment"}, {"def", "keyword"}, {"f(", "function"}, {"return", "keyword"}, {"\"s\"", "string"},
			{"1.5", "number"}, {"class", "keyword"}, {"Foo", "type"}}},
		{".js", "// c\nfunction f(a) { return a + 1 + `t${a}`; }\nconst x = new Map();\n", []check{
			{"// c", "comment"}, {"function", "keyword"}, {"f(", "function"}, {"return", "keyword"}, {"1 +", "number"},
			{"`t", "string"}, {"const", "keyword"}, {"Map", "type"}}},
	} {
		lang := languageFor("x" + tc.suffix)
		if lang == nil {
			t.Errorf("%s: no language", tc.suffix)
			continue
		}
		spans := highlight(lang, []byte(tc.src), false)
		for _, c := range tc.checks {
			i := strings.Index(tc.src, c.at)
			if i < 0 {
				t.Fatalf("%s: %q not in source", tc.suffix, c.at)
			}
			if got := spanAt(spans, uint(i)); got != c.want {
				t.Errorf("%s %q: style %q; want %q", tc.suffix, c.at, got, c.want)
			}
		}
	}
}
