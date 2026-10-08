<div align="center">

# doomcode

**English** | [Русский](README.RU.md)

[Install](#install) • [Features](#features) • [Keys](#getting-help) • [Docs](docs/) • [Roadmap](#roadmap) • [Contribute](#contribute)

[![CI](https://img.shields.io/github/actions/workflow/status/deusxyz/doomcode/doomcode.yml?branch=main&style=flat-square&label=CI&logo=github)](https://github.com/deusxyz/doomcode/actions/workflows/doomcode.yml)
![Go 1.24+](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat-square&logo=go&logoColor=white)
![Platform: macOS](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20(soon)-lightgrey?style=flat-square&logo=apple&logoColor=white)
![acme(4) compatible](https://img.shields.io/badge/acme(4)-compatible-8a2be2?style=flat-square)
[![License: MIT](https://img.shields.io/badge/license-MIT-brightgreen?style=flat-square)](LICENSE)
![Last commit](https://img.shields.io/github/last-commit/deusxyz/doomcode?style=flat-square)

![doomcode screenshot](docs/images/screenshot.png)

</div>

---

### Table of Contents
- [Introduction](#introduction)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Install](#install)
- [Roadmap](#roadmap)
- [Getting help](#getting-help)
- [Contribute](#contribute)
- [Acknowledgements](#acknowledgements)

# Introduction

> Rip and tear, until it is done.
>
> — the Doom Slayer

doomcode is a minimalist, comfortable and fast text editor for developers, flexible to configure and extend. It keeps the soul of Acme, Rob Pike's editor from Plan 9: columns of windows instead of tabs and panels, a tag line that is each window's command line, and text that is never just text, since any word can be run with the middle button or opened with the right one. Around that core it adds what a modern developer reaches for on day one: familiar keys, syntax highlighting, diagnostics and formatting.

The name comes from two inspirations. Doom Emacs took a powerful old editor and made it a comfortable everyday tool without breaking what made it powerful. The game Doom showed what a small team can do when it treats speed and simplicity as a craft. doomcode wants to do for Acme what Doom Emacs did for Emacs.

It follows a few mantras:

+ **Minimal.** Few concepts, each working everywhere. New things are added only when old ones cannot express them.
+ **Comfortable.** Familiar keys from the first minute: CUA editing, <kbd>Ctrl</kbd>+<kbd>/</kbd> to comment, <kbd>F12</kbd> to jump to a definition. The Acme mouse model stays, and the keyboard is its equal.
+ **Fast.** Instant start, no lag between a key and the screen. Heavy work, such as parsing and diagnostics, runs in separate processes that never block typing.
+ **Your config, plain text.** Keys, theme and formatters live in small text files under `~/.config/doomcode`. Anything you might want to change is configuration, not code.
+ **Extensions are just programs.** Anything that reads and writes files can extend the editor through the acme(4) file interface: `Syn`, `Diag` and acme-lsp all work that way. There is no extension language to learn.

# Features

- Acme's model intact: columns, windows, tags, mouse chords, plumbing, the `Edit` command language, `win` terminals.
- CUA keys: arrows that move the cursor, <kbd>Shift</kbd> selection, <kbd>Ctrl</kbd>+<kbd>C</kbd>/<kbd>X</kbd>/<kbd>V</kbd>/<kbd>Z</kbd>/<kbd>Y</kbd>, <kbd>Ctrl</kbd>+<kbd>S</kbd> to save, word moves with <kbd>Option</kbd> or <kbd>Ctrl</kbd>; <kbd>Cmd</kbd> twins on macOS.
- A tmux-style <kbd>Ctrl</kbd>+<kbd>B</kbd> prefix for windows and columns: move focus, new, close, zoom, reorder, list windows.
- Keyboard equivalents of the mouse: <kbd>Ctrl</kbd>+<kbd>E</kbd> executes like button 2, <kbd>Ctrl</kbd>+<kbd>O</kbd> opens like button 3; find and go to line through the tag.
- Syntax highlighting with tree-sitter for Go, C, JSON, shell, Rust, Python, JavaScript, TypeScript and Markdown, including code inside fenced blocks; parse errors marked as you type.
- Language server features through acme-lsp: diagnostics right in the text, go to definition, references, rename, completion, hover, all on keys.
- Formatting on save with any filter you like (`gofmt` by default), comments toggled for the file's language.
- Every key binding, the theme and the formatters configurable in plain files, reloaded without a restart.

# Prerequisites

+ Required:
  + Go 1.24 or newer, with a C compiler for the tree-sitter grammars
  + [plan9port](https://github.com/9fans/plan9port), for `devdraw` (the window), `rc` and `9p`
+ Optional:
  + [acme-lsp](https://github.com/fhs/acme-lsp) and a language server such as `gopls`, for diagnostics and code navigation
  + formatters for your languages (`gofmt` ships with Go)

> [!IMPORTANT]
> Special keys with modifiers (<kbd>Shift</kbd>+arrows, <kbd>Ctrl</kbd>+<kbd>Enter</kbd>, <kbd>Option</kbd>+arrows) need a small patch to `devdraw`, on the `devdraw/modkeys` branch of [deusxyz/plan9port](https://github.com/deusxyz/plan9port). Without it everything else works, those keys just arrive unmodified.

> [!NOTE]
> doomcode is developed and used on macOS. The Linux build of the patched `devdraw` is written but not yet tested.

# Install

```sh
git clone https://github.com/deusxyz/doomcode
cd doomcode
./build.sh                # bin/doomcode, bin/Syn, bin/Diag
./run.sh                  # start the editor
```

`run.sh` points the editor at plan9port's `devdraw` and gives it its own namespace, so doomcode can run next to a plan9port acme. Start syntax highlighting by running `Syn` from any tag, and diagnostics by running `acme-lsp` and then `Diag`.

# Roadmap

- [Status and plan](docs/06-status-and-plan.md): what is done, what is deferred and why.
- [Ideas](docs/08-ideas.md): where doomcode may go next, starting with an Org mode for Markdown.
- [Manual test plan](docs/test-plan.md): what still waits for a human with a keyboard.

# Getting help

The documentation lives in [docs/](docs/), in Russian for now. Where the settings live, what goes in them and what is built in: [docs/configuration.md](docs/configuration.md). The [keyboard specification](docs/03-keyboard-spec.md) lists every binding; inside the editor, run `Keys` for the bindings in effect and `Keys actions` for everything a key can do.

A few keys to start with:

| Keys | Action |
|---|---|
| <kbd>Ctrl</kbd>+<kbd>E</kbd> / <kbd>Ctrl</kbd>+<kbd>O</kbd> | execute / open the word under the cursor, like buttons 2 and 3 |
| <kbd>Ctrl</kbd>+<kbd>F</kbd>, <kbd>Ctrl</kbd>+<kbd>G</kbd> | find, go to line |
| <kbd>Ctrl</kbd>+<kbd>/</kbd> | comment or uncomment the selected lines |
| <kbd>F12</kbd>, <kbd>F2</kbd> | go to definition, rename (with acme-lsp) |
| <kbd>Ctrl</kbd>+<kbd>B</kbd> <kbd>←</kbd><kbd>→</kbd><kbd>↑</kbd><kbd>↓</kbd> | move between windows and columns |
| <kbd>Ctrl</kbd>+<kbd>B</kbd> <kbd>c</kbd> / <kbd>x</kbd> / <kbd>z</kbd> | new window / close / zoom |

Found a bug or have a question? [Open an issue](https://github.com/deusxyz/doomcode/issues).

# Contribute

doomcode is written by **Igor Kozlitin** ([@deusxyz](https://github.com/deusxyz)) together with **Claude** (Anthropic). Most of the code is written by Claude in conversation with Igor, who sets the direction, decides, and tests every change by hand; the design decisions and their reasons are recorded in [docs/](docs/).

Issues and pull requests are welcome. Before a larger change, open an issue to talk it over: the project tries to stay small, and every new feature first has to show that existing tools such as acme-lsp do not already cover it.

# Acknowledgements

- [Acme](http://acme.cat-v.org/) by Rob Pike, and [plan9port](https://github.com/9fans/plan9port) by Russ Cox, which carries Plan 9 to Unix.
- [Edwood](https://github.com/rjkroege/edwood) by Rob Kroeger and contributors: doomcode's editor started as a fork of it.
- [tree-sitter](https://tree-sitter.github.io/) and its grammars, and [acme-lsp](https://github.com/fhs/acme-lsp).
- [Doom Emacs](https://github.com/doomemacs/doomemacs) and id Software's Doom, for the inspiration.

doomcode is not affiliated with id Software, Bethesda or the Doom Emacs project.

[MIT](LICENSE), © 2026 Igor Kozlitin. The editor code in `editor/` inherited from Edwood, plan9port and Project Serenity stays under the terms of [editor/LICENSE](editor/LICENSE) (BSD-3-Clause and MIT), and its notices are kept.
