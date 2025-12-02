package fulltext

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/blevesearch/bleve/v2"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/lru"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/model"
)

var (
	mu          sync.Mutex
	idx         bleve.Index
	searchCache *lru.LRU
)

// Init открывает (или создаёт) Bleve индекс в root (путь: <root>/index.bleve).
// Вызывается из store.Open(root).
func Init(root string) error {
	mu.Lock()
	defer mu.Unlock()
	if idx != nil {
		return nil
	}
	path := filepath.Join(root, "index.bleve")
	// Try open
	i, err := bleve.Open(path)
	if err == nil {
		idx = i
		searchCache = lru.New(1024)
		return nil
	}
	// create new
	mapping := bleve.NewIndexMapping()
	i, err = bleve.New(path, mapping)
	if err != nil {
		return fmt.Errorf("fulltext: create index: %w", err)
	}
	idx = i
	searchCache = lru.New(1024)
	return nil
}

// Close closes index (best-effort).
func Close() error {
	mu.Lock()
	defer mu.Unlock()
	if idx == nil {
		return nil
	}
	err := idx.Close()
	idx = nil
	if searchCache != nil {
		searchCache.Clear()
	}
	return err
}

// IndexNote индексирует заметку (title, text, tags).
// Best-effort: возвращает ошибку чтобы caller мог логировать.
func IndexNote(n *model.Note) error {
	mu.Lock()
	defer mu.Unlock()
	if idx == nil {
		return fmt.Errorf("fulltext: index not initialized")
	}
	// document we index: fields must be exported
	doc := struct {
		Title string
		Text  string
		Tags  string
	}{
		Title: n.Title,
		Text:  n.Text,
		Tags:  strings.Join(n.Tags, " "),
	}
	if err := idx.Index(n.ID, doc); err != nil {
		return err
	}

	if searchCache != nil {
		searchCache.Clear()
	}
	return nil

}

// Search returns note IDs matching query (size = max hits).
// The query uses Bleve query-string syntax (simple and powerful).
func Search(q string, size int) ([]string, error) {
	mu.Lock()
	defer mu.Unlock()
	if idx == nil {
		return nil, fmt.Errorf("fulltext: index not initialized")
	}
	if size <= 0 {
		size = 1000
	}
	key := fmt.Sprintf("%s|%d", q, size)
	if searchCache != nil {
		if v, ok := searchCache.Get(key); ok {
			if ids, ok2 := v.([]string); ok2 {
				out := make([]string, len(ids))
				copy(out, ids)
				return out, nil
			}
		}
	}

	qq := bleve.NewQueryStringQuery(q)
	req := bleve.NewSearchRequestOptions(qq, size, 0, false)
	res, err := idx.Search(req)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(res.Hits))
	for _, h := range res.Hits {
		out = append(out, h.ID)
	}
	if searchCache != nil {
		cp := make([]string, len(out))
		copy(cp, out)
		searchCache.Add(key, cp)
	}
	return out, nil
}
