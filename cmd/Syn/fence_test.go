package main

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestHighlightTypeScript(t *testing.T) {
	src := "interface Point { x: number }\nexport function len(p: Point): number {\n\t// distance\n\treturn Math.sqrt(p.x * p.x)\n}\nconst s = `hi`\n"
	for _, suf := range []string{".ts", ".tsx"} {
		spans := highlight(languages[suf], []byte(src), false)
		idx := func(s string) uint { return uint(strings.Index(src, s)) }
		for _, tc := range []struct{ at, want string }{
			{"interface", "keyword"},
			{"Point {", "type"},
			{"number }", "type"},
			{"export", "keyword"},
			{"function", "keyword"},
			{"len(", "function"},
			{"// distance", "comment"},
			{"return", "keyword"},
			{"sqrt", "function"},
			{"const", "keyword"},
			{"`hi`", "string"},
		} {
			if got := spanAt(spans, idx(tc.at)); got != tc.want {
				t.Errorf("%s %q: style %q; want %q", suf, tc.at, got, tc.want)
			}
		}
	}
}

func TestFenceLanguage(t *testing.T) {
	for info, want := range map[string]string{
		"go": "go", "Go title=x": "go", "golang": "go", "{.go}": "go", "ts": "typescript",
		"typescript": "typescript", "tsx": "tsx", "js": "javascript", "python": "python",
		"sh": "bash", "shell": "bash", "rust": "rust", "json": "json",
		"": "", "text": "", "md": "", "markdown": "", "cpp": "",
	} {
		got := ""
		if l := fenceLanguage(info); l != nil {
			got = l.Name
		}
		if got != want {
			t.Errorf("fenceLanguage(%q) = %q; want %q", info, got, want)
		}
	}
}

func TestHighlightFences(t *testing.T) {
	src := "# T\n\n```ts\nlet x: number = 1\n```\n\n```\nplain words\n```\n\n- item\n  ```go\n  func f() {}\n  ```\n\n> ```py\n> def g(): pass\n> ```\n\n```go\nx := := broken\n```\n"
	spans := highlight(languages[".md"], []byte(src), false)
	idx := func(s string) uint { return uint(strings.Index(src, s)) }
	for _, tc := range []struct{ at, want string }{
		{"let", "keyword"},
		{"number =", "type"},
		{"plain", "string"}, // no language: a literal, as before
		{"func f", "keyword"},
		{"def g", "keyword"},
		{"g():", "function"},
		{"> def", ""}, // the quote marker is not code
	} {
		if got := spanAt(spans, idx(tc.at)); got != tc.want {
			t.Errorf("%q: style %q; want %q", tc.at, got, tc.want)
		}
	}
	for _, sp := range spans {
		if sp.style == errorStyle {
			t.Errorf("parse error marked inside a fence: %q", src[sp.start:sp.end])
		}
	}
}

func TestDocumentIncrementalFences(t *testing.T) {
	src := "# Doc\n\nText.\n\n```go\nfunc f() { return }\n```\n\nMore *text*.\n"
	d, err := newDocument(languages[".md"], []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	defer d.close()
	text := []byte(src)
	m := &styleModel{styles: make([]string, utf8.RuneCount(text))}
	m.apply(d.flush(true, false))
	at := func(s string) int { return utf8.RuneCount(text[:strings.Index(string(text), s)]) }

	// Change the language: go -> py, the whole block recolours.
	q := at("go\nfunc")
	step(t, d, &text, m, del(q, q+2))
	step(t, d, &text, m, ins(q, "py"))
	// Type inside the block.
	q = at("return")
	step(t, d, &text, m, ins(q, "x = 1; "))
	// Remove the language: the block becomes a literal.
	q = at("py\n")
	step(t, d, &text, m, del(q, q+2))
	// Name a language back, typing it.
	for k, r := range "ts" {
		step(t, d, &text, m, ins(q+k, string(r)))
	}
	// Open a new fence above: everything after changes structure.
	q = at("Text.")
	step(t, d, &text, m, ins(q, "```js\nlet a\n```\n"))
}
