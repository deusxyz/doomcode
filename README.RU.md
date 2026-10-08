# doomcode

[English](README.md) | **Русский**

Минималистичный, удобный и быстрый текстовый редактор для разработчиков, с гибкой настройкой и расширением. Наследник Acme: колонки и окна, тег как командная строка, любой текст — команда или ссылка, расширения — обычные программы, работающие с окнами через файловый интерфейс acme(4). Поверх этого — привычная клавиатура, подсветка синтаксиса, диагностика и форматирование.

Миссия и происхождение имени — [docs/00-mission.md](docs/00-mission.md).

## Что внутри

- `editor/` — редактор. Вырос из форка [Edwood](https://github.com/rjkroege/edwood) (Go-версия Acme), история сохранена.
- `cmd/Syn` — подсветка синтаксиса на tree-sitter: Go, C, JSON, shell, Rust, Python, JavaScript, TypeScript, Markdown с кодом в fenced-блоках; ошибки разбора при наборе.
- `cmd/Diag` — диагностика языковых серверов (через [acme-lsp](https://github.com/fhs/acme-lsp)) прямо в тексте.
- `docs/` — спецификации и решения (на русском).

Возможности: CUA-клавиатура и префикс `Ctrl-B` в стиле tmux для окон и колонок, выделение с Shift, комментирование `Ctrl-/`, переход к определению `F12`, форматирование при сохранении, настраиваемые клавиши, тема и форматтеры в `~/.config/doomcode`.

## Сборка и запуск

Нужны Go и [plan9port](https://github.com/9fans/plan9port) (devdraw для окна; модификаторы спецклавиш требуют патча из ветки `devdraw/modkeys` форка [deusxyz/plan9port](https://github.com/deusxyz/plan9port)).

```bash
./build.sh            # bin/doomcode, bin/Syn, bin/Diag
./run.sh файлы...
```

## Лицензия

[MIT](LICENSE), © 2026 Igor Kozlitin. Код редактора в `editor/`, унаследованный от Edwood, plan9port и Project Serenity, остаётся под условиями [editor/LICENSE](editor/LICENSE) (BSD-3-Clause и MIT); их уведомления сохраняются. Сторонние Go-модули — под своими лицензиями (MIT, BSD-2-Clause, BSD-3-Clause).
