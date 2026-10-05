package main

import (
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// styleModel mimics the editor's StyleTable for a style write: a style
// name per rune, updated by "clear", "clear q0 q1", and "q0 q1 name"
// lines, shifted by insertions and deletions like the editor does.
type styleModel struct {
	styles []string
}

func (m *styleModel) apply(write string) {
	for _, line := range strings.Split(strings.TrimRight(write, "\n"), "\n") {
		f := strings.Fields(line)
		switch {
		case len(f) == 0:
		case f[0] == "clear" && len(f) == 1:
			for i := range m.styles {
				m.styles[i] = ""
			}
		case f[0] == "clear":
			q0, _ := strconv.Atoi(f[1])
			q1, _ := strconv.Atoi(f[2])
			only := map[string]bool{}
			for _, n := range f[3:] {
				only[n] = true
			}
			for i := q0; i < q1 && i < len(m.styles); i++ {
				if len(only) == 0 || only[m.styles[i]] {
					m.styles[i] = ""
				}
			}
		default:
			q0, _ := strconv.Atoi(f[0])
			q1, _ := strconv.Atoi(f[1])
			for i := q0; i < q1 && i < len(m.styles); i++ {
				m.styles[i] = f[2]
			}
		}
	}
}

func (m *styleModel) insert(q0 int, nr int) {
	ins := make([]string, nr)
	// Insert strictly inside a span inherits it, like the editor.
	if q0 > 0 && q0 < len(m.styles) && m.styles[q0-1] != "" && m.styles[q0-1] == m.styles[q0] {
		for i := range ins {
			ins[i] = m.styles[q0-1]
		}
	}
	m.styles = append(m.styles[:q0], append(ins, m.styles[q0:]...)...)
}

func (m *styleModel) delete(q0, q1 int) {
	m.styles = append(m.styles[:q0], m.styles[q1:]...)
}

// fullStyles is the reference: a complete highlight of text as rune styles.
func fullStyles(lang *Language, text []byte) []string {
	out := make([]string, utf8.RuneCount(text))
	var m styleModel
	m.styles = out
	m.apply(styleText(text, highlight(lang, text, false)))
	return m.styles
}

func runeIndex(text []byte, q int) int {
	b := 0
	for i := 0; i < q; i++ {
		_, s := utf8.DecodeRune(text[b:])
		b += s
	}
	return b
}

// step applies one change to the document, the reference text and the
// style model, flushes, and checks the model against a full highlight.
func step(t *testing.T, d *document, text *[]byte, m *styleModel, c change) {
	t.Helper()
	if !d.apply(c) {
		t.Fatalf("apply(%+v) rejected", c)
	}
	// Reference text.
	b0 := runeIndex(*text, c.Q0)
	if c.Insert {
		*text = append((*text)[:b0], append(append([]byte(nil), c.Text...), (*text)[b0:]...)...)
		m.insert(c.Q0, c.Q1-c.Q0)
	} else {
		b1 := runeIndex(*text, c.Q1)
		*text = append((*text)[:b0], (*text)[b1:]...)
		m.delete(c.Q0, c.Q1)
	}
	if string(d.text) != string(*text) {
		t.Fatalf("document text %q; want %q", d.text, *text)
	}
	write := d.flush(false, false)
	m.apply(write)
	want := fullStyles(d.lang, *text)
	for i := range want {
		if m.styles[i] != want[i] {
			t.Fatalf("after %+v: rune %d (%q) styled %q; want %q\nwrite was:\n%s", c, i, string([]rune(string(*text))[i]), m.styles[i], want[i], write)
		}
	}
}

func ins(q0 int, s string) change {
	return change{Insert: true, Q0: q0, Q1: q0 + utf8.RuneCountInString(s), Text: []byte(s), HasText: true}
}
func del(q0, q1 int) change { return change{Q0: q0, Q1: q1} }

func TestDocumentIncrementalGo(t *testing.T) {
	src := "package main\n\n// greet says hello\nfunc greet(name string) string {\n\treturn \"hi \" + name\n}\n"
	d, err := newDocument(languages[".go"], []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	defer d.close()
	text := []byte(src)
	m := &styleModel{styles: make([]string, utf8.RuneCount(text))}
	m.apply(d.flush(true, false))

	// Type a word into the comment: the comment grows at its end.
	q := utf8.RuneCountInString("package main\n\n// greet says hello")
	step(t, d, &text, m, ins(q, " politely"))
	// Turn the string into a longer one with a Cyrillic word (multibyte).
	q = utf8.RuneCountInString(string(text[:strings.Index(string(text), "hi ")])) + 3
	step(t, d, &text, m, ins(q, "привет "))
	// Delete the keyword "return " so the parse changes structurally.
	i := strings.Index(string(text), "return ")
	q = utf8.RuneCount(text[:i])
	step(t, d, &text, m, del(q, q+7))
	// Put it back.
	step(t, d, &text, m, ins(q, "return "))
	// Start a new comment at the end: unstyled text becomes a comment.
	q = utf8.RuneCount(text)
	step(t, d, &text, m, ins(q, "// tail"))
	// Delete a chunk spanning several tokens.
	i = strings.Index(string(text), "func greet")
	q = utf8.RuneCount(text[:i])
	step(t, d, &text, m, del(q, q+10))
	// Type the function header back one character at a time.
	for k, r := range "func greet" {
		step(t, d, &text, m, ins(q+k, string(r)))
	}
}

func TestDocumentIncrementalMarkdown(t *testing.T) {
	src := "# Title\n\nSome *emphasis* here.\n\n- item one\n"
	d, err := newDocument(languages[".md"], []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	defer d.close()
	text := []byte(src)
	m := &styleModel{styles: make([]string, utf8.RuneCount(text))}
	m.apply(d.flush(true, false))

	q := utf8.RuneCountInString("# Title")
	step(t, d, &text, m, ins(q, " and more"))
	q = utf8.RuneCountInString("# Title and more\n\nSome *emphasis")
	step(t, d, &text, m, ins(q, " added"))
	q = utf8.RuneCount(text)
	step(t, d, &text, m, ins(q, "\n[link](http://x.y)\n"))
	q = utf8.RuneCountInString("# Title and more\n\nSome *")
	step(t, d, &text, m, del(q-1, q)) // remove the opening *: emphasis disappears
}

func TestDocumentApplyRejectsBadChanges(t *testing.T) {
	d, err := newDocument(languages[".go"], []byte("x := 1\n"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.close()
	for _, c := range []change{
		{Insert: true, Q0: 100, Q1: 101, Text: []byte("a"), HasText: true},
		{Insert: true, Q0: 0, Q1: 5, Text: []byte("ab"), HasText: true}, // length mismatch
		{Insert: true, Q0: 0, Q1: 300, HasText: false},                  // text omitted
		{Q0: 5, Q1: 3},
		{Q0: 0, Q1: 99},
	} {
		if d.apply(c) {
			t.Errorf("apply(%+v) accepted", c)
		}
	}
	if d.flush(false, false) != "" {
		t.Error("flush without edits wrote something")
	}
}

func TestParseChanges(t *testing.T) {
	buf := []byte("KI0 2 0 2 ab\nKD3 5 0 0 \nFI10 12 0 2 \xd0\xbf\xd1\x80\nKI0 300 0 0 \nKI7 8 0 1 ")
	cs, rest, err := parseChanges(buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) != 4 {
		t.Fatalf("got %d changes: %+v", len(cs), cs)
	}
	if !cs[0].Insert || cs[0].Q0 != 0 || cs[0].Q1 != 2 || string(cs[0].Text) != "ab" || !cs[0].HasText {
		t.Errorf("c0 = %+v", cs[0])
	}
	if cs[1].Insert || cs[1].Q0 != 3 || cs[1].Q1 != 5 {
		t.Errorf("c1 = %+v", cs[1])
	}
	if string(cs[2].Text) != "пр" || cs[2].Q1 != 12 {
		t.Errorf("c2 = %+v", cs[2])
	}
	if !cs[3].Insert || cs[3].HasText {
		t.Errorf("long insert without text: %+v", cs[3])
	}
	if string(rest) != "KI7 8 0 1 " {
		t.Errorf("rest %q", rest)
	}
	// The remainder completes with the next read.
	cs, rest, err = parseChanges(append(rest, "x\n"...))
	if err != nil || len(cs) != 1 || string(cs[0].Text) != "x" || len(rest) != 0 {
		t.Errorf("completed: %+v %q %v", cs, rest, err)
	}
	if _, _, err := parseChanges([]byte("KQ1 2 0 0 \n")); err == nil {
		t.Error("unknown type accepted")
	}
}
