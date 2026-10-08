package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Formatting on Put: a table from file suffix (or exact file name) to a
// filter command that reads the body on standard input and writes the
// formatted text on standard output. Put runs the filter first when a
// rule matches; the Fmt command runs it on demand.
//
// Location: $DOOMCODE_FMT if set, otherwise $XDG_CONFIG_HOME/doomcode/fmt,
// otherwise $HOME/.config/doomcode/fmt. The file is applied on top of the
// defaults (defaultFmtRules), so it only needs to list changes:
//
//	.go gofmt -s           # suffix, then the command and its arguments
//	.rs rustfmt --emit stdout
//	Makefile -             # a lone - removes the rule
//	put off                # do not format on Put; Fmt still works
//
// # starts a comment; blank lines are ignored.
//
// See docs/07-format-spec.md in the justcode project.

type fmtRule struct {
	pattern string   // ".go" or "Makefile"
	argv    []string // command and arguments
}

type fmtTable struct {
	rules []fmtRule
	onPut bool
}

var defaultFmtRules = []fmtRule{
	{".go", []string{"gofmt"}},
}

func defaultFmtTable() *fmtTable {
	t := &fmtTable{onPut: true}
	for _, r := range defaultFmtRules {
		t.set(r.pattern, r.argv)
	}
	return t
}

func (t *fmtTable) set(pattern string, argv []string) {
	for i, r := range t.rules {
		if r.pattern == pattern {
			t.rules[i].argv = argv
			return
		}
	}
	t.rules = append(t.rules, fmtRule{pattern, argv})
}

func (t *fmtTable) remove(pattern string) {
	for i, r := range t.rules {
		if r.pattern == pattern {
			t.rules = append(t.rules[:i], t.rules[i+1:]...)
			return
		}
	}
}

// ruleFor returns the rule for a file name: an exact base name wins over
// a suffix. nil when there is none.
func (t *fmtTable) ruleFor(name string) *fmtRule {
	if t == nil {
		return nil
	}
	base := filepath.Base(name)
	ext := strings.ToLower(filepath.Ext(base))
	var bySuffix *fmtRule
	for i := range t.rules {
		r := &t.rules[i]
		switch {
		case r.pattern == base:
			return r
		case strings.HasPrefix(r.pattern, ".") && strings.ToLower(r.pattern) == ext:
			bySuffix = r
		}
	}
	return bySuffix
}

// doc lists the rules the way the file spells them.
func (t *fmtTable) doc() string {
	var sb strings.Builder
	for _, r := range t.rules {
		fmt.Fprintf(&sb, "\t%s %s\n", r.pattern, strings.Join(r.argv, " "))
	}
	if !t.onPut {
		sb.WriteString("\tput off\n")
	}
	return sb.String()
}

func fmtFilePath() string { return configFilePath("fmt") }

// loadFmtText applies the lines read from r to t. Bad lines are reported
// in errs and skipped. n is the number of lines applied.
func loadFmtText(t *fmtTable, r io.Reader) (n int, errs []error) {
	sc := bufio.NewScanner(r)
	for lineno := 1; sc.Scan(); lineno++ {
		line := sc.Text()
		if i := strings.Index(line, "#"); i >= 0 {
			line = line[:i]
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "put" {
			switch {
			case len(fields) == 2 && fields[1] == "on":
				t.onPut = true
			case len(fields) == 2 && fields[1] == "off":
				t.onPut = false
			default:
				errs = append(errs, fmt.Errorf("line %d: want \"put on\" or \"put off\", got %q", lineno, strings.TrimSpace(line)))
				continue
			}
			n++
			continue
		}
		if len(fields) < 2 {
			errs = append(errs, fmt.Errorf("line %d: want \"suffix command [args]\", got %q", lineno, strings.TrimSpace(line)))
			continue
		}
		if fields[1] == "-" && len(fields) == 2 {
			t.remove(fields[0])
		} else {
			t.set(fields[0], fields[1:])
		}
		n++
	}
	if err := sc.Err(); err != nil {
		errs = append(errs, err)
	}
	return n, errs
}

// loadFmtFile builds the table from the defaults and the file at path. A
// missing file is not an error.
func loadFmtFile(path string) (*fmtTable, int, []error) {
	t := defaultFmtTable()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return t, 0, nil
		}
		return t, 0, []error{err}
	}
	defer f.Close()
	n, errs := loadFmtText(t, f)
	return t, n, errs
}

// The Fmt command:
//
//	Fmt            format the window's body now with its rule
//	Fmt cmd args   format it with this filter instead
//	Fmt reload     re-read the fmt file on top of the defaults
//	Fmt rules      list the rules and the fmt file path
//	Fmt file       print the fmt file path
func fmtCmd(et *Text, _ *Text, argt *Text, _, _ bool, arg string) {
	if r, _ := getarg(argt, false, true); r != "" {
		arg = r
	}
	arg = strings.TrimSpace(arg)
	path := fmtFilePath()
	switch arg {
	case "reload":
		t, n, errs := loadFmtFile(path)
		for _, err := range errs {
			warning(nil, "Fmt: %s: %v\n", path, err)
		}
		global.fmtRules = t
		warning(nil, "Fmt: %s: %d line(s) applied on top of defaults\n", path, n)
		return
	case "rules":
		warning(nil, "Fmt: rules (file %s):\n%s", path, global.fmtRules.doc())
		return
	case "file":
		warning(nil, "%s\n", path)
		return
	}
	if et == nil || et.w == nil || et.w.body.file.IsDir() {
		return
	}
	w := et.w
	var argv []string
	if arg != "" {
		argv = strings.Fields(arg)
	} else {
		r := global.fmtRules.ruleFor(w.body.file.Name())
		if r == nil {
			warning(nil, "Fmt: no rule for %s; see Fmt rules\n", w.body.file.Name())
			return
		}
		argv = r.argv
	}
	w.formatBody(argv)
}
