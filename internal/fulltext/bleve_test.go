package fulltext

import (
	"testing"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/model"
)

func TestInitIndexAndSearch(t *testing.T) {
	root := t.TempDir()

	if err := Init(root); err != nil {
		t.Fatalf("Init: %v", err)
	}
	defer func() {
		if err := Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	n1 := &model.Note{ID: "1", Title: "Hello world", Text: "test note", Tags: []string{"go", "cli"}}
	n2 := &model.Note{ID: "2", Title: "Other", Text: "something about bleve", Tags: []string{"search"}}

	if err := IndexNote(n1); err != nil {
		t.Fatalf("IndexNote(n1): %v", err)
	}
	if err := IndexNote(n2); err != nil {
		t.Fatalf("IndexNote(n2): %v", err)
	}

	ids, err := Search("Hello", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(ids) == 0 {
		t.Fatalf("Search returned no hits, want at least 1")
	}

	found := false
	for _, id := range ids {
		if id == n1.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find note ID %q in search results, got %v", n1.ID, ids)
	}

	if _, err := Search("Hello", 10); err != nil {
		t.Fatalf("Search (cached) failed: %v", err)
	}
}
