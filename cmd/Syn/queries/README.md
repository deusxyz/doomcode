# Highlight queries

`*.scm` are tree-sitter highlight queries, one per grammar. They are the
grammars' own `queries/highlights.scm` files (MIT licensed by their
respective authors: the tree-sitter authors for go, c, json, bash, rust,
python, javascript; nvim-treesitter for markdown), with small changes:

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

Capture names are mapped to Edwood style names by `styleFor` in lang.go.
