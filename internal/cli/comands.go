// Вверхнеуровневые команды для main
// После прочтения удалить
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		fmt.Printf("tags: %s\n", strings.Join(n.Tags, ", "))
	}
	fmt.Printf("created: %s\n", n.CreatedAt.Format("2006-01-02 15:04:05"))
	if !n.UpdatedAt.IsZero() {
		fmt.Printf("updated: %s\n", n.UpdatedAt.Format("2006-01-02 15:04:05"))
	}
	fmt.Println("---")
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
			fmt.Printf("  tags: %s\n", strings.Join(n.Tags, ", "))
		}
		fmt.Printf("  created: %s\n", n.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}
