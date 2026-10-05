package main

import (
	"bufio"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

// A diagnostic is one line of acme-lsp's /LSP/Diagnostics window:
//
//	/abs/path/file.go:12.5,12.9: message
//
// with 1-based lines and columns (LSP counts columns in UTF-16 units;
// they are used as rune offsets here, which is exact for ASCII lines).
type diagnostic struct {
	File        string
	Line0, Col0 int // start, 1-based
	Line1, Col1 int // end, 1-based, exclusive column
	Message     string
	Style       string
}

var diagLine = regexp.MustCompile(`^(.+?):(\d+)\.(\d+),(\d+)\.(\d+): (.*)$`)

// parseDiagnostics reads the window body and groups diagnostics by file.
func parseDiagnostics(body string, defaultStyle string) map[string][]diagnostic {
	out := map[string][]diagnostic{}
	sc := bufio.NewScanner(strings.NewReader(body))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		m := diagLine.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		n := func(s string) int { v, _ := strconv.Atoi(s); return v }
		d := diagnostic{
			File: m[1], Line0: n(m[2]), Col0: n(m[3]), Line1: n(m[4]), Col1: n(m[5]),
			Message: m[6], Style: severityStyle(m[6], defaultStyle),
		}
		out[d.File] = append(out[d.File], d)
	}
	return out
}

// severityStyle guesses a style from the message: the Diagnostics window
// does not carry the LSP severity. Everything is an error unless the
// message says otherwise.
func severityStyle(msg, def string) string {
	l := strings.ToLower(msg)
	switch {
	case strings.HasPrefix(l, "warning"), strings.Contains(l, "deprecated"), strings.Contains(l, "should "):
		return "warning"
	case strings.HasPrefix(l, "hint"), strings.HasPrefix(l, "info"):
		return "hint"
	}
	return def
}

// diagStyles are the style names Diag owns in a window; they are the ones
// its clear removes, so syntax colours from Syn are left alone.
var diagStyles = []string{"error", "warning", "info", "hint"}

// styleWrite converts the diagnostics for one file into a style write
// against body: a clear of the diagnostic styles over the whole text,
// then one span per diagnostic. An empty range is widened to the word
// under it, or one rune.
func styleWrite(body []byte, diags []diagnostic) string {
	lines := lineStarts(body)
	total := utf8.RuneCount(body)
	var sb strings.Builder
	fmt.Fprintf(&sb, "clear 0 %d %s\n", total, strings.Join(diagStyles, " "))
	type sp struct {
		q0, q1 int
		style  string
	}
	var spans []sp
	for _, d := range diags {
		q0, ok0 := offset(lines, body, d.Line0, d.Col0)
		q1, ok1 := offset(lines, body, d.Line1, d.Col1)
		if !ok0 {
			continue
		}
		if !ok1 || q1 < q0 {
			q1 = q0
		}
		if q1 == q0 {
			q1 = wordEnd(body, q0)
			if q1 == q0 && q0 < total {
				q1 = q0 + 1
			}
		}
		if q1 > total {
			q1 = total
		}
		if q0 < q1 {
			spans = append(spans, sp{q0, q1, d.Style})
		}
	}
	// Errors last so they win where ranges overlap.
	rank := map[string]int{"hint": 0, "info": 1, "warning": 2, "error": 3}
	sort.SliceStable(spans, func(i, j int) bool { return rank[spans[i].style] < rank[spans[j].style] })
	for _, s := range spans {
		fmt.Fprintf(&sb, "%d %d %s\n", s.q0, s.q1, s.style)
	}
	return sb.String()
}

// lineStarts returns the rune offset of the start of each line.
func lineStarts(body []byte) []int {
	starts := []int{0}
	q := 0
	for _, r := range string(body) {
		q++
		if r == '\n' {
			starts = append(starts, q)
		}
	}
	return starts
}

// offset converts a 1-based line and column into a rune offset, clamped
// to the line. ok is false when the line does not exist.
func offset(lines []int, body []byte, line, col int) (int, bool) {
	if line < 1 || line > len(lines) {
		return 0, false
	}
	start := lines[line-1]
	end := utf8.RuneCount(body)
	if line < len(lines) {
		end = lines[line] - 1 // before the newline
	}
	q := start + col - 1
	if q < start {
		q = start
	}
	if q > end {
		q = end
	}
	return q, true
}

// wordEnd returns the end of the identifier starting at rune offset q.
func wordEnd(body []byte, q int) int {
	rs := []rune(string(body))
	e := q
	for e < len(rs) && (rs[e] == '_' || rs[e] >= '0' && rs[e] <= '9' || rs[e] >= 'a' && rs[e] <= 'z' || rs[e] >= 'A' && rs[e] <= 'Z' || rs[e] > 127) {
		e++
	}
	return e
}
