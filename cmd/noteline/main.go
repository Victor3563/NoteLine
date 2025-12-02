package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/cli"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/docs"
)

const helpText = `noteline — CLI-блокнот с сегментированным хранением.

Использование:
  noteline init [--root DIR]
  noteline create [--root DIR] --title "..." --text "..." [--tags "a,b,c"]
  noteline read [--root DIR] --id ID [--json]
  noteline update [--root DIR] --id ID --title "..." --text "..." [--tags "a,b,c"]
  noteline delete [--root DIR] --id ID
  noteline list   [--root DIR] [--tag TAG] [--contains STR] [--limit N] [--json]
  noteline search [--root DIR] [--tag TAG] [--contains STR] [--limit N] [--json]
  noteline import [--root DIR] --dir PATH [--ext "md,markdown,txt"] [--dry-run] [--verbose]
  noteline completion SHELL      # SHELL = bash|zsh|fish
  noteline manual                # мини-мануал (RU)
  noteline man                   # man-страница в формате roff (RU)
  noteline --help | -h | help

Примеры:
  noteline init
  noteline create --title "Идея" --text "Сделать CLI" --tags go,ideas
  noteline list --tag go --limit 20
  noteline search --contains "cli"
  noteline import --dir ./notes
  noteline completion bash > /etc/bash_completion.d/noteline
  noteline man > noteline.1
`

