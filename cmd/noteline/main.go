// Тут просто ловим ввод терминала, обрабатываем флаги и вызываем функции из comands
// После прочтения удалить
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/cli"
)

const helpText = `noteline — простой CLI-блокнот.
Использование:
  noteline init [--root DIR]
  noteline create [--root DIR] --title "..." --text "..." [--tags "a,b,c"]
  noteline read [--root DIR] --id ID [--json]
  noteline list [--root DIR] [--tag TAG] [--contains STR] [--limit N] [--json]
  noteline --help | -h | help

Примеры:
  noteline init
  noteline create --title "Идея" --text "Сделать CLI" --tags go,ideas
  noteline list --tag go --limit 20
  noteline read --id 01JABCDXYZ...`

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
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.clinote)")
		_ = fs.Parse(args)
		if err := cli.CmdInit(*root); err != nil {
			fmt.Fprintln(os.Stderr, "init:", err)
			os.Exit(1)
		}
	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.clinote)")
		title := fs.String("title", "", "Заголовок заметки")
		text := fs.String("text", "", "Текст заметки (если пусто — будет прочитан из stdin)")
		tags := fs.String("tags", "", "Список тегов через запятую")
		_ = fs.Parse(args)

		body := *text
		if strings.TrimSpace(body) == "" {
			data, _ := os.ReadFile("/dev/stdin")
			body = string(data)
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
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.clinote)")
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
	case "list":
		fs := flag.NewFlagSet("list", flag.ExitOnError)
		root := fs.String("root", "", "Путь к каталогу данных (по умолчанию ~/.clinote)")
		tag := fs.String("tag", "", "Фильтр по тегу (входит в список тегов)")
		contains := fs.String("contains", "", "Фильтр по вхождению подстроки в заголовок/текст")
		limit := fs.Int("limit", 0, "Ограничить количество результатов")
		asJSON := fs.Bool("json", false, "Вывести список в JSON")
		_ = fs.Parse(args)
		if err := cli.CmdList(*root, *tag, *contains, *limit, *asJSON); err != nil {
			fmt.Fprintln(os.Stderr, "list:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "неизвестная команда: %s\n\n", cmd)
		fmt.Println(helpText)
		os.Exit(2)
	}

}
