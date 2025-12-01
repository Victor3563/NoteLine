// Вверхнеуровневые команды для main
// После прочтения удалить
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/i18n"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/model"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/store"
)

func defaultRoot(_ string) string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".data")
}

func CmdInit(root string) error {
	root = defaultRoot(root)
	return store.Ensure(root)
}

func CmdCreate(root, title, text string, tags []string) (string, error) {
	root = defaultRoot(root)
	s, err := store.Open(root)
	if err != nil {
		return "", err
	}
	defer s.Close()
	n := model.NewNote(title, text, tags)
	if err := s.Append(n); err != nil {
		return "", err
	}
	return n.ID, nil
}

func CmdRead(root, id string, asJSON bool) error {
	root = defaultRoot(root)
	s, err := store.Open(root)
	if err != nil {
		return err
	}
	defer s.Close()
	n, err := s.GetByID(id)
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(n)
	}
	fmt.Printf("[%s] %s\n", n.ID, n.Title)
	if len(n.Tags) > 0 {
		fmt.Printf(i18n.T("cmd.tags")+"\n", strings.Join(n.Tags, ", "))
	}
	fmt.Printf(i18n.T("cmd.created")+"\n", n.CreatedAt.Format("2006-01-02 15:04:05"))
	if !n.UpdatedAt.IsZero() {
		fmt.Printf(i18n.T("cmd.updated")+"\n", n.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println(i18n.T("cmd.sep"))
	fmt.Println(n.Text)
	return nil
}

func CmdList(root, tag, contains string, limit int, asJSON bool) error {
	root = defaultRoot(root)
	s, err := store.Open(root)
	if err != nil {
		return err
	}
	defer s.Close()

	filter := store.Filter{
		Tag:      strings.TrimSpace(tag),
		Contains: strings.TrimSpace(contains),
		Limit:    limit,
	}
	list, err := s.List(filter)
	if err != nil {
		return err
	}

	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}

	for i, n := range list {
		if limit > 0 && i >= limit {
			break
		}
		if len(n.Title) > 0 {
			fmt.Printf("[%s] %s\n", n.ID, n.Title)
		} else {
			fmt.Printf("[%s]\n", n.ID)
		}
		if len(n.Tags) > 0 {
			fmt.Printf(i18n.T("cmd.tags_indented")+"\n", strings.Join(n.Tags, ", "))
		}
		fmt.Printf(i18n.T("cmd.created_indented")+"\n", n.CreatedAt.Format("2006-01-02 15:04:05"))

		// подсветка совпадений — только при текстовом выводе (не JSON) и если задан contains
		if strings.TrimSpace(contains) != "" && !asJSON {
			sn := highlightSnippet(&n, contains)
			if sn != "" {
				fmt.Printf(i18n.T("cmd.match")+"\n", sn)
			}
		}
	}

	return nil
}

var tokenRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

// extractTokens извлекает "токены" (слова/числа) из поискового запроса.
// Убирает операторы и спецсимволы; возвращает список в нижнем регистре.
func extractTokens(query string) []string {
	if strings.TrimSpace(query) == "" {
		return nil
	}
	m := tokenRe.FindAllString(query, -1)
	out := make([]string, 0, len(m))
	for _, t := range m {
		t = strings.ToLower(strings.TrimSpace(t))
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// highlightSnippet возвращает короткий фрагмент текста из title+text
// с подсветкой первого вхождения любого токена (ANSI цвет).
// Если совпадений нет, возвращает пустую строку.
func highlightSnippet(n *model.Note, query string) string {
	toks := extractTokens(query)
	if len(toks) == 0 {
		return ""
	}

	// соберём целевой текст: title + " — " + text (если есть)
	var txt string
	if strings.TrimSpace(n.Title) != "" {
		if strings.TrimSpace(n.Text) != "" {
			txt = n.Title + " — " + n.Text
		} else {
			txt = n.Title
		}
	} else {
		txt = n.Text
	}

	// подготовим regexp для поиска любого токена, case-insensitive
	// экранируем токены (на случай спецсимволов)
	esc := make([]string, 0, len(toks))
	for _, t := range toks {
		esc = append(esc, regexp.QuoteMeta(t))
	}
	pattern := `(?i)(` + strings.Join(esc, `|`) + `)`
	re := regexp.MustCompile(pattern)

	// ищем первое вхождение
	loc := re.FindStringIndex(txt)
	// если не нашли в title/text, попробуем теги
	if loc == nil {
		tags := strings.Join(n.Tags, " ")
		loc = re.FindStringIndex(tags)
		if loc == nil {
			return ""
		}
		txt = tags
	}

	// сформируем контекстный фрагмент (по 40 символов слева и справа)
	left := 40
	right := 40
	start := loc[0] - left
	if start < 0 {
		start = 0
	}
	end := loc[1] + right
	if end > len(txt) {
		end = len(txt)
	}
	snippet := txt[start:end]

	// заменим все вхождения в snippet на ANSI-код
	highlighted := re.ReplaceAllStringFunc(snippet, func(m string) string {
		// \x1b[31;1m = красный жирный, \x1b[0m = сброс
		return "\x1b[31;1m" + m + "\x1b[0m"
	})

	if start > 0 {
		highlighted = "..." + highlighted
	}
	if end < len(txt) {
		highlighted = highlighted + "..."
	}
	return highlighted
}
