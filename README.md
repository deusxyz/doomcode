# doomcode

**English** | [Русский](README.RU.md)

A minimalist, comfortable and fast text editor for developers, flexible to configure and extend. It is a descendant of Acme: columns and windows, a tag line that serves as each window's command line, any text can be run as a command or followed as a link, and extensions are ordinary programs that talk to the editor's windows through the acme(4) file interface. On top of that come familiar keys, syntax highlighting, diagnostics and formatting.

The name comes from two inspirations: Doom Emacs, which turned a powerful old editor into a comfortable everyday tool, and the game Doom, whose small team made the impossible fast. doomcode aims to do for Acme what Doom Emacs did for Emacs. The mission is written up in [docs/00-mission.md](docs/00-mission.md) (in Russian, like the rest of `docs/`).

## What is inside

- `editor/` is the editor. It grew out of a fork of [Edwood](https://github.com/rjkroege/edwood), the Go version of Acme, with its history kept.
- `cmd/Syn` highlights syntax with tree-sitter: Go, C, JSON, shell, Rust, Python, JavaScript, TypeScript, and Markdown including code in fenced blocks. It also marks parse errors as you type.
- `cmd/Diag` shows language server diagnostics, collected by [acme-lsp](https://github.com/fhs/acme-lsp), right in the text.
- `docs/` holds specifications and design decisions, in Russian.

Features include CUA editing keys and a tmux-style `Ctrl-B` prefix for windows and columns, Shift selection, `Ctrl-/` to toggle comments, `F12` to go to a definition, formatting on save, and keys, theme and formatters configured in plain files under `~/.config/doomcode`. Everything Acme clients rely on keeps working: the editor still serves the `acme` 9P service.

## Building and running

You need Go and [plan9port](https://github.com/9fans/plan9port), which provides devdraw for the window. Special keys with modifiers, such as Shift+arrows, need the patch on the `devdraw/modkeys` branch of [deusxyz/plan9port](https://github.com/deusxyz/plan9port).

```bash
./build.sh            # bin/doomcode, bin/Syn, bin/Diag
./run.sh files...
```

## License

[MIT](LICENSE), © 2026 Igor Kozlitin. The editor code in `editor/` inherited from Edwood, plan9port and Project Serenity stays under the terms of [editor/LICENSE](editor/LICENSE) (BSD-3-Clause and MIT), and its notices are kept. Third-party Go modules are under their own licenses (MIT, BSD-2-Clause, BSD-3-Clause).
