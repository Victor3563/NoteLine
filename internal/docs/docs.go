package docs

// MiniManualRU — текстовый мини-мануал (RU).
const MiniManualRU = `noteline — консольный блокнот

noteline хранит заметки в виде JSON-записей в сегментированных файлах
<root>/segments/notes-XXXXXXXX.ndjson. Каждая строка — одна заметка.

Базовые команды:

  noteline init [--root DIR]
      Создаёт хранилище (по умолчанию ~/.noteline).

  noteline create --title "..." --text "..." [--tags "a,b,c"]
      Создаёт заметку. Текст можно передать через --text или stdin.

  noteline read --id ID [--json]
      Показывает заметку по ID. В режиме --json выводит JSON-структуру.

  noteline update --id ID --title "..." --text "..." [--tags "..."]
      Создаёт новую версию заметки с тем же ID (лог-структурное обновление).

  noteline delete --id ID
      Помечает заметку как удалённую (tombstone).

  noteline list [--tag TAG] [--contains STR] [--limit N] [--json]
      Выводит список заметок, фильтруя по тегам и подстроке в тексте/заголовке.

  noteline search ...
      Синоним list, логически отделённая команда "поиск".

  noteline import --dir PATH [--ext "md,markdown,txt"] [--dry-run] [--verbose]
      Импортирует markdown-файлы с front matter. При повторном запуске
      обновляет существующие заметки и пропускает неизменённые.

  noteline completion SHELL
      Выводит скрипт автодополнения для bash/zsh/fish.

  noteline manual
      Показывает этот мини-мануал.

  noteline man
      Выводит man-страницу в формате roff (можно установить в систему).

Формат заметки:

  {
    "id":          "hex...",
    "title":       "Заголовок",
    "text":        "Текст заметки",
    "tags":        ["go","cli"],
    "created_at":  "...",
    "updated_at":  "...",
    "deleted":     false
  }

Импорт Markdown:

  Файл может начинаться с блока front matter:

  ---
  title: Моя заметка
  tags: go, cli
  created: 2025-10-22
  updated: 2025-10-22T12:34:56Z
  id: my-note-id
  ---
  Дальше идёт текст в Markdown.

  При импорте noteline:
    - старается использовать created/updated, если они заданы;
    - собирает теги из строки tags;
    - если указан id, использует его для "склеивания" импортов;
    - хранит индекс соответствия файлов и заметок в imports.json.

Completion:

  bash:
    noteline completion bash > /etc/bash_completion.d/noteline

  zsh:
    noteline completion zsh  > ~/.zsh/completions/_noteline

  fish:
    noteline completion fish > ~/.config/fish/completions/noteline.fish
`

// ManPageRU — man-страница в формате roff (раздел 1, RU).
const ManPageRU = `.TH NOTELINE 1 "2025-10-22" "noteline" "Пользовательские команды"
.SH ИМЯ
noteline \- консольный блокнот с сегментированным хранением заметок
.SH ОБЗОР
.B noteline
[\fICOMMAND\fR] [\fIOPTIONS\fR]
.SH ОПИСАНИЕ
.B noteline
хранит заметки в файлах сегментов формата NDJSON. Каждая строка сегмента
представляет собой JSON-структуру заметки. Обновления и удаления
реализованы лог-структурно: новые версии дописываются в конец.

.SH КОМАНДЫ
.TP
.B init
Создаёт новое хранилище (по умолчанию \fI~/.noteline\fR).

.TP
.B create
Создаёт заметку. Опции:
.RS
.TP
\fB\-\-title\fR STRING
Заголовок заметки (обязателен).
.TP
\fB\-\-text\fR STRING
Текст заметки. Если не указан, считывается из stdin.
.TP
\fB\-\-tags\fR "a,b,c"
Список тегов через запятую.
.RE

.TP
.B read
Читает заметку по ID. Опции:
.RS
.TP
\fB\-\-id\fR ID
Идентификатор заметки (обязателен).
.TP
\fB\-\-json\fR
Выводить заметку в формате JSON.
.RE

.TP
.B update
Создаёт новую версию заметки с тем же ID. Опции аналогичны
команде \fBcreate\fR плюс обязательный \fB\-\-id\fR.

.TP
.B delete
Помечает заметку как удалённую (tombstone) по ID.

.TP
.B list
Выводит список заметок. Опции:
.RS
.TP
\fB\-\-tag\fR TAG
Фильтр по тегу.
.TP
\fB\-\-contains\fR STR
Фильтр по подстроке в заголовке и тексте.
.TP
\fB\-\-limit\fR N
Ограничение на количество результатов.
.TP
\fB\-\-json\fR
Вывод списка в JSON.
.RE

.TP
.B search
Синоним команды \fBlist\fR. Логически отделён как "поиск".

.TP
.B import
Импортирует markdown-файлы из каталога. Опции:
.RS
.TP
\fB\-\-dir\fR PATH
Каталог с исходными файлами (обязателен).
.TP
\fB\-\-ext\fR "md,markdown,txt"
Список расширений файлов.
.TP
\fB\-\-dry\-run\fR
Показывать, что будет сделано, но не изменять хранилище.
.TP
\fB\-\-verbose\fR
Подробный отчёт по каждому файлу.
.RE

.TP
.B completion
Генерирует скрипт автодополнения для оболочек bash, zsh, fish.

.TP
.B manual
Печатает краткий русскоязычный мануал.

.TP
.B man
Выводит эту man-страницу в stdout. Можно установить:
.PP
.nf
  noteline man > noteline.1
  sudo mv noteline.1 /usr/share/man/man1/
  man noteline
.fi

.SH ХРАНЕНИЕ
Каталог хранилища по умолчанию:
.PP
  \fI~/.noteline\fR
.PP
Внутри:
.PP
.nf
  manifest.json      \- метаданные хранилища
  segments/notes\-*.ndjson \- сегменты с заметками
  imports.json       \- индекс соответствия импортируемых файлов и заметок
.fi

.SH АВТОРЫ
Проектная работа студентов ПМИ ВШЭ.
`

