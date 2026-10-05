package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// A document is Syn's copy of one window body together with its parse
// tree. Edits from the changes file are applied to the copy and to the
// tree; a flush reparses incrementally and produces a style write that
// covers only what may have changed.
type document struct {
	lang   *Language
	text   []byte
	parser *tree_sitter.Parser
	inline *tree_sitter.Parser // Markdown inline grammar, if any
	tree   *tree_sitter.Tree

	// dirty is the byte range, in current text coordinates, that edits
	// have touched since the last flush; valid when hasDirty.
	dirty    [2]uint
	hasDirty bool
	// edits counts applied changes, for the periodic consistency check.
	edits int
}

func newDocument(lang *Language, text []byte) (*document, error) {
	d := &document{lang: lang, text: append([]byte(nil), text...)}
	d.parser = tree_sitter.NewParser()
	if err := d.parser.SetLanguage(lang.lang); err != nil {
		d.parser.Close()
		return nil, err
	}
	if lang.inline != nil {
		d.inline = tree_sitter.NewParser()
		if err := d.inline.SetLanguage(lang.inline.lang); err != nil {
			d.close()
			return nil, err
		}
	}
	d.tree = d.parser.Parse(d.text, nil)
	if d.tree == nil {
		d.close()
		return nil, fmt.Errorf("parse failed")
	}
	return d, nil
}

func (d *document) close() {
	if d.tree != nil {
		d.tree.Close()
	}
	if d.parser != nil {
		d.parser.Close()
	}
	if d.inline != nil {
		d.inline.Close()
	}
}

// Runes is the length of the text in runes.
func (d *document) Runes() int { return utf8.RuneCount(d.text) }

// byteOffset converts a rune offset into a byte offset and the
// tree-sitter point (row, column in bytes) at that place. A rune offset
// past the end clamps to the end.
func (d *document) byteOffset(q int) (uint, tree_sitter.Point) {
	var b uint
	var pt tree_sitter.Point
	for i := 0; i < q && int(b) < len(d.text); i++ {
		r, size := utf8.DecodeRune(d.text[b:])
		if r == '\n' {
			pt.Row++
			pt.Column = 0
		} else {
			pt.Column += uint(size)
		}
		b += uint(size)
	}
	return b, pt
}

// pointAt returns the tree-sitter point of byte offset b.
func (d *document) pointAt(b uint) tree_sitter.Point {
	var pt tree_sitter.Point
	for _, c := range d.text[:b] {
		if c == '\n' {
			pt.Row++
			pt.Column = 0
		} else {
			pt.Column++
		}
	}
	return pt
}

// apply records one change from the changes file in the text copy and
// the tree. It reports false when the change does not fit the copy, in
// which case the caller should resync from the body.
func (d *document) apply(c change) bool {
	n := d.Runes()
	if c.Q0 < 0 || c.Q0 > c.Q1 || c.Q1 > n+ifInsert(c) {
		return false
	}
	if c.Insert {
		if !c.HasText || utf8.RuneCount(c.Text) != c.Q1-c.Q0 {
			return false
		}
		if c.Q0 > n {
			return false
		}
		start, startPt := d.byteOffset(c.Q0)
		d.edit(start, start, start+uint(len(c.Text)), startPt, c.Text)
		return true
	}
	// Deletion of [Q0,Q1).
	start, startPt := d.byteOffset(c.Q0)
	end, _ := d.byteOffset(c.Q1)
	if end < start {
		return false
	}
	d.edit(start, end, start, startPt, nil)
	return true
}

// ifInsert allows an insertion's Q1 to lie beyond the current end.
func ifInsert(c change) int {
	if c.Insert {
		return c.Q1 - c.Q0
	}
	return 0
}

// edit replaces text[start:oldEnd) with repl (len(repl) == newEnd-start),
// tells the tree, and widens the dirty range.
func (d *document) edit(start, oldEnd, newEnd uint, startPt tree_sitter.Point, repl []byte) {
	oldEndPt := d.pointAt(oldEnd)
	nt := make([]byte, 0, len(d.text)-int(oldEnd-start)+len(repl))
	nt = append(nt, d.text[:start]...)
	nt = append(nt, repl...)
	nt = append(nt, d.text[oldEnd:]...)
	d.text = nt
	newEndPt := d.pointAt(newEnd)
	d.tree.Edit(&tree_sitter.InputEdit{
		StartByte: start, OldEndByte: oldEnd, NewEndByte: newEnd,
		StartPosition: startPt, OldEndPosition: oldEndPt, NewEndPosition: newEndPt,
	})
	d.edits++

	delta := int(newEnd) - int(oldEnd)
	if !d.hasDirty {
		d.dirty = [2]uint{start, newEnd}
		d.hasDirty = true
		return
	}
	lo, hi := d.dirty[0], d.dirty[1]
	switch {
	case oldEnd < lo: // entirely before the dirty range: it shifts
		lo = start
		hi = uint(int(hi) + delta)
	case start > hi: // entirely after: extend to cover it
		hi = newEnd
	default: // overlapping or adjacent
		if start < lo {
			lo = start
		}
		hi = uint(int(hi) + delta)
		if newEnd > hi {
			hi = newEnd
		}
	}
	if hi > uint(len(d.text)) {
		hi = uint(len(d.text))
	}
	if lo > hi {
		lo = hi
	}
	d.dirty = [2]uint{lo, hi}
}

