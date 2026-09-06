# CS2 Market Arbitrage Scanner

Веб-инструмент для сравнения цен на скины CS2 между торговыми площадками и поиска арбитражных связок.

## Основные части

- Go-парсер на базе проекта [csgodatabase-scraper](https://github.com/eovacius/csgodatabase-scraper).
- Chromium-автоматизация для загрузки динамических страниц CSGO Database.
- GitHub Actions для регулярного обновления цен.
- Единый файл `json/data.json` с актуальными данными.
- Node.js-сервер и браузерная таблица для сравнения цен.
- Поддержка обычных скинов и `StatTrak`.
- Все цены отображаются только в USD, без конвертации валют и курсов.

## Схема работы

```text
CSGO Database
      |
      v
Go-парсер на базе csgodatabase-scraper
      |
      v
json/data.json
      |
      v
GitHub Actions -> GitHub repository
      |
      v
Node.js server -> браузерная таблица
```

1. Парсер открывает страницы оружия и скинов в Chromium.
2. Из карточек цен извлекаются магазины, состояния износа, цены и ссылки.
3. Результат сохраняется в `json/data.json`.
4. GitHub Actions отправляет обновлённый файл в ветку `main`.
5. Сервер скачивает `data.json` только с GitHub.
6. Таблица сравнивает цены покупки и продажи и рассчитывает прибыль в USD.

## Поддерживаемые магазины

Список зависит от того, какие карточки и объявления доступны для конкретного скина:

- Steam
- Tradeit.GG
- CS.MONEY
- PirateSwap
- SkinSwap
- Skin.Land
- Skinport
- Market.CSGO
- DMarket
- LIS-SKINS
- iTrade.GG

Если у магазина нет объявления, такая цена не записывается.

## Состояния износа

- `FN` — Factory New
- `MW` — Minimal Wear
- `FT` — Field-Tested
- `WW` — Well-Worn
- `BS` — Battle-Scarred

## Normal и StatTrak

Обычная версия и `StatTrak` сохраняются как отдельные записи:

```json
{
  "name": "CZ75-Auto | Victoria",
  "type": "Normal"
}
```

```json
{
  "name": "CZ75-Auto | Victoria",
  "type": "StatTrak"
}
```

Для каждой версии сохраняются отдельные цены, состояния износа и ссылки на магазины.

## Формат `json/data.json`

Парсер всегда перезаписывает один файл:

```text
json/data.json
```

Пример цены:

```json
{
  "market": "Steam",
  "wear": "FN",
  "price": 146.71,
  "currency": "USD",
  "has_price": true,
  "url": "https://steamcommunity.com/market/listings/730/..."
}
```

Цены в веб-интерфейсе показываются в том же значении, которое пришло в `data.json`:

```text
$ 146.71
```

Никакие курсы USD/RUB и другие конвертации не применяются.

## Требования

- Go 1.24 или новее.
- Google Chrome, Chromium или Microsoft Edge.
- Node.js 18 или новее.
- Доступ к интернету.

## Локальный запуск парсера

Установить Go-зависимости:

```bash
go mod tidy
```

Полный запуск всех оружий:

```bash
go run . --delay=3 --workers=2
```

После завершения файл будет записан в:

```text
json/data.json
```

### Параметры парсера

- `--delay=3` — задержка между запросами.
- `--workers=2` — количество параллельных страниц со скинами.
- `--max-weapons=2` — тестовый запуск только первых двух оружий.
- `--interactive` — видимый Chrome для ручного прохождения Cloudflare.
- `--aggressive` — запуск без задержки.
- `--stealth` — дополнительная случайная задержка.

Тестовый запуск двух оружий:

```bash
go run . --delay=3 --workers=2 --max-weapons=2
```

Если страница скина прошла Cloudflare, но цены ещё не успели появиться, парсер ждёт 10 секунд и повторяет обработку.

## GitHub Actions

### Полный парсер

Файл:

```text
.github/workflows/go.yml
```

Запускается автоматически каждые 6 часов или вручную через `workflow_dispatch`.

Команда workflow:

```bash
go run . --delay=3 --workers=2
```

После завершения `json/data.json` коммитится и отправляется в GitHub.

### Тестовый парсер

Файл:

```text
.github/workflows/go-test-2-weapons.yml
```

Запускается вручную через GitHub Actions и обрабатывает только два оружия:

```bash
go run . --delay=3 --workers=2 --max-weapons=2
```

Тестовый workflow также сохраняет результат в `json/data.json`.

## Запуск веб-сервера

Установить Node.js-зависимости:

```bash
npm install
```

Запустить сервер:

```bash
npm start
```

Открыть в браузере:

```text
http://localhost:3000
```

Сервер загружает только удалённый файл:

```text
https://raw.githubusercontent.com/davlanca/VortexFocus/main/json/data.json
```

Локальный `json/data.json` сервером не используется. Если файла нет на GitHub, сервер сообщает об ошибке загрузки и не переключается на старые файлы с датой.

## API сервера

- `GET /api/config` — доступные магазины, комиссии и количество предметов;
- `GET /api/diagnostic` — диагностика загрузки GitHub-файла;
- `POST /api/refresh` — повторная загрузка `data.json` с GitHub;
- `POST /api/fees` — изменение комиссий в памяти сервера;
- `POST /api/arbitrage` — поиск арбитражных связок между двумя магазинами.

Пример запроса арбитража:

```json
{
  "buyMarket": "Market.CSGO",
  "sellMarket": "LIS-SKINS",
  "onlyLiquid": false,
  "minProfit": 0,
  "search": "CZ75-Auto",
  "sortBy": "profitPercent",
  "sortDirection": "desc"
}
```

Все цены в ответе API находятся в USD:

- `buyPrice`;
- `buyTotal`;
- `sellPrice`;
- `sellNet`;
- `profitVal`.

## Структура проекта

```text
main.go                                  CLI Go-парсера
scraper/worker/                          обход оружия и скинов
scraper/discovery/                       поиск страниц оружия и скинов
scraper/common/                          загрузка страниц и обработка цен
scraper/config/models.go                 структуры JSON
scraper/config/scripts/prices.js         извлечение карточек магазинов
scraper/config/scripts/config.js         настройки Chromium-страницы
server.js                                Node.js API и арбитражный расчёт
index.html                               веб-интерфейс
json/data.json                           последний файл с ценами
.github/workflows/go.yml                 полный запуск парсера
.github/workflows/go-test-2-weapons.yml  тестовый запуск двух оружий
```

## Проверка проекта

Проверить Go-парсер:

```bash
go test ./...
```

Проверить Node.js-сервер:

```bash
node --check server.js
```

## Источник парсера

Go-парсер проекта основан на репозитории:

[https://github.com/eovacius/csgodatabase-scraper](https://github.com/eovacius/csgodatabase-scraper)

В этом проекте код адаптирован для:

- сбора карточек нескольких торговых площадок;
- обработки Normal и StatTrak;
- сохранения `json/data.json`;
- повторной попытки после Cloudflare;
- запуска через GitHub Actions;
- последующего анализа цен в арбитражной таблице.

## Важные замечания

Цены зависят от текущего состояния внешних торговых площадок. Используйте парсер с учётом правил и ограничений сайта-источника.