// Скрипт автодополнения для bash.
const BashCompletion = `# bash completion for noteline
_noteline_completion() {
  local cur prev
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"

  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "init create read update delete list search import completion manual man help" -- "$cur") )
    return
  fi

  case "${COMP_WORDS[1]}" in
    create)
      COMPREPLY=( $(compgen -W "--root --title --text --tags" -- "$cur") )
      ;;
    read)
      COMPREPLY=( $(compgen -W "--root --id --json" -- "$cur") )
      ;;
    update)
      COMPREPLY=( $(compgen -W "--root --id --title --text --tags" -- "$cur") )
      ;;
    delete)
      COMPREPLY=( $(compgen -W "--root --id" -- "$cur") )
      ;;
    list|search)
      COMPREPLY=( $(compgen -W "--root --tag --contains --limit --json" -- "$cur") )
      ;;
    import)
      COMPREPLY=( $(compgen -W "--root --dir --ext --dry-run --verbose" -- "$cur") )
      ;;
    completion)
      COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") )
      ;;
    *)
      ;;
  esac
}
complete -F _noteline_completion noteline
`

// Скрипт автодополнения для zsh.
const ZshCompletion = `#compdef noteline

_arguments -C \
  '1:command:(init create read update delete list search import completion manual man help)' \
  '*::arg:->args'

case $words[1] in
  create)
    _arguments '--root[Путь к хранилищу]' '--title[Заголовок]' '--text[Текст]' '--tags[Теги через запятую]'
    ;;
  read)
    _arguments '--root[Путь к хранилищу]' '--id[ID заметки]' '--json[Вывод в JSON]'
    ;;
  update)
    _arguments '--root[Путь к хранилищу]' '--id[ID заметки]' '--title[Новый заголовок]' '--text[Новый текст]' '--tags[Новые теги]'
    ;;
  delete)
    _arguments '--root[Путь к хранилищу]' '--id[ID заметки]'
    ;;
  list|search)
    _arguments '--root[Путь к хранилищу]' '--tag[Фильтр по тегу]' '--contains[Подстрока поиска]' '--limit[Лимит]' '--json[Вывод в JSON]'
    ;;
  import)
    _arguments '--root[Путь к хранилищу]' '--dir[Каталог импорта]' '--ext[Расширения файлов]' '--dry-run[Без изменений]' '--verbose[Подробный отчёт]'
    ;;
  completion)
    _arguments '1: :(bash zsh fish)'
    ;;
esac
`

// Скрипт автодополнения для fish.
const FishCompletion = `# fish completion for noteline

complete -c noteline -n "not __fish_seen_subcommand_from init create read update delete list search import completion manual man help" -a "init create read update delete list search import completion manual man help"

complete -c noteline -n "__fish_seen_subcommand_from create" -s - -l root   -d "Путь к хранилищу"
complete -c noteline -n "__fish_seen_subcommand_from create" -l title       -d "Заголовок"
complete -c noteline -n "__fish_seen_subcommand_from create" -l text        -d "Текст"
complete -c noteline -n "__fish_seen_subcommand_from create" -l tags        -d "Теги"

complete -c noteline -n "__fish_seen_subcommand_from read" -l root   -d "Путь к хранилищу"
complete -c noteline -n "__fish_seen_subcommand_from read" -l id     -d "ID заметки"
complete -c noteline -n "__fish_seen_subcommand_from read" -l json   -d "Вывод в JSON"

complete -c noteline -n "__fish_seen_subcommand_from update" -l root   -d "Путь к хранилищу"
complete -c noteline -n "__fish_seen_subcommand_from update" -l id     -d "ID заметки"
complete -c noteline -n "__fish_seen_subcommand_from update" -l title  -d "Новый заголовок"
complete -c noteline -n "__fish_seen_subcommand_from update" -l text   -d "Новый текст"
complete -c noteline -n "__fish_seen_subcommand_from update" -l tags   -d "Новые теги"

complete -c noteline -n "__fish_seen_subcommand_from delete" -l root   -d "Путь к хранилищу"
complete -c noteline -n "__fish_seen_subcommand_from delete" -l id     -d "ID заметки"

complete -c noteline -n "__fish_seen_subcommand_from list search" -l root     -d "Путь к хранилищу"
complete -c noteline -n "__fish_seen_subcommand_from list search" -l tag      -d "Фильтр по тегу"
complete -c noteline -n "__fish_seen_subcommand_from list search" -l contains -d "Подстрока"
complete -c noteline -n "__fish_seen_subcommand_from list search" -l limit    -d "Лимит"
complete -c noteline -n "__fish_seen_subcommand_from list search" -l json     -d "Вывод в JSON"

complete -c noteline -n "__fish_seen_subcommand_from import" -l root     -d "Путь к хранилищу"
complete -c noteline -n "__fish_seen_subcommand_from import" -l dir      -d "Каталог импорта"
complete -c noteline -n "__fish_seen_subcommand_from import" -l ext      -d "Расширения файлов"
complete -c noteline -n "__fish_seen_subcommand_from import" -l dry-run  -d "Без изменений"
complete -c noteline -n "__fish_seen_subcommand_from import" -l verbose  -d "Подробный отчёт"
`
