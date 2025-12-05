package crash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReportIfPanicCreatesFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NOTELINE_CRASH_DIR", dir)
	t.Setenv("NOTELINE_CRASH_UPLOAD_URL", "") // не отправляем

	func() {
		defer ReportIfPanic("test-version")
		panic("test panic")
	}()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 crash file, got %d", len(entries))
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "test-version") || !strings.Contains(s, "panic=test panic") {
		t.Fatalf("crash report does not contain expected fragments:\n%s", s)
	}
}
