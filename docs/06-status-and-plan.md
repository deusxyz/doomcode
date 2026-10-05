# Состояние проекта и план (точка передачи)

Обновлено 2026-10-05. Этот файл — то, что нужно прочитать, чтобы продолжить работу без истории разговора. Детали — в соседних документах: `01` обзор Acme, `02` рамки и инварианты, `03` клавиатура, `04` процесс и правила кода, `05` подсветка, `90` журнал отличий от Acme.

## 1. Что это

justcode — доработанный Acme на базе форка Edwood (Go): CUA-клавиатура с префиксом `Ctrl-B` в стиле tmux, подсветка синтаксиса и разметка через файловый интерфейс acme(4), внешний подсветчик `Syn` на tree-sitter. Совместимость с расширениями VS Code вынесена за рамки. Пользователь — Игорь (deusxyz на GitHub), macOS, общение на русском, код и коммиты на английском.

## 2. Репозитории и ветки

| Что | Где | Remotes | Ветки |
|---|---|---|---|
| justcode (документы, `cmd/Syn`, скрипты) | `~/projects/justcode` | нет (локальный git) | `main` |
| Edwood (форк) | `~/projects/justcode/edwood` (вложенный репозиторий, в `.gitignore` justcode) | `origin` = git@github.com:deusxyz/edwood.git, `upstream` = rjkroege/edwood | `main` — интеграционная, всё принятое; `keys/phase1`, `style/frame` — слиты в `main` (fast-forward); `master` не используется |
| 9fans.net/go (форк) | `~/projects/9fans/go` | `origin` = deusxyz/9fans-go, `upstream` = 9fans/go | `main` = upstream; `devdraw/keys` — две правки Go-devdraw только для PR в upstream, **в работе не используется** |
| plan9port (форк) | `~/projects/plan9` | `origin` = deusxyz/plan9port, `upstream` = 9fans/plan9port | `master` = upstream b6564bd9; 75 файлов «изменены» — это подстановка пути установки скриптом `lib/moveplan9.sh`, нормально |

Пуш: ветки Edwood пушатся в `origin` после каждого среза (пользователь разрешил коммитить и пушить по ходу). justcode некуда пушить.

## 3. Сборка, запуск, тесты

```bash
cd ~/projects/justcode && ./build.sh          # bin/edwood и bin/Syn
./run.sh <файлы>                               # Edwood с правильным окружением
cd edwood && ./presub.sh                       # gofmt -s, vet, staticcheck, misspell, go test -race
go test ./cmd/Syn/                             # тесты Syn (cgo, tree-sitter)
```

- `run.sh` ставит `PLAN9=~/projects/plan9`, `PATH` с `bin/` и `$PLAN9/bin`, `NAMESPACE=/tmp/ns.edwood` (чтобы жить рядом с plan9port acme, у них одно имя сервиса) и `DEVDRAW=$PLAN9/bin/devdraw` (**C-devdraw из plan9port**; Go-devdraw отвергнут: медленный, без курсоров и колеса, Backspace/Delete неверны).
- Тестам Edwood нужен `rc` в PATH (`$PLAN9/bin`) или `acmeshell=sh`, иначе `TestRunproc`/`TestMntDecRef` падают — это окружение, не flaky.
- Клиентам (`9p`, `Syn`, `win`, `acme-lsp`) нужен тот же `NAMESPACE`. Из тега Edwood всё наследуется само.
- Проверка GUI без пользователя: скриншот окна devdraw. `winlist` (Swift, в scratchpad; при необходимости пересобрать из `CGWindowListCopyWindowInfo`) даёт window id, затем `screencapture -x -o -l<id> out.png`. Нажатия клавиш за пользователя сделать нельзя; текст и команды — через `9p write acme/N/{addr,data,ctl,style}`.
- `go vet ./...` в Edwood ругается на `file/buffer_adapter.go:53 unreachable` и свежий staticcheck на `xfid.go` SA4006 — upstream, не трогаем (правило «не чиним upstream внутри фич»).

## 4. Что сделано

### Клавиатура (Edwood `main`, спецификация `03`)
- CUA: ↑↓ по строкам с липкой колонкой, Home/End по строке, Ctrl/Cmd-A/C/X/V/Z/Y/S/K, Ctrl-E = Execute (B2), Ctrl-O = Look (B3), Ctrl-L строка, Ctrl-D слово/следующее, Ctrl-F/Ctrl-G через тег (`Look `/`:` + Esc + Ctrl-E/O), Ctrl-N New, Tab/Ctrl-B Tab сдвиг строк.
- Таблица действий `keys.go` (`Action`, `Keymap`, `ParseKey` для `C-x`, `Cmd-x`, `Left`, `F3`, `0xF800`), файл `~/.config/edwood/keys` (`key action`, `prefix key action`, `key -`), команда `Keys [reload|actions|file]`.
- Префикс `Ctrl-B` (`prefix.go`): стрелки/o/;/0–9 фокус, t тег⇄тело, : командная строка, c/%/x/& New/Newcol/Del/Delcol, z/Z/+ размер, s/S Sort/Putall, g/G начало/конец, Enter/`/` Execute/Look, Space якорь, b блок, Tab outdent. Повторный Ctrl-B держит префикс (литерала нет из-за автоповтора), Esc отменяет, курсор-ромб.
- Фокус по щелчку по умолчанию (`-b=true`), `-b=false` — под мышью; до первого щелчка под мышью; навигация переносит указатель.
- Якорь выделения (`select.go`): каретка-модель `caret`/`moveCaret`, `anchorOn/anchor` в `Text`.
- Не сделано: prefix q (номера окон), w (окно `+windows`), { } [ ] (перемещение окон), − (уменьшить), F3 (повтор поиска).

