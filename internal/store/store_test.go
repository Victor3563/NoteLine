package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/model"
)

func TestEnsureCreatesManifestAndSegmentsDir(t *testing.T) {
	root := t.TempDir()

	if err := Ensure(root); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	segDir := filepath.Join(root, dirSegments)
	if fi, err := os.Stat(segDir); err != nil || !fi.IsDir() {
		t.Fatalf("segments dir not created correctly, err=%v, fi=%v", err, fi)
	}

	manPath := filepath.Join(root, filenameManifest)
	if _, err := os.Stat(manPath); err != nil {
		t.Fatalf("manifest.json not created: %v", err)
	}
}

func TestAppendAndGetByID(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	n := model.NewNote("Title", "Body", []string{"go"})
	if err := s.Append(n); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := s.GetByID(n.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != n.ID || got.Title != n.Title || got.Text != n.Text {
		t.Fatalf("GetByID returned %+v, want %+v", got, n)
	}
}

func TestListWithTagAndContains(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	n1 := model.NewNote("Hello world", "first", []string{"go", "cli"})
	n2 := model.NewNote("Other", "something else", []string{"misc"})

	if err := s.Append(n1); err != nil {
		t.Fatalf("Append n1: %v", err)
	}
	if err := s.Append(n2); err != nil {
		t.Fatalf("Append n2: %v", err)
	}

	list, err := s.List(Filter{Tag: "go"})
	if err != nil {
		t.Fatalf("List(Tag): %v", err)
	}
	if len(list) != 1 || list[0].ID != n1.ID {
		t.Fatalf("List(Tag) = %#v, want only n1", list)
	}

	list2, err := s.List(Filter{Contains: "world"})
	if err != nil {
		t.Fatalf("List(Contains): %v", err)
	}
	if len(list2) == 0 {
		t.Fatalf("List(Contains) returned no results, want at least 1")
	}
	found := false
	for _, n := range list2 {
		if n.ID == n1.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("List(Contains) did not return note with ID %q, got %#v", n1.ID, list2)
	}
}

func TestListLimitAndSorting(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	for i := 0; i < 3; i++ {
		n := model.NewNote(
			fmt.Sprintf("N%d", i),
			"text",
			nil,
		)

		n.CreatedAt = time.Now().Add(time.Duration(i) * time.Minute)
		n.UpdatedAt = n.CreatedAt
		if err := s.Append(n); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	list, err := s.List(Filter{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List Limit=2 returned %d, want 2", len(list))
	}
	if !list[0].CreatedAt.After(list[1].CreatedAt) && !list[0].CreatedAt.Equal(list[1].CreatedAt) {
		t.Fatalf("expected list to be sorted by CreatedAt desc, got:\n%v\n%v", list[0].CreatedAt, list[1].CreatedAt)
	}
}

func TestSoftDelete(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	n := model.NewNote("Title", "Body", nil)
	if err := s.Append(n); err != nil {
		t.Fatalf("Append: %v", err)
	}

	tomb := *n
	tomb.Deleted = true
	tomb.UpdatedAt = time.Now().UTC()
	if err := s.Append(&tomb); err != nil {
		t.Fatalf("Append tombstone: %v", err)
	}

	if _, err := s.GetByID(n.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID after delete = %v, want ErrNotFound", err)
	}
}

func TestRotateCreatesNewSegment(t *testing.T) {
	root := t.TempDir()
	s, err := Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	s.man.SegmentSizeBytes = 64

	n1 := model.NewNote("Title1", strings.Repeat("x", 200), nil)
	n2 := model.NewNote("Title2", strings.Repeat("y", 200), nil)

	if err := s.Append(n1); err != nil {
		t.Fatalf("Append n1: %v", err)
	}
	if err := s.Append(n2); err != nil {
		t.Fatalf("Append n2: %v", err)
	}

	segDir := filepath.Join(root, dirSegments)
	files, _ := filepath.Glob(filepath.Join(segDir, "notes-*.ndjson"))
	if len(files) < 2 {
		t.Fatalf("expected at least 2 segment files after rotation, got %d", len(files))
	}
}
