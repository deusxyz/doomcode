# justcode

Доработанный Acme на базе форка Edwood (Go). Обсуждение и документы — на русском; код, комментарии и коммиты — на английском.

## Где что

- `docs/` — спецификации и решения, читать перед работой над разделом: `01` обзор Acme, `02` рамки и инварианты, `03` клавиатура, `04` процесс и правила кода, `05` подсветка (файл `style`), `90` журнал отличий от Acme.
- `edwood/` — клон Edwood, remote `upstream` = rjkroege/edwood. Код правим только здесь, в фичевых ветках (`keys/*`, `style/*`, `fmt/*`, `devdraw/*`).
- `~/projects/9fans/go` — клон форка 9fans-go (origin = deusxyz/9fans-go, upstream = 9fans/go), ветка `devdraw/keys`: правки Go-devdraw только для upstream; в работе Go-devdraw НЕ используется (медленный, без курсоров/колеса, неверные Backspace/Delete).
- Рабочий devdraw — C-версия plan9port `$PLAN9/bin/devdraw`; фаза 2 (модификаторы) — патч plan9port в форке.
- `cmd/Syn` — подсветчик на tree-sitter (cgo допустим: внешняя программа), модуль `justcode` в корне.
- `bin/edwood`, `bin/Syn` — сборки для ручной проверки, не в git.
- `~/projects/plan9` (plan9port) — справочные исходники и рабочий devdraw; пока только читать.

## Команды

- Сборка: `./build.sh` (edwood и Syn в `bin/`); тесты Syn: `go test ./cmd/Syn/`; проверка: `cd edwood && ./presub.sh` (gofmt -s, vet, staticcheck, misspell, go test -race). Тестам нужен `rc` в PATH (`$PLAN9/bin`) или `acmeshell=sh`, иначе TestRunproc/TestMntDecRef падают — это окружение, не flaky.
- Запуск: `./run.sh <файлы>`; он выставляет `DEVDRAW=$PLAN9/bin/devdraw` и `NAMESPACE=/tmp/ns.edwood`, потому что у пользователя может параллельно работать plan9port acme с сервисом `acme`. Его не трогать.

## Правила

- Инварианты из `docs/02-scope-v1.md`: совместимость acme(4), мышиная модель Acme, стили поверх текста.
- Привязки клавиш — только через `keys.go` (`defaultBindings`, `actionTable`), не новые `case` в `Text.Type`; настраиваемое — в конфиге.
- Каждое изменение поведения: тест через `MakeWindowScaffold` + запись в `docs/90-differences-from-acme.md`.
- Не переименовывать модуль `github.com/rjkroege/edwood`, не чинить upstream-баги внутри фичевых веток, не добавлять зависимости и cgo без обсуждения.
- Коммиты только по команде пользователя; пуш и PR тоже.
- Сообщения коммитов в стиле Go: `text: move the cursor by line on Up/Down`, в теле ссылка на раздел спецификации.