### Подсветка (Edwood `main`, спецификация `05`)
- `frame`: `Style uint8` в `frbox`, `InsertStyled`, `Restyle`, `SetStyleTable`, `StyleColours{Text, Back, Underline, Line}`; выделение скрывает цвета, сохраняет подчёркивание.
- `file.StyleTable` в `ObservableEditableBuffer` (`Styles()`), сдвиг на `inserted/deleted` (покрывает undo/redo/load), `ResetBuffer` чистит.
- `theme`: `Styles` в каждой палитре (`acme`, `vampira`, `solarizedlight`, `solarizeddark`), `StyleSet` имя→индекс (ленивый, `globals.styleSet`), имена с точкой падают на префикс.
- `Text` (`textstyle.go`): `styleIndices`, `Restyle`; стили в `fill`, `Inserted` и `setorigin` (прокрутка назад).
- Файлы окна `style` (rw; `q0 q1 name`, `clear`, `clear q0 q1 [names]`, `#`) и `changes` (ro, поток `I`/`D` в формате `event`, раздача всем, без перехвата) — `xfidstyle.go`.
- `Syn` (`cmd/Syn`): 8 языков (go c json bash rust python javascript markdown), запросы в `cmd/Syn/queries/*.scm` (go:embed), `log` + `changes`, полная перекраска через 100 мс после последней правки, запись кусками по строкам (9P режет записи), `style`/`changes` через `plan9/client` (библиотека acme знает только свои имена файлов). Флаги `-v`, `-all`, `-delay`.

## 5. План (приоритет сверху вниз)

1. **Инкрементальный разбор в `Syn`.** Сейчас каждая пауза = `ReadAll body` + полный `Parse` + полный `clear`+отрезки. Сделать: держать локальную копию текста (`[]byte` + индекс рун→байты), применять сообщения `changes` (`I q0 q1 0 n text`, `D q0 q1 0 0`; при `n=0` для длинных вставок текст читать через `addr`/`xdata`) к копии и к дереву через `tree.Edit(InputEdit)`, парсить с `oldTree`, и писать только изменившиеся отрезки: `clear q0 q1` + отрезки затронутого диапазона (объединение изменённых диапазонов из `tree.ChangedRanges(old)`). Сверка целостности: раз в N правок или при расхождении длины — полная перечитка. Тест: последовательность правок даёт тот же результат, что полный разбор.
2. **Файл `theme`** (`$HOME/.config/edwood/theme`, как `keys`): строки `style <имя> fg=#rrggbb bg=#rrggbb underline line=#rrggbb`, плюс переопределение палитры (`tag.back=…`, `text.text=…`) — по возможности; применяется поверх выбранной `-palette`; команда `Theme reload`. Реализация: парсер в `theme`, `StyleSet` пересоздаётся, все `Text` → `Restyle` видимого.
3. **Диагностика LSP через `style`**: не трогая acme-lsp upstream, проверить, работает ли он с Edwood+`NAMESPACE`; затем небольшой мост `Diag` (в justcode) или патч форка acme-lsp: писать `error`/`warning`/`info`/`hint` в `style` с `clear q0 q1 error warning info hint`. Диагностика по умолчанию — подчёркивание + слабый фон (уже в теме).
4. **Остатки клавиатуры**: F3/`Ctrl-B n` повтор поиска, окно `+windows` (`Ctrl-B w`), перемещение окон (`{ } [ ]`), номера окон (`Ctrl-B q`), уменьшение (`−`).
5. **Фаза 2 клавиатуры**: модификаторы спецклавиш через патч C-devdraw из plan9port (форк есть): коды `0xF200 | mods<<5 | key`, включение `DEVDRAW_MODKEYS=1`; тогда Shift+стрелки, Ctrl+стрелки, Ctrl+Enter, Ctrl+Backspace. Файлы `src/cmd/devdraw/mac-screen.m` (`doCommandBySelector`), `x11-screen.c`; пересборка `cd src/cmd/devdraw && mk install`.
6. Позже: инкрементальный `Restyle` в `frame` (сейчас полный пересчёт диапазона), TypeScript и Markdown в fenced-блоках в `Syn`, upstream-PR в Edwood/9fans/go/plan9port.

## 6. Правила, о которых легко забыть

- Ветки по разделам спецификации (`keys/*`, `style/*`, `fmt/*`, `devdraw/*`), маленькие коммиты с тестами, `main` только fast-forward после ручной проверки пользователем; коммиты в стиле Go с ссылкой на раздел спецификации; подпись `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.
- Инварианты: совместимость acme(4) (новые файлы только добавляются), мышиная модель Acme не меняется, стили — слой поверх текста (`body` чистый), привязки клавиш только через `keys.go`, без cgo в ядре Edwood (в `Syn` можно), без новых зависимостей без обсуждения, путь модуля `github.com/rjkroege/edwood` не менять.
- Каждое отличие от Acme — строка в `docs/90-differences-from-acme.md`.
- Перед диагностикой «что сломано» проверять, какой бинарник реально запущен (`lsof -p <pid> | grep txt`), и сверять с окружением пользователя (`zsh -lic 'which -a devdraw'`). Два цикла ушли на патчи Go-devdraw, который попал в работу только из-за моего PATH.
- Остановиться и спросить пользователя: при изменении принятых решений, при нарушении инвариантов, при необходимости его действий на GitHub, при неясном UX-выборе. Остальное решать самостоятельно и подтверждать тестами и скриншотами.
