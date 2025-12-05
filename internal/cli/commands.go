package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/i18n"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/importer"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/model"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/store"
)

func defaultRoot(root string) string {
	if strings.TrimSpace(root) != "" {
		return root
	}
	home, _ := os.UserHomeDir()

	return filepath.Join(home, ".noteline")
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
	if !n.UpdatedAt.IsZero() && !n.UpdatedAt.Equal(n.CreatedAt) {
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

		if strings.TrimSpace(contains) != "" && !asJSON {
			sn := highlightSnippet(&n, contains)
			if sn != "" {
				fmt.Printf(i18n.T("cmd.match")+"\n", sn)
			}
		}
	}

	return nil
}

func CmdUpdate(root, id, title, text string, tags []string) error {
	root = defaultRoot(root)
	s, err := store.Open(root)
	if err != nil {
		return err
	}
	defer s.Close()

	old, err := s.GetByID(id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	n := &model.Note{
		ID:        old.ID,
		Title:     title,
		Text:      text,
		Tags:      tags,
		CreatedAt: old.CreatedAt,
		UpdatedAt: now,
		Deleted:   false,
	}

	return s.Append(n)
}

func CmdDelete(root, id string) error {
	root = defaultRoot(root)
	s, err := store.Open(root)
	if err != nil {
		return err
	}
	defer s.Close()

	old, err := s.GetByID(id)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	tomb := &model.Note{
		ID:        old.ID,
		Title:     old.Title,
		Text:      old.Text,
		Tags:      old.Tags,
		CreatedAt: old.CreatedAt,
		UpdatedAt: now,
		Deleted:   true,
	}

	return s.Append(tomb)
}

func CmdImport(root, dir, extList string, dryRun, verbose bool) error {
	root = defaultRoot(root)

	dir = filepath.Clean(dir)
	if info, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("каталог %q не существует", dir)
		}
		return err
	} else if !info.IsDir() {
		return fmt.Errorf("%q не является каталогом", dir)
	}

	exts := parseExtList(extList)

	rep, err := importer.ImportDir(root, dir, exts, dryRun)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Импорт из каталога: %s\n", rep.SourceDir)
	fmt.Fprintf(os.Stdout, "Корень хранилища: %s\n\n", rep.Root)

	fmt.Fprintf(os.Stdout, "Всего подходящих файлов: %d\n", rep.TotalFiles)
	fmt.Fprintf(os.Stdout, "Распознано как заметки: %d\n", rep.Parsed)
	fmt.Fprintf(os.Stdout, "Создано новых заметок: %d\n", rep.Created)
	fmt.Fprintf(os.Stdout, "Обновлено заметок: %d\n", rep.Updated)
	fmt.Fprintf(os.Stdout, "Пропущено (без изменений): %d\n", rep.Skipped)
	fmt.Fprintf(os.Stdout, "Ошибок: %d\n", rep.Errors)

	if verbose && len(rep.Results) > 0 {
		fmt.Fprintln(os.Stdout)
		for _, r := range rep.Results {
			line := fmt.Sprintf("%-8s %s", r.Action, r.Path)
			if r.Error != "" {
				line += " (" + r.Error + ")"
			}
			fmt.Fprintln(os.Stdout, line)
		}
	}

	if dryRun {
		fmt.Fprintln(os.Stdout, "\nРежим dry-run: хранилище не изменено.")
	}

	return nil
}

func parseExtList(list string) []string {
	list = strings.TrimSpace(list)
	if list == "" {
		return nil
	}
	parts := strings.Split(list, ",")
	var exts []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, ".") {
			p = "." + p
		}
		exts = append(exts, strings.ToLower(p))
	}
	return exts
}

var tokenRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

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

func highlightSnippet(n *model.Note, query string) string {
	toks := extractTokens(query)
	if len(toks) == 0 {
		return ""
	}

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

	esc := make([]string, 0, len(toks))
	for _, t := range toks {
		esc = append(esc, regexp.QuoteMeta(t))
	}
	pattern := `(?i)(` + strings.Join(esc, `|`) + `)`
	re := regexp.MustCompile(pattern)

	loc := re.FindStringIndex(txt)

	if loc == nil {
		tags := strings.Join(n.Tags, " ")
		loc = re.FindStringIndex(tags)
		if loc == nil {
			return ""
		}
		txt = tags
	}

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

	highlighted := re.ReplaceAllStringFunc(snippet, func(m string) string {

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
