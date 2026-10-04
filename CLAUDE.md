# justcode

Доработанный Acme на базе форка Edwood (Go). Обсуждение и документы — на русском; код, комментарии и коммиты — на английском.

## Где что

- `docs/` — спецификации и решения, читать перед работой над разделом: `01` обзор Acme, `02` рамки и инварианты, `03` клавиатура, `04` процесс и правила кода, `90` журнал отличий от Acme.
- `edwood/` — клон Edwood, remote `upstream` = rjkroege/edwood. Код правим только здесь, в фичевых ветках (`keys/*`, `style/*`, `fmt/*`, `devdraw/*`).
- `devdraw/` — форк Go-devdraw (модуль `justcode/devdraw`), правки клавиатурного ввода делаются здесь, не в `~/projects/9fans/go`.
- `bin/edwood`, `bin/devdraw` — сборки для ручной проверки, не в git.
- `~/projects/plan9` (plan9port) и `~/projects/9fans/go` — справочные исходники, только читать.

## Команды

- Сборка и проверка: `cd edwood && go build -o ../bin/edwood . && ./presub.sh` (gofmt -s, vet, staticcheck, misspell, go test -race).
- Запуск: см. `docs/04-process.md §5`; обязательно `DEVDRAW=$PWD/bin/devdraw` и `NAMESPACE=/tmp/ns.edwood`, потому что у пользователя параллельно работает plan9port acme с сервисом `acme`. Его не трогать.

## Правила

- Инварианты из `docs/02-scope-v1.md`: совместимость acme(4), мышиная модель Acme, стили поверх текста.
- Привязки клавиш — только через `keys.go` (`defaultBindings`, `actionTable`), не новые `case` в `Text.Type`; настраиваемое — в конфиге.
- Каждое изменение поведения: тест через `MakeWindowScaffold` + запись в `docs/90-differences-from-acme.md`.
- Не переименовывать модуль `github.com/rjkroege/edwood`, не чинить upstream-баги внутри фичевых веток, не добавлять зависимости и cgo без обсуждения.
- Коммиты только по команде пользователя; пуш и PR тоже.
- Сообщения коммитов в стиле Go: `text: move the cursor by line on Up/Down`, в теле ссылка на раздел спецификации.
