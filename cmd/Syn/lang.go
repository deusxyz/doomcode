package main

import (
	"embed"
	"path/filepath"
	"strings"
	"unsafe"

	tree_sitter_markdown "github.com/tree-sitter-grammars/tree-sitter-markdown/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_bash "github.com/tree-sitter/tree-sitter-bash/bindings/go"
	tree_sitter_c "github.com/tree-sitter/tree-sitter-c/bindings/go"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_json "github.com/tree-sitter/tree-sitter-json/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tree_sitter_rust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tree_sitter_typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// A Language is a tree-sitter grammar with a highlight query whose capture
// names are mapped to Edwood style names by styleFor.
type Language struct {
	Name   string
	lang   *tree_sitter.Language
	query  *tree_sitter.Query
	inline *Language // for Markdown: grammar applied inside "inline" nodes
}

// The highlight queries, one file per grammar; see queries/README.md.
//
//go:embed queries/*.scm
var queryFiles embed.FS

var languages = map[string]*Language{}

// mustLang loads a grammar with the query queries/<name>.scm, or with the
// concatenation of the named query files when files are given: a later
// file's patterns win over an earlier one's, which is how the TypeScript
// query extends the JavaScript one.
func mustLang(name string, ptr unsafe.Pointer, files ...string) *Language {
	if len(files) == 0 {
		files = []string{name}
	}
	var src []byte
	for _, f := range files {
		b, err := queryFiles.ReadFile("queries/" + f + ".scm")
		if err != nil {
			panic(name + ": " + err.Error())
		}
		src = append(append(src, b...), '\n')
	}
	lang := tree_sitter.NewLanguage(ptr)
	q, qerr := tree_sitter.NewQuery(lang, string(src))
	if qerr != nil {
		panic(name + ": " + qerr.Error())
	}
	return &Language{Name: name, lang: lang, query: q}
}

func register(l *Language, suffixes ...string) {
	for _, suf := range suffixes {
		languages[suf] = l
	}
}

func init() {
	register(mustLang("go", tree_sitter_go.Language()), ".go")
	register(mustLang("c", tree_sitter_c.Language()), ".c", ".h")
	register(mustLang("json", tree_sitter_json.Language()), ".json")
	register(mustLang("bash", tree_sitter_bash.Language()), ".sh", ".bash", ".zsh")
	register(mustLang("rust", tree_sitter_rust.Language()), ".rs")
	register(mustLang("python", tree_sitter_python.Language()), ".py")
	register(mustLang("javascript", tree_sitter_javascript.Language()), ".js", ".mjs", ".cjs", ".jsx")
	register(mustLang("typescript", tree_sitter_typescript.LanguageTypescript(), "javascript", "typescript"), ".ts", ".mts", ".cts")
	register(mustLang("tsx", tree_sitter_typescript.LanguageTSX(), "javascript", "typescript"), ".tsx")
	md := mustLang("markdown", tree_sitter_markdown.Language())
	md.inline = mustLang("markdown-inline", tree_sitter_markdown.InlineLanguage())
	register(md, ".md", ".markdown")
}

// languageFor returns the language for a window name, or nil.
func languageFor(name string) *Language {
	return languages[strings.ToLower(filepath.Ext(name))]
}

// styleFor maps a tree-sitter capture name to an Edwood style name. ok is
// false for captures that are not drawn (variables, punctuation,
// operators unless -all) or carry no style.
func styleFor(capture string, all bool) (string, bool) {
	switch capture {
	case "string.special.key": // JSON object keys: set them apart from values
		return "type", true
	case "text.title":
		return "heading", true
	case "text.emphasis", "text.strong":
		return "emphasis", true
	case "text.uri", "text.reference":
		return "link", true
	case "text.literal":
		return "string", true
	}
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
	case "type", "constructor":
		return "type", true
	case "function", "method":
		return "function", true
	case "constant", "boolean", "label":
		return "constant", true
	case "preproc", "macro", "attribute":
		return "preproc", true
	case "operator":
		return "operator", all
	case "punctuation", "delimiter":
		return "punctuation", all
	case "variable", "property", "parameter", "field":
		return "variable", all
	}
	return "", false
}
