// Тут просто ловим ввод терминала, обрабатываем флаги и вызываем функции из comands
// После прочтения удалить
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/cli"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/i18n"
)

func main() {
	_ = i18n.InitFromEnv()
	helpText := i18n.T("help_text")
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
	// case "init":
	// 	fs := flag.NewFlagSet("init", flag.ExitOnError)
	// 	_ = fs.Parse(args)
	// 	if err := cli.CmdInit(""); err != nil {
	// 		fmt.Fprintln(os.Stderr, "init:", err)
	// 		os.Exit(1)
	// 	}
	case "create":
		fs := flag.NewFlagSet("create", flag.ExitOnError)
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
		id, err := cli.CmdCreate("", *title, body, tagsSlice)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("cmd.create"), err)
			os.Exit(1)
		}
		fmt.Println(id)
	case "read":
		fs := flag.NewFlagSet("read", flag.ExitOnError)
		id := fs.String("id", "", "ID заметки")
		asJSON := fs.Bool("json", false, "Вывести заметку в JSON")
		_ = fs.Parse(args)
		if strings.TrimSpace(*id) == "" {
			fmt.Fprintln(os.Stderr, i18n.T("main.read_missing_id"))
			os.Exit(2)
		}
		if err := cli.CmdRead("", *id, *asJSON); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("cmd.read"), err)
			os.Exit(1)
		}
	case "list":
		fs := flag.NewFlagSet("list", flag.ExitOnError)
		tag := fs.String("tag", "", "Фильтр по тегу (входит в список тегов)")
		contains := fs.String("contains", "", "Фильтр по вхождению подстроки в заголовок/текст")
		limit := fs.Int("limit", 0, "Ограничить количество результатов")
		asJSON := fs.Bool("json", false, "Вывести список в JSON")
		_ = fs.Parse(args)
		if err := cli.CmdList("", *tag, *contains, *limit, *asJSON); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", i18n.T("cmd.list"), err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, i18n.T("main.unknown_cmd"), cmd, helpText)
		fmt.Println(helpText)
		os.Exit(2)
	}

}
