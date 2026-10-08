package main

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// Parse errors. tree-sitter recovers from bad input by wrapping what it
// cannot place in ERROR nodes and by inventing MISSING tokens that take no
// space. Syn marks both with errorStyle, which the theme draws like
// "error" (dotted names fall back to their prefix) while staying separate
// from the "error" marks Diag writes: each program clears only its own
// names.
//
// While the user types, tree-sitter may wrap a large part of the file in
// one ERROR node. Marking all of it would flash the screen, so a long
// ERROR node only gets its first token marked: that is where the parser
// gave up, which is what the user needs to see.

const errorStyle = "error.syntax"

// maxErrorSpan is the longest ERROR node, in bytes, marked as a whole;
// it must also lie on one line.
const maxErrorSpan = 64

// errorPattern sorts error spans after every query pattern at the same
// start, so that they replace the syntax colour of what they cover.
const errorPattern = ^uint(0)

// errorSpans returns the parse error marks of the tree under root whose
// node intersects [lo,hi), in byte offsets of text.
func errorSpans(root *tree_sitter.Node, text []byte, lo, hi uint) []span {
	var out []span
	var visit func(n *tree_sitter.Node)
	visit = func(n *tree_sitter.Node) {
		if !n.HasError() || n.EndByte() < lo || n.StartByte() > hi {
			return
		}
		switch {
		case n.IsMissing():
			if s, e, ok := tokenBefore(text, n.StartByte()); ok {
				out = append(out, span{s, e, errorStyle, errorPattern})
			}
			return
		case n.IsError():
			if s, e, ok := errorExtent(n, text); ok {
				out = append(out, span{s, e, errorStyle, errorPattern})
			}
			// An ERROR node can hold further errors (MISSING tokens);
			// they are inside what is marked or past its first token,
			// and the first mark is enough.
			return
		}
		for i := uint(0); i < n.ChildCount(); i++ {
			if c := n.Child(i); c != nil {
				visit(c)
			}
		}
	}
	visit(root)
	return out
}

// errorExtent is what to mark for an ERROR node: all of it when short and
// on one line, else its first token.
func errorExtent(n *tree_sitter.Node, text []byte) (uint, uint, bool) {
	s, e := n.StartByte(), n.EndByte()
	if s < e && e-s <= maxErrorSpan && !hasNewline(text[s:e]) {
		return s, e, true
	}
	// The first leaf with some text is the first token.
	for c := n; c != nil; {
		if c.ChildCount() == 0 {
			if c.StartByte() < c.EndByte() {
				return c.StartByte(), c.EndByte(), true
			}
			break
		}
		var next *tree_sitter.Node
		for i := uint(0); i < c.ChildCount(); i++ {
			if k := c.Child(i); k != nil && k.StartByte() < k.EndByte() {
				next = k
				break
			}
		}
		c = next
	}
	if s < e { // no leaf: mark up to the end of the first line
		end := s
		for end < e && text[end] != '\n' {
			end++
		}
		return s, end, end > s
	}
	return 0, 0, false
}

// tokenBefore returns the run of non-space bytes that ends at or before
// b, skipping white space: where a missing token should have followed.
func tokenBefore(text []byte, b uint) (uint, uint, bool) {
	e := b
	for e > 0 && isSpace(text[e-1]) {
		e--
	}
	s := e
	for s > 0 && !isSpace(text[s-1]) {
		s--
	}
	return s, e, s < e
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

func hasNewline(b []byte) bool {
	for _, c := range b {
		if c == '\n' {
			return true
		}
	}
	return false
}
