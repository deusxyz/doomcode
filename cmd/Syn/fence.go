package main

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// Fenced code blocks in Markdown. The block grammar leaves the content of
// ```go … ``` opaque; Syn highlights it with the language the info string
// names, and as a plain literal when it names none or one Syn does not
// know. The content is parsed in place with included ranges, so the
// indentation or "> " a block carries inside a list or a quote
// (block_continuation nodes) is left out and offsets stay those of the
// document. Parse errors inside fences are not marked: snippets are often
// fragments, and marking them would cover READMEs in red.

// fenceAliases are info-string names that are not file suffixes.
var fenceAliases = map[string]string{
	"golang": ".go", "javascript": ".js", "node": ".js", "typescript": ".ts",
	"python": ".py", "python3": ".py", "rust": ".rs", "shell": ".sh", "console": ".sh",
	"jsonc": ".json", "cpp": "", "markdown": "", "md": "",
}

// fenceLanguage returns the language an info string names, or nil. Only
// the first word counts ("go title=x"); a pandoc-style "{.go}" works too.
// Markdown inside Markdown is not highlighted.
func fenceLanguage(info string) *Language {
	f := strings.Fields(info)
	if len(f) == 0 {
		return nil
	}
	name := strings.ToLower(strings.Trim(f[0], "{}."))
	if suf, ok := fenceAliases[name]; ok {
		if suf == "" {
			return nil
		}
		return languages[suf]
	}
	l := languages["."+name]
	if l != nil && l.inline != nil {
		return nil
	}
	return l
}

// fenceSpans returns the spans for the content of every fenced code block
// under root that intersects [lo,hi), and the byte range those blocks
// cover, which the caller must rewrite as a whole: an edit on one line of
// a block, or of its info string, can change the colours of all of it.
// ok is false when no block intersects.
func fenceSpans(root *tree_sitter.Node, text []byte, lo, hi uint, all bool) (spans []span, blo, bhi uint, ok bool) {
	walk(root, func(n *tree_sitter.Node) {
		if n.Kind() != "fenced_code_block" || n.EndByte() < lo || n.StartByte() > hi {
			return
		}
		if !ok || n.StartByte() < blo {
			blo = n.StartByte()
		}
		if !ok || n.EndByte() > bhi {
			bhi = n.EndByte()
		}
		ok = true
		var info string
		var content *tree_sitter.Node
		for i := uint(0); i < n.ChildCount(); i++ {
			c := n.Child(i)
			switch c.Kind() {
			case "info_string":
				info = string(text[c.StartByte():c.EndByte()])
			case "code_fence_content":
				content = c
			}
		}
		if content == nil || content.StartByte() >= content.EndByte() {
			return
		}
		if lang := fenceLanguage(info); lang != nil {
			spans = append(spans, embeddedSpans(lang, content, text, all)...)
			return
		}
		spans = append(spans, span{content.StartByte(), content.EndByte(), "string", 0})
	})
	return spans, blo, bhi, ok
}

// embeddedSpans parses the text of content, less its block_continuation
// children, with lang and returns the highlight spans in document offsets.
func embeddedSpans(lang *Language, content *tree_sitter.Node, text []byte, all bool) []span {
	var ranges []tree_sitter.Range
	start, startPt := content.StartByte(), content.StartPosition()
	for i := uint(0); i < content.ChildCount(); i++ {
		c := content.Child(i)
		if c.Kind() != "block_continuation" {
			continue
		}
		if c.StartByte() > start {
			ranges = append(ranges, tree_sitter.Range{StartByte: start, EndByte: c.StartByte(), StartPoint: startPt, EndPoint: c.StartPosition()})
		}
		start, startPt = c.EndByte(), c.EndPosition()
	}
	if content.EndByte() > start {
		ranges = append(ranges, tree_sitter.Range{StartByte: start, EndByte: content.EndByte(), StartPoint: startPt, EndPoint: content.EndPosition()})
	}
	if len(ranges) == 0 {
		return nil
	}
	p := tree_sitter.NewParser()
	defer p.Close()
	if p.SetLanguage(lang.lang) != nil || p.SetIncludedRanges(ranges) != nil {
		return nil
	}
	t := p.Parse(text, nil)
	if t == nil {
		return nil
	}
	defer t.Close()
	var out []span
	for _, sp := range captureSpans(lang, t.RootNode(), text, 0, all) {
		// Clip to the content: a capture may reach into an excluded
		// continuation (a multi-line string in a list), which is fine,
		// but never past the block.
		if sp.start < content.StartByte() {
			sp.start = content.StartByte()
		}
		if sp.end > content.EndByte() {
			sp.end = content.EndByte()
		}
		if sp.start < sp.end {
			out = append(out, sp)
		}
	}
	return out
}
