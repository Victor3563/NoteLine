package importer

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/model"
)

func TestSplitFrontMatterNone(t *testing.T) {
	content := "line1\nline2\n"
	meta, body := splitFrontMatter(content)

	if len(meta) != 0 {
		t.Fatalf("expected empty meta, got %v", meta)
	}
	if body != content {
		t.Fatalf("body = %q, want %q", body, content)
	}
}

func TestSplitFrontMatterWithMeta(t *testing.T) {
	content := `---
title: My note
tags: go, cli
created: 2025-01-02
---
Body
text
`
	meta, body := splitFrontMatter(content)

	if meta["title"] != "My note" {
		t.Fatalf("title=%q, want %q", meta["title"], "My note")
	}
	if meta["tags"] != "go, cli" {
		t.Fatalf("tags=%q, want %q", meta["tags"], "go, cli")
	}
	if meta["created"] != "2025-01-02" {
		t.Fatalf("created=%q, want %q", meta["created"], "2025-01-02")
	}
	wantBody := "Body\ntext\n"
	if body != wantBody {
		t.Fatalf("body=%q, want %q", body, wantBody)
	}
}

func TestParseTags(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"go,cli", []string{"go", "cli"}},
		{" go , cli ,  dev ", []string{"go", "cli", "dev"}},
		{`["a","b"]`, []string{"a", "b"}},
		{`[a, b]`, []string{"a", "b"}},
	}
	for _, tt := range tests {
		got := parseTags(tt.in)
		if len(got) != len(tt.want) {
			t.Fatalf("parseTags(%q) len=%d, want %d", tt.in, len(got), len(tt.want))
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("parseTags(%q)[%d]=%q, want %q", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

func TestParseTimeFlexible(t *testing.T) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
	}
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)

	for _, l := range layouts {
		s := now.Format(l)
		got, err := parseTimeFlexible(s)
		if err != nil {
			t.Fatalf("parseTimeFlexible(%q) error: %v", s, err)
		}
		if got.IsZero() {
			t.Fatalf("parseTimeFlexible(%q) returned zero time", s)
		}
	}
}

func TestHashNoteContentChanges(t *testing.T) {
	n1 := &model.Note{Title: "A", Text: "B", Tags: []string{"x"}}
	n2 := &model.Note{Title: "A", Text: "B", Tags: []string{"x"}}
	n3 := &model.Note{Title: "A", Text: "C", Tags: []string{"x"}}

	h1 := hashNoteContent(n1)
	h2 := hashNoteContent(n2)
	h3 := hashNoteContent(n3)

	if h1 != h2 {
		t.Fatalf("same content should yield same hash, got %q vs %q", h1, h2)
	}
	if h1 == h3 {
		t.Fatalf("different content should yield different hash, got %q vs %q", h1, h3)
	}
}

func TestSourceKeyFor(t *testing.T) {
	meta := map[string]string{"id": "my-id"}
	if got := sourceKeyFor(meta, "path/to/file.md"); got != "id:my-id" {
		t.Fatalf("sourceKeyFor with id = %q, want %q", got, "id:my-id")
	}

	meta2 := map[string]string{}
	if got := sourceKeyFor(meta2, "path/to/file.md"); got != "path:path/to/file.md" {
		t.Fatalf("sourceKeyFor path = %q, want %q", got, "path:path/to/file.md")
	}
}

type fakeInfo struct {
	mod time.Time
}

func (f fakeInfo) Name() string       { return "fake" }
func (f fakeInfo) Size() int64        { return 0 }
func (f fakeInfo) Mode() os.FileMode  { return 0 }
func (f fakeInfo) ModTime() time.Time { return f.mod }
func (f fakeInfo) IsDir() bool        { return false }
func (f fakeInfo) Sys() any           { return nil }

func TestBuildNoteFromMarkdown(t *testing.T) {
	meta := map[string]string{
		"title":   "Title",
		"tags":    "a, b",
		"created": "2025-01-02",
		"updated": "2025-01-03T10:00:00Z",
	}
	body := "Body"
	mod := time.Date(2025, 1, 4, 0, 0, 0, 0, time.UTC)
	info := fakeInfo{mod: mod}

	n := buildNoteFromMarkdown(meta, body, "note.md", info)

	if n.Title != "Title" {
		t.Fatalf("Title=%q, want %q", n.Title, "Title")
	}
	if len(n.Tags) != 2 || n.Tags[0] != "a" || n.Tags[1] != "b" {
		t.Fatalf("Tags=%v, want [a b]", n.Tags)
	}
	if n.Text != body {
		t.Fatalf("Text=%q, want %q", n.Text, body)
	}
	if n.CreatedAt.IsZero() || n.UpdatedAt.IsZero() {
		t.Fatalf("CreatedAt/UpdatedAt should not be zero")
	}
}

func TestImportDirCreatesAndSkips(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	content := `---
title: Note
tags: go, cli
---
Body
`
	path := filepath.Join(src, "note.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	rep, err := ImportDir(root, src, []string{".md"}, false)
	if err != nil {
		t.Fatalf("ImportDir: %v", err)
	}
	if rep.Created != 1 || rep.TotalFiles != 1 {
		t.Fatalf("first import: Created=%d TotalFiles=%d, want 1/1", rep.Created, rep.TotalFiles)
	}

	rep2, err := ImportDir(root, src, []string{".md"}, false)
	if err != nil {
		t.Fatalf("ImportDir second: %v", err)
	}
	if rep2.Skipped != 1 {
		t.Fatalf("second import: Skipped=%d, want 1", rep2.Skipped)
	}
}