// flush reparses after the applied edits and returns the style write
// that brings the editor up to date: a "clear q0 q1" for the affected
// rune range followed by the spans inside it. It returns "" when nothing
// has changed. full forces a complete "clear" plus all spans.
func (d *document) flush(full, all bool) string {
	if !full && !d.hasDirty {
		return ""
	}
	old := d.tree
	nt := d.parser.Parse(d.text, old)
	if nt == nil {
		return ""
	}
	d.tree = nt

	if full {
		old.Close()
		d.hasDirty = false
		spans := d.allSpans(all)
		return styleText(d.text, spans)
	}

	// The region to rewrite: what the edits touched, what the parse says
	// changed structurally, widened to whole captures (and whole Markdown
	// inline nodes) that intersect it.
	lo, hi := d.dirty[0], d.dirty[1]
	for _, r := range nt.ChangedRanges(old) {
		if r.StartByte < lo {
			lo = r.StartByte
		}
		if r.EndByte > hi {
			hi = r.EndByte
		}
	}
	old.Close()
	d.hasDirty = false
	if hi > uint(len(d.text)) {
		hi = uint(len(d.text))
	}
	// Grow to the enclosing lines: tokens rarely cross them, and this
	// catches a token whose extent changed without a structural change.
	for lo > 0 && d.text[lo-1] != '\n' {
		lo--
	}
	for hi < uint(len(d.text)) && d.text[hi] != '\n' {
		hi++
	}

	spans, lo, hi := d.regionSpans(lo, hi, all)
	tr := newRuneCounter(d.text)
	q0, q1 := tr.at(int(lo)), tr.at(int(hi))
	var sb strings.Builder
	sb.WriteString(clearLine(q0, q1))
	tr = newRuneCounter(d.text)
	for _, sp := range spans {
		a, b := tr.at(int(sp.start)), tr.at(int(sp.end))
		if a < b {
			fmt.Fprintf(&sb, "%d %d %s\n", a, b, sp.style)
		}
	}
	return sb.String()
}

// allSpans highlights the whole document.
func (d *document) allSpans(all bool) []span {
	root := d.tree.RootNode()
	spans := captureSpans(d.lang, root, d.text, 0, all)
	if d.inline != nil {
		walk(root, func(n *tree_sitter.Node) {
			if n.Kind() == "inline" {
				spans = append(spans, d.inlineSpans(n, all)...)
			}
		})
	}
	sortSpans(spans)
	return spans
}

// regionSpans returns the spans that intersect [lo,hi), widening the
// region so that every returned span lies entirely inside it.
func (d *document) regionSpans(lo, hi uint, all bool) ([]span, uint, uint) {
	root := d.tree.RootNode()
	names := d.lang.query.CaptureNames()
	qc := tree_sitter.NewQueryCursor()
	defer qc.Close()
	qc.SetByteRange(lo, hi)
	var spans []span
	matches := qc.Matches(d.lang.query, root, d.text)
	for m := matches.Next(); m != nil; m = matches.Next() {
		for _, c := range m.Captures {
			style, ok := styleFor(names[c.Index], all)
			if !ok {
				continue
			}
			s, e := c.Node.StartByte(), c.Node.EndByte()
			if s < e && e > lo && s < hi {
				spans = append(spans, span{s, e, style, m.PatternIndex})
			}
		}
	}
	if d.inline != nil {
		walk(root, func(n *tree_sitter.Node) {
			if n.Kind() == "inline" && n.EndByte() > lo && n.StartByte() < hi {
				spans = append(spans, d.inlineSpans(n, all)...)
			}
		})
	}
	for _, sp := range spans {
		if sp.start < lo {
			lo = sp.start
		}
		if sp.end > hi {
			hi = sp.end
		}
	}
	sortSpans(spans)
	return spans, lo, hi
}

// inlineSpans runs the inline grammar over a Markdown inline node.
func (d *document) inlineSpans(n *tree_sitter.Node, all bool) []span {
	s, e := n.StartByte(), n.EndByte()
	sub := d.text[s:e]
	t := d.inline.Parse(sub, nil)
	if t == nil {
		return nil
	}
	defer t.Close()
	return captureSpans(d.lang.inline, t.RootNode(), sub, s, all)
}

func sortSpans(spans []span) {
	sort.SliceStable(spans, func(i, j int) bool {
		if spans[i].start != spans[j].start {
			return spans[i].start < spans[j].start
		}
		return spans[i].pattern < spans[j].pattern
	})
}
