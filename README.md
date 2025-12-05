
## 🌐 Language / Язык
[🇬🇧 English](#english-version) | [🇷🇺 Русский](#русская-версия)

---

<a id="русская-версия"></a>


# 📝 NoteLine

Минимальный консольный блокнот на Go с хранением заметок в **NDJSON-сегментах**, возможностью фильтрации, чтения и добавления через терминал.

## 📦 Установка и запуск

1. **Скачивание и запуск скрипта установки одной командой**

Для **Linux/macOS**:

Устновитe zip/unzip:
```
sudo apt update
sudo apt install zip unzip
```
Загрузите данные:
```bash
curl -sSL https://raw.githubusercontent.com/Victor3563/NoteLine/main/install.sh | bash
```

Для **Windows (PowerShell обязательно)**:

Обновите Powershell:
```powershell
winget search --id Microsoft.PowerShell
```
Загружаем:
```powershell
iwr 'https://raw.githubusercontent.com/Victor3563/NoteLine/main/install.ps1' -OutFile $env:TEMP\install.ps1; powershell -NoProfile -ExecutionPolicy Bypass -File $env:TEMP\install.ps1; Remove-Item $env:TEMP\install.ps1 -Force
```
и перезагрузите Powershell
1. **Запуск программы**

После выполнения скрипта можно сразу запускать:

```bash
noteline -h
```
1. **Обновление**

Чтобы обновить до последней версии, достаточно снова запустить скрипт:

```bash
curl -sSL https://raw.githubusercontent.com/Victor3563/NoteLine/main/install.sh | bash
```

или соответствующий для PowerShell на Windows.


## 🌐 Смена языка интерфейса

NoteLine поддерживает локализацию через переменную окружения `NOTELINE_LANG`.

### Установка языка

Укажите язык перед запуском программы:

```bash
export NOTELINE_LANG=ru
./noteline
```

Доступные значения: `en`, `ru`.

Если переменная не указана или некорректна — используется язык по умолчанию (`en`).

## Использование

#### `create` — создать новую заметку

```bash
noteline create --title "Заголовок" --text "Текст заметки" [--tags "тег1,тег2"]
```

* Если `--text` не указан, текст читается из stdin.
* Теги указываются через запятую.

#### `read` — вывести заметку по ID

```bash
noteline read --id <ID> [--json]
```

* Флаг `--json` выводит заметку в формате JSON.

#### `list` — показать список заметок

```bash
noteline list [--tag <TAG>] [--contains <STR>] [--limit N] [--json]
```

* `--tag` — искать заметки с определённым тегом.
* `--contains` — поиск по подстроке в заголовке или тексте.
* `--limit` — ограничить количество результатов.
* `--json` — вывод в формате JSON.

#### `help` — показать справку

```bash
noteline help
```

## Особенности

* Все заметки хранятся в сегментированных NDJSON-файлах (~8 МБ каждый) в `~/.data`.
* Поддерживается полнотекстовый поиск по заголовкам, тексту и тегам.
* Система кеширования LRU ускоряет доступ к недавно прочитанным заметкам.

## ⚡ Benchmark

Встроенный инструмент нагрузочного тестирования.

### Запуск
```bash
go build ./cmd/bench
./bench --n 2000 --q 500 --out bench.csv
```

### Параметры

* `--n` — сколько создать заметок
* `--q` — сколько выполнить поисковых запросов
* `--out` — CSV с временем выполнения каждого запроса (в ns)

### Что делает

Создаёт временную директорию `/tmp/noteline-bench-*`,
генерирует N заметок, выполняет Q поисков,
пишет статистику и удаляет директорию.


## 🧪 Regtest — изолированные регресc-тесты

`regtest` запускает набор предопределённых запросов против **временного хранилища** NoteLine.
Данные никогда не затрагивают основную базу: перед тестами создаётся временное хранилище, после выполнения он удаляется.

## Что делает
- создаёт чистое временное хранилище
- генерирует 20 тестовых заметок
- запускает 20 `exact` (поиск по id) и 5 `contains` запросов
- сверяет результат с ожидаемыми значениями
- печатает `PASS/FAIL` и общее количество успешных тестов
- удаляет временную директорию

## Запуск
```bash
go build ./cmd/regtest
./regtest
```


<a id="english-version"></a>

---

<div align="center">

# 🌐🌐🌐

</div>



# 📝 NoteLine

A minimal console-based notebook written in Go, storing notes in **NDJSON segments**, with filtering, reading, and creation directly from the terminal.

## 📦 Installation & Run

### 1. **Single-command installation**

#### For **Linux/macOS**:

Install zip/unzip:

```
sudo apt update
sudo apt install zip unzip
```

Download and run the installer:

```bash
curl -sSL https://raw.githubusercontent.com/Victor3563/NoteLine/main/install.sh | bash
```

#### For **Windows (PowerShell required)**:

Check for PowerShell updates:

```powershell
winget search --id Microsoft.PowerShell
```

Download and install:

```powershell
iwr 'https://raw.githubusercontent.com/Victor3563/NoteLine/main/install.ps1' -OutFile $env:TEMP\install.ps1; powershell -NoProfile -ExecutionPolicy Bypass -File $env:TEMP\install.ps1; Remove-Item $env:TEMP\install.ps1 -Force
```

Then restart PowerShell.

### 2. **Running the program**

After the installer finishes, the program is ready:

```bash
noteline -h
```

### 3. **Updating**

To update to the latest version, simply rerun the script:

```bash
curl -sSL https://raw.githubusercontent.com/Victor3563/NoteLine/main/install.sh | bash
```

(or the Windows PowerShell version).

---

## 🌐 Changing Interface Language

NoteLine supports localization via the `NOTELINE_LANG` environment variable.

### Setting a language

Specify a language before launching the program:

```bash
export NOTELINE_LANG=en
./noteline
```

Available values: `en`, `ru`.

If missing or invalid, the default language (`en`) is used.

---

## Usage

#### `create` — create a new note

```bash
noteline create --title "Title" --text "Note body" [--tags "tag1,tag2"]
```

* If `--text` is not provided, text is read from stdin.
* Tags are comma-separated.

#### `read` — print a note by ID

```bash
noteline read --id <ID> [--json]
```

* `--json` prints the note in JSON format.

#### `list` — display list of notes

```bash
noteline list [--tag <TAG>] [--contains <STR>] [--limit N] [--json]
```

* `--tag` — filter notes by tag
* `--contains` — search by substring in title or body
* `--limit` — limit results
* `--json` — output in JSON format

#### `help` — show help

```bash
noteline help
```

---

## Features

* All notes are stored in segmented NDJSON files (~8 MB each) in `~/.data`.
* Full-text search across titles, bodies, and tags.
* Built-in LRU caching to speed up access to recently read notes.

---

## ⚡ Benchmark

Includes an internal benchmarking tool.

### Run

```bash
go build ./cmd/bench
./bench --n 2000 --q 500 --out bench.csv
```

### Parameters

* `--n` — how many notes to create
* `--q` — how many search queries to run
* `--out` — CSV path for timing of each query (ns)

### What it does

Creates a temporary directory `/tmp/noteline-bench-*`,
generates N notes, performs Q searches, writes stats,
and removes the directory afterward.

---

## 🧪 Regtest — isolated regression tests

`regtest` executes a predefined test suite against a **temporary NoteLine storage**.
It never touches the user's real note database.

### What it does

* creates a fresh temporary store
* generates 20 test notes
* runs 20 `exact` (ID lookup) and 5 `contains` queries
* compares results with expected values
* prints `PASS/FAIL` and total success count
* deletes the temporary directory afterward

### Run

```bash
go build ./cmd/regtest
./regtest
```

