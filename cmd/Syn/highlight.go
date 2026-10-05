package main

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// A span is a styled range in byte offsets of the text. pattern is the
// index of the query pattern that produced it: where spans start at the
// same offset, a later pattern wins, as in tree-sitter's own highlighter.
type span struct {
	start, end uint
	style      string
	pattern    uint
}

// highlight parses text with lang and returns the styled spans, in byte
// offsets, sorted by start and then by query pattern. Edwood's style
// table lets a later span replace what it covers, so an inner span wins
// over an enclosing one and, at the same start, a later pattern wins.
func highlight(lang *Language, text []byte, all bool) []span {
	parser := tree_sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(lang.lang); err != nil {
		return nil
	}
	tree := parser.Parse(text, nil)
	if tree == nil {
		return nil
	}
	defer tree.Close()
	root := tree.RootNode()

	spans := captureSpans(lang, root, text, 0, all)

	if lang.inline != nil {
		// Markdown: the block grammar leaves "inline" nodes for the
		// inline grammar. Parse each separately and offset the result.
		inlineParser := tree_sitter.NewParser()
		defer inlineParser.Close()
		if err := inlineParser.SetLanguage(lang.inline.lang); err == nil {
			walk(root, func(n *tree_sitter.Node) {
				if n.Kind() != "inline" {
					return
				}
				s, e := n.StartByte(), n.EndByte()
				sub := text[s:e]
				t := inlineParser.Parse(sub, nil)
				if t == nil {
					return
				}
				spans = append(spans, captureSpans(lang.inline, t.RootNode(), sub, s, all)...)
				t.Close()
			})
		}
	}
	sort.SliceStable(spans, func(i, j int) bool {
		if spans[i].start != spans[j].start {
			return spans[i].start < spans[j].start
		}
		return spans[i].pattern < spans[j].pattern
	})
	return spans
}

// captureSpans runs lang's query over node and converts the captures to
// spans, adding base to their offsets.
func captureSpans(lang *Language, node *tree_sitter.Node, text []byte, base uint, all bool) []span {
	names := lang.query.CaptureNames()
	qc := tree_sitter.NewQueryCursor()
	defer qc.Close()
	var out []span
	matches := qc.Matches(lang.query, node, text)
	for m := matches.Next(); m != nil; m = matches.Next() {
		for _, c := range m.Captures {
			style, ok := styleFor(names[c.Index], all)
			if !ok {
				continue
			}
			s, e := c.Node.StartByte(), c.Node.EndByte()
			if s < e {
				out = append(out, span{s + base, e + base, style, m.PatternIndex})
			}
		}
	}
	return out
}

// walk visits every node of the tree in document order.
func walk(n *tree_sitter.Node, f func(*tree_sitter.Node)) {
	f(n)
	for i := uint(0); i < n.ChildCount(); i++ {
		if c := n.Child(i); c != nil {
			walk(c, f)
		}
	}
}

// styleText renders spans as the contents of a style file write: a clear
// followed by one "q0 q1 name" line per span, with the byte offsets of the
// spans converted to rune offsets in text.
func styleText(text []byte, spans []span) string {
	var sb strings.Builder
	sb.WriteString("clear\n")
	// One sweep over the text converts byte offsets to rune offsets; the
	// spans are sorted by start, the ends are looked up with a second
	// cursor that may lag behind.
	toRune := newRuneCounter(text)
	for _, sp := range spans {
		q0 := toRune.at(int(sp.start))
		q1 := toRune.at(int(sp.end))
		if q0 < q1 {
			fmt.Fprintf(&sb, "%d %d %s\n", q0, q1, sp.style)
		}
	}
	return sb.String()
}

// runeCounter converts byte offsets of text to rune offsets, caching
// progress so that monotone or nearly monotone queries are cheap.
type runeCounter struct {
	text []byte
	b, r int // runes before byte offset b
}

func newRuneCounter(text []byte) *runeCounter { return &runeCounter{text: text} }

func (rc *runeCounter) at(b int) int {
	if b > len(rc.text) {
		b = len(rc.text)
	}
	if b < rc.b {
		rc.b, rc.r = 0, 0
	}
	rc.r += utf8.RuneCount(rc.text[rc.b:b])
	rc.b = b
	return rc.r
}
