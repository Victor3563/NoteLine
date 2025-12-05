package model

import "testing"

func TestNewNoteFields(t *testing.T) {
	n := NewNote("Title", "Body", []string{"go", "cli"})

	if n.ID == "" {
		t.Fatalf("expected non-empty ID")
	}
	if n.Title != "Title" {
		t.Fatalf("Title=%q, want %q", n.Title, "Title")
	}
	if n.Text != "Body" {
		t.Fatalf("Text=%q, want %q", n.Text, "Body")
	}
	if len(n.Tags) != 2 || n.Tags[0] != "go" || n.Tags[1] != "cli" {
		t.Fatalf("Tags=%v, want [go cli]", n.Tags)
	}
	if n.CreatedAt.IsZero() || n.UpdatedAt.IsZero() {
		t.Fatalf("CreatedAt/UpdatedAt must not be zero")
	}
	if !n.CreatedAt.Equal(n.UpdatedAt) {
		t.Fatalf("CreatedAt and UpdatedAt must be equal for new note")
	}
	if n.Deleted {
		t.Fatalf("Deleted must be false for new note")
	}
}

func TestNewNoteIdIsUnique(t *testing.T) {
	n1 := NewNote("T", "X", nil)
	n2 := NewNote("T", "X", nil)

	if n1.ID == n2.ID {
		t.Fatalf("expected different IDs, got %q", n1.ID)
	}
}
