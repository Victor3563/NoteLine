package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadLocaleAndT(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "en.json")

	data := []byte(`{"hello":"world %d"}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := LoadLocale(path, "en"); err != nil {
		t.Fatalf("LoadLocale: %v", err)
	}

	if got := T("hello", 5); got != "world 5" {
		t.Fatalf("T(hello) = %q, want %q", got, "world 5")
	}

	if got := T("missing_key"); got != "missing_key" {
		t.Fatalf("T(missing_key) = %q, want %q", got, "missing_key")
	}
}

func TestInitFromEnvUsesCustomDir(t *testing.T) {
	dir := t.TempDir()
	locals := filepath.Join(dir, "locals")
	if err := os.MkdirAll(locals, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	path := filepath.Join(locals, "ru.json")
	data := []byte(`{"x":"y"}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	t.Setenv("NOTELINE_I18N_DIR", dir)
	t.Setenv("NOTELINE_LANG", "ru_RU.UTF-8")

	if err := InitFromEnv(); err != nil {
		t.Fatalf("InitFromEnv: %v", err)
	}

	if got := Locale(); got != "ru" {
		t.Fatalf("Locale() = %q, want %q", got, "ru")
	}
	if got := T("x"); got != "y" {
		t.Fatalf("T(x) = %q, want %q", got, "y")
	}
}

func TestUniqueStringsAndErrsToString(t *testing.T) {
	in := []string{"a", "b", "a", "", "b", "c"}
	out := uniqueStrings(in)
	want := []string{"a", "b", "c"}
	if len(out) != len(want) {
		t.Fatalf("uniqueStrings len=%d, want %d", len(out), len(want))
	}
	for i, v := range want {
		if out[i] != v {
			t.Fatalf("uniqueStrings[%d]=%q, want %q", i, out[i], v)
		}
	}

	if s := errsToString(nil); s != "" {
		t.Fatalf("errsToString(nil) = %q, want empty", s)
	}
}