func main() {
	if len(os.Args) < 2 {
		fmt.Println(helpText)
		os.Exit(0)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "help", "-h", "--help":
		fmt.Println(helpText)
		return

	case "init":
		fs := flag.NewFlagSet("init", flag.ExitOnError)
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.noteline)")
		_ = fs.Parse(args)
		if err := cli.CmdInit(*root); err != nil {
			fmt.Fprintln(os.Stderr, "init:", err)
			os.Exit(1)
		}

	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.noteline)")
		title := fs.String("title", "", "Заголовок заметки")
		text := fs.String("text", "", "Текст заметки (если пусто — будет прочитан из stdin)")
		tags := fs.String("tags", "", "Список тегов через запятую")
		_ = fs.Parse(args)

		body := strings.TrimSpace(*text)
		if body == "" {
			// читаем из stdin (pipe или ручной ввод с завершением Ctrl+D)
			data, err := io.ReadAll(os.Stdin)
			if err == nil {
				body = strings.TrimSpace(string(data))
			}
		}

		if strings.TrimSpace(*title) == "" {
			fmt.Fprintln(os.Stderr, "create: требуется --title (непустой)")
			os.Exit(2)
		}
		if body == "" {
			fmt.Fprintln(os.Stderr, "create: требуется текст: укажи --text или подай stdin")
			os.Exit(2)
		}

		var tagsSlice []string
		if strings.TrimSpace(*tags) != "" {
			for _, t := range strings.Split(*tags, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tagsSlice = append(tagsSlice, t)
				}
			}
		}

		id, err := cli.CmdCreate(*root, *title, body, tagsSlice)
		if err != nil {
			fmt.Fprintln(os.Stderr, "create:", err)
			os.Exit(1)
		}
		fmt.Println(id)

	case "read":
		fs := flag.NewFlagSet("read", flag.ExitOnError)
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.noteline)")
		id := fs.String("id", "", "ID заметки")
		asJSON := fs.Bool("json", false, "Вывести заметку в JSON")
		_ = fs.Parse(args)
		if strings.TrimSpace(*id) == "" {
			fmt.Fprintln(os.Stderr, "read: требуется --id")
			os.Exit(2)
		}
		if err := cli.CmdRead(*root, *id, *asJSON); err != nil {
			fmt.Fprintln(os.Stderr, "read:", err)
			os.Exit(1)
		}

	case "update":
		fs := flag.NewFlagSet("update", flag.ExitOnError)
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.noteline)")
		id := fs.String("id", "", "ID заметки")
		title := fs.String("title", "", "Новый заголовок заметки")
		text := fs.String("text", "", "Новый текст заметки (если пусто — будет прочитан из stdin)")
		tags := fs.String("tags", "", "Новый список тегов через запятую (полностью заменяет старый)")
		_ = fs.Parse(args)

		if strings.TrimSpace(*id) == "" {
			fmt.Fprintln(os.Stderr, "update: требуется --id")
			os.Exit(2)
		}

		body := strings.TrimSpace(*text)
		if body == "" {
			data, err := io.ReadAll(os.Stdin)
			if err == nil {
				body = strings.TrimSpace(string(data))
			}
		}

		if strings.TrimSpace(*title) == "" {
			fmt.Fprintln(os.Stderr, "update: требуется --title (непустой)")
			os.Exit(2)
		}
		if body == "" {
			fmt.Fprintln(os.Stderr, "update: требуется текст: укажи --text или подай stdin")
			os.Exit(2)
		}

		var tagsSlice []string
		if strings.TrimSpace(*tags) != "" {
			for _, t := range strings.Split(*tags, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tagsSlice = append(tagsSlice, t)
				}
			}
		}

		if err := cli.CmdUpdate(*root, *id, *title, body, tagsSlice); err != nil {
			fmt.Fprintln(os.Stderr, "update:", err)
			os.Exit(1)
		}

	case "delete":
		fs := flag.NewFlagSet("delete", flag.ExitOnError)
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.noteline)")
		id := fs.String("id", "", "ID заметки")
		_ = fs.Parse(args)
		if strings.TrimSpace(*id) == "" {
			fmt.Fprintln(os.Stderr, "delete: требуется --id")
			os.Exit(2)
		}
		if err := cli.CmdDelete(*root, *id); err != nil {
			fmt.Fprintln(os.Stderr, "delete:", err)
			os.Exit(1)
		}

	case "list":
		fs := flag.NewFlagSet("list", flag.ExitOnError)
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.noteline)")
		tag := fs.String("tag", "", "Фильтр по тегу (точное совпадение)")
		contains := fs.String("contains", "", "Фильтр по вхождению подстроки в заголовок/текст")
		limit := fs.Int("limit", 0, "Ограничить количество результатов")
		asJSON := fs.Bool("json", false, "Вывести список в JSON")
		_ = fs.Parse(args)
		if err := cli.CmdList(*root, *tag, *contains, *limit, *asJSON); err != nil {
			fmt.Fprintln(os.Stderr, "list:", err)
			os.Exit(1)
		}

	case "search":
		// алиас для list, просто логически другая команда
		fs := flag.NewFlagSet("search", flag.ExitOnError)
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.noteline)")
		tag := fs.String("tag", "", "Фильтр по тегу (точное совпадение)")
		contains := fs.String("contains", "", "Фильтр по вхождению подстроки в заголовок/текст")
		limit := fs.Int("limit", 0, "Ограничить количество результатов")
		asJSON := fs.Bool("json", false, "Вывести список в JSON")
		_ = fs.Parse(args)
		if err := cli.CmdList(*root, *tag, *contains, *limit, *asJSON); err != nil {
			fmt.Fprintln(os.Stderr, "search:", err)
			os.Exit(1)
		}

	case "import":
		fs := flag.NewFlagSet("import", flag.ExitOnError)
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.noteline)")
		dir := fs.String("dir", "", "Каталог с markdown-файлами")
		exts := fs.String("ext", "md,markdown,txt", "Список расширений через запятую (без точки или с точкой)")
		dryRun := fs.Bool("dry-run", false, "Показать, что будет сделано, но не изменять хранилище")
		verbose := fs.Bool("verbose", false, "Подробный отчёт по каждому файлу")
		_ = fs.Parse(args)

		if strings.TrimSpace(*dir) == "" {
			// позволяем передать каталог позиционным аргументом
			rest := fs.Args()
			if len(rest) > 0 {
				*dir = rest[0]
			}
		}
		if strings.TrimSpace(*dir) == "" {
			fmt.Fprintln(os.Stderr, "import: требуется указать --dir PATH или позиционный параметр каталога")
			os.Exit(2)
		}

		if err := cli.CmdImport(*root, *dir, *exts, *dryRun, *verbose); err != nil {
			fmt.Fprintln(os.Stderr, "import:", err)
			os.Exit(1)
		}

	case "completion":
		fs := flag.NewFlagSet("completion", flag.ExitOnError)
		shell := fs.String("shell", "", "Тип оболочки: bash, zsh или fish")
		_ = fs.Parse(args)

		if strings.TrimSpace(*shell) == "" {
			rest := fs.Args()
			if len(rest) > 0 {
				*shell = rest[0]
			}
		}

		s := strings.ToLower(strings.TrimSpace(*shell))
		switch s {
		case "bash":
			fmt.Print(docs.BashCompletion)
		case "zsh":
			fmt.Print(docs.ZshCompletion)
		case "fish":
			fmt.Print(docs.FishCompletion)
		default:
			fmt.Fprintln(os.Stderr, "completion: укажи оболочку: bash, zsh или fish")
			os.Exit(2)
		}

	case "manual":
		fmt.Println(docs.MiniManualRU)

	case "man":
		fmt.Println(docs.ManPageRU)

	default:
		fmt.Fprintf(os.Stderr, "неизвестная команда: %s\n\n", cmd)
		fmt.Println(helpText)
		os.Exit(2)
	}

}
