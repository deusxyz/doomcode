# Highlight queries

`*.scm` are tree-sitter highlight queries, one per grammar. They are the
grammars' own `queries/highlights.scm` files, MIT licensed with the
grammars: the tree-sitter authors for go, c, json, bash, rust, python,
javascript and typescript; tree-sitter-grammars for markdown and
markdown-inline (whose files note they come from nvim-treesitter). Small
changes:

- `go.scm`: `package` identifier and `import` declarations captured as
  `@preproc`.
- `markdown.scm`: heading markers captured as `@text.title`.
- `javascript.scm`: the `#is-not? local` predicates removed; the Go
  bindings do not evaluate them.
- `json.scm`: the object-key pattern moved after `(string) @string` so
  that keys win (later patterns win where captures overlap).
- `rust.scm`: integer and float literals captured as `@number` instead of
  `@constant.builtin`.
- `python.scm`: class names captured as `@type`.
- `typescript.scm`: the grammar's file unchanged. As upstream intends, it
  is loaded after `javascript.scm` for both TypeScript and TSX
  (`mustLang` concatenates them), so its patterns extend and override the
  JavaScript ones.
- `markdown.scm`: fenced code blocks no longer captured whole; only the
  fences and the info string are, and the content is highlighted with
  the language the info string names (fence.go).

Capture names are mapped to Edwood style names by `styleFor` in lang.go.
