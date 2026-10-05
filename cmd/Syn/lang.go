package main

import (
	"path/filepath"
	"strings"
	"unsafe"

	tree_sitter_markdown "github.com/tree-sitter-grammars/tree-sitter-markdown/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// A Language is a tree-sitter grammar with a highlight query whose capture
// names are mapped to Edwood style names by styleFor.
type Language struct {
	Name   string
	lang   *tree_sitter.Language
	query  *tree_sitter.Query
	inline *Language // for Markdown: grammar applied inside "inline" nodes
}

var languages = map[string]*Language{}

func mustLang(name string, ptr unsafe.Pointer, query string) *Language {
	lang := tree_sitter.NewLanguage(ptr)
	q, qerr := tree_sitter.NewQuery(lang, query)
	if qerr != nil {
		panic(name + ": " + qerr.Error())
	}
	return &Language{Name: name, lang: lang, query: q}
}

func init() {
	golang := mustLang("go", tree_sitter_go.Language(), goHighlights)
	md := mustLang("markdown", tree_sitter_markdown.Language(), markdownHighlights)
	md.inline = mustLang("markdown-inline", tree_sitter_markdown.InlineLanguage(), markdownInlineHighlights)
	for _, suf := range []string{".go"} {
		languages[suf] = golang
	}
	for _, suf := range []string{".md", ".markdown"} {
		languages[suf] = md
	}
}

// languageFor returns the language for a window name, or nil.
func languageFor(name string) *Language {
	return languages[strings.ToLower(filepath.Ext(name))]
}

// styleFor maps a tree-sitter capture name to an Edwood style name. ok is
// false for captures that are not drawn (variables, punctuation,
// operators unless -all) or carry no style.
func styleFor(capture string, all bool) (string, bool) {
	head := capture
	if i := strings.IndexByte(capture, '.'); i >= 0 {
		head = capture[:i]
	}
	switch head {
	case "comment":
		return "comment", true
	case "keyword", "include", "repeat", "conditional":
		return "keyword", true
	case "string", "escape", "character":
		return "string", true
	case "number", "float":
		return "number", true
	case "type":
		return "type", true
	case "function", "method", "constructor":
		return "function", true
	case "constant", "boolean":
		return "constant", true
	case "preproc", "macro", "attribute":
		return "preproc", true
	case "operator":
		return "operator", all
	case "punctuation":
		return "punctuation", all
	case "variable", "property", "parameter", "field":
		return "variable", all
	case "text":
		switch capture {
		case "text.title":
			return "heading", true
		case "text.emphasis", "text.strong":
			return "emphasis", true
		case "text.uri", "text.reference":
			return "link", true
		case "text.literal":
			return "string", true
		}
	}
	return "", false
}

// Queries adapted from the grammars' own queries/highlights.scm files
// (tree-sitter-go: MIT, The tree-sitter authors; tree-sitter-markdown:
// MIT, from nvim-treesitter). Captures are the conventional names.

const goHighlights = `
(call_expression function: (identifier) @function)
(call_expression function: (selector_expression field: (field_identifier) @function.method))
(function_declaration name: (identifier) @function)
(method_declaration name: (field_identifier) @function.method)
(type_identifier) @type
(field_identifier) @property
(identifier) @variable
(package_clause (package_identifier) @preproc)
(import_declaration) @preproc
[ "--" "-" "-=" ":=" "!" "!=" "..." "*" "*=" "/" "/=" "&" "&&" "&=" "%" "%=" "^" "^="
  "+" "++" "+=" "<-" "<" "<<" "<<=" "<=" "=" "==" ">" ">=" ">>" ">>=" "|" "|=" "||" "~" ] @operator
[ "break" "case" "chan" "const" "continue" "default" "defer" "else" "fallthrough" "for"
  "func" "go" "goto" "if" "import" "interface" "map" "package" "range" "return" "select"
  "struct" "switch" "type" "var" ] @keyword
[ (interpreted_string_literal) (raw_string_literal) (rune_literal) ] @string
(escape_sequence) @escape
[ (int_literal) (float_literal) (imaginary_literal) ] @number
[ (true) (false) (nil) (iota) ] @constant.builtin
(comment) @comment
`

const markdownHighlights = `
(atx_heading (inline) @text.title)
(setext_heading (paragraph) @text.title)
[ (atx_h1_marker) (atx_h2_marker) (atx_h3_marker) (atx_h4_marker) (atx_h5_marker) (atx_h6_marker)
  (setext_h1_underline) (setext_h2_underline) ] @text.title
[ (link_title) (indented_code_block) (fenced_code_block) ] @text.literal
[ (link_destination) ] @text.uri
[ (link_label) ] @text.reference
[ (list_marker_plus) (list_marker_minus) (list_marker_star) (list_marker_dot) (list_marker_parenthesis)
  (thematic_break) (block_quote_marker) ] @punctuation.special
`

const markdownInlineHighlights = `
[ (code_span) (link_title) ] @text.literal
(emphasis) @text.emphasis
(strong_emphasis) @text.strong
[ (link_destination) (uri_autolink) ] @text.uri
[ (link_label) (link_text) (image_description) ] @text.reference
`
