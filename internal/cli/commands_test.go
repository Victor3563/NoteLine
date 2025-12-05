package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCmdCreateReadUpdateDelete(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")

	if err := CmdInit(root); err != nil {
		t.Fatalf("CmdInit: %v", err)
	}

	id, err := CmdCreate(root, "Title", "Text", []string{"go", "cli"})
	if err != nil {
		t.Fatalf("CmdCreate: %v", err)
	}
	if id == "" {
		t.Fatalf("CmdCreate returned empty id")
	}

	if err := CmdRead(root, id, false); err != nil {
		t.Fatalf("CmdRead: %v", err)
	}

	if err := CmdList(root, "", "", 0, false); err != nil {
		t.Fatalf("CmdList: %v", err)
	}

	if err := CmdUpdate(root, id, "New title", "New text", []string{"go", "updated"}); err != nil {
		t.Fatalf("CmdUpdate: %v", err)
	}

	if err := CmdDelete(root, id); err != nil {
		t.Fatalf("CmdDelete: %v", err)
	}
}

func TestCmdImport(t *testing.T) {
	root := filepath.Join(t.TempDir(), "store")
	if err := CmdInit(root); err != nil {
		t.Fatalf("CmdInit: %v", err)
	}

	src := filepath.Join(t.TempDir(), "notes")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	content := `---
title: Imported
tags: cli
---
Body
`
	path := filepath.Join(src, "note.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := CmdImport(root, src, "md", true, true); err != nil {
		t.Fatalf("CmdImport dry-run: %v", err)
	}

	if err := CmdImport(root, src, "md", false, true); err != nil {
		t.Fatalf("CmdImport real: %v", err)
	}

	if err := CmdList(root, "", "Imported", 10, false); err != nil {
		t.Fatalf("CmdList after import: %v", err)
	}
}
