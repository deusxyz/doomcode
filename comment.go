package main

import (
	"path/filepath"
	"strings"
)

// Line comments by file type for the comment-toggle action (prefix /).
// Only single-line comment prefixes for now; see docs/03-keyboard-spec.md
// section 4 in the justcode project.

var commentPrefixes = []struct {
	suffixes string // space separated, with the dot
	prefix   string
}{
	{".go .c .h .cc .cpp .cxx .hh .hpp .m .mm .rs .js .mjs .cjs .jsx .ts .tsx .java .kt .swift .cs .scala .zig .proto", "//"},
	{".sh .bash .zsh .fish .py .rb .pl .toml .yaml .yml .rc .mk .cmake .tf .conf .ini .ps1 .r .jl .nim .ex .exs", "#"},
	{".sql .lua .hs .ada .adb .ads", "--"},
	{".scm .ss .lisp .lsp .el .cl .rkt .clj .cljs .edn", ";"},
	{".tex .sty .cls .erl .hrl", "%"},
	{".vim", "\""},
}

// commentFileNames are files recognised by name rather than suffix.
var commentFileNames = map[string]string{
	"Makefile": "#", "makefile": "#", "GNUmakefile": "#", "mkfile": "#",
	"Dockerfile": "#", "Containerfile": "#", "Vagrantfile": "#", "Rakefile": "#", "Gemfile": "#",
	"CMakeLists.txt": "#", ".gitignore": "#", ".gitattributes": "#", ".bashrc": "#", ".zshrc": "#", ".profile": "#",
}

// commentPrefix returns the line comment prefix for a file name, or "".
func commentPrefix(name string) string {
	base := filepath.Base(name)
	if p, ok := commentFileNames[base]; ok {
		return p
	}
	ext := strings.ToLower(filepath.Ext(base))
	if ext == "" {
		return ""
	}
	for _, c := range commentPrefixes {
		for _, s := range strings.Fields(c.suffixes) {
			if s == ext {
				return c.prefix
			}
		}
	}
	return ""
}

// commentToggle comments out the lines touched by the selection with the
// file's line comment prefix, or uncomments them if every non-blank line
// is already commented. Comments go at the shallowest indentation of the
// lines, as most editors do. A selection grows to whole lines; a plain
// caret stays on its line.
func (t *Text) commentToggle() {
	if t.what != Body || t.w == nil {
		return
	}
	prefix := commentPrefix(t.w.body.file.Name())
	if prefix == "" {
		warning(nil, "comment-toggle: unknown file type %q\n", t.w.body.file.Name())
		return
	}
	t.markUndo()
	t.TypeCommit()
	t.dropAnchor()

	oq0, oq1 := t.q0, t.q1
	q0 := t.lineStart(oq0)
	q1 := oq1
	if q1 > q0 && t.file.ReadC(q1-1) == '\n' {
		q1-- // a selection ending at a line start does not include that line
	}
	if q1 < q0 {
		q1 = q0
	}
	// Collect the line starts first; edits below move everything after.
	var starts []int
	for q := q0; ; {
		starts = append(starts, q)
		e := t.lineEnd(q)
		if e >= t.file.Nr() || e+1 > q1 {
			break
		}
		q = e + 1
	}

	// Decide: uncomment only if every non-blank line already starts with
	// the prefix after its indentation.
	pr := []rune(prefix)
	indentEnd := func(q int) int {
		e := t.lineEnd(q)
		for q < e && (t.file.ReadC(q) == ' ' || t.file.ReadC(q) == '\t') {
			q++
		}
		return q
	}
	hasPrefix := func(q int) bool {
		for i, r := range pr {
			if q+i >= t.file.Nr() || t.file.ReadC(q+i) != r {
				return false
			}
		}
		return true
	}
	uncomment := true
	minIndent := -1
	for _, s := range starts {
		ie := indentEnd(s)
		if ie == t.lineEnd(s) {
			continue // blank line
		}
		if !hasPrefix(ie) {
			uncomment = false
		}
		if ind := ie - s; minIndent < 0 || ind < minIndent {
			minIndent = ind
		}
	}
	if minIndent < 0 {
		return // nothing but blank lines
	}

	caret := oq0
	delta := 0 // total change before the caret's line, for a plain caret
	caretLine := t.lineStart(oq0)
	for _, s := range starts {
		s += delta
		var d int
		if uncomment {
			ie := indentEnd(s)
			if ie == t.lineEnd(s) {
				continue
			}
			n := len(pr)
			if ie+n < t.file.Nr() && t.file.ReadC(ie+n) == ' ' {
				n++ // the space we put after the prefix
			}
			t.Delete(ie, ie+n, true)
			d = -n
			if s == caretLine+delta && caret > ie {
				caret += d
				if caret < ie {
					caret = ie
				}
			}
		} else {
			if indentEnd(s) == t.lineEnd(s) {
				continue // leave blank lines alone
			}
			ins := append(append([]rune{}, pr...), ' ')
			t.file.InsertAt(s+minIndent, ins)
			d = len(ins)
			if s == caretLine+delta && caret >= s+minIndent {
				caret += d
			}
		}
		delta += d
	}

	if oq0 == oq1 {
		t.Show(caret, caret, true)
		return
	}
	end := t.lineEnd(q1 + delta)
	if end < t.file.Nr() {
		end++
	}
	t.Show(q0, end, true)
}
