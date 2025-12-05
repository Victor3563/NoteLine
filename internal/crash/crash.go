package crash

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"
)

func ReportIfPanic(version string) {
	if r := recover(); r != nil {
		data := buildReport(version, r)
		path := writeReport(data)
		_, _ = fmt.Fprintf(os.Stderr,
			"\nnoteline: unexpected panic, crash report saved to %s\n", path)
		maybeUpload(data)

	}
}

func buildReport(version string, r any) []byte {
	var buf bytes.Buffer
	now := time.Now().UTC()

	fmt.Fprintf(&buf, "time=%s\n", now.Format(time.RFC3339))
	fmt.Fprintf(&buf, "version=%s\n", version)
	fmt.Fprintf(&buf, "go=%s\n", runtime.Version())
	fmt.Fprintf(&buf, "os=%s\narch=%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&buf, "args=%q\n", os.Args)
	fmt.Fprintf(&buf, "panic=%v\n\n", r)
	buf.WriteString("stack:\n")
	buf.Write(debug.Stack())
	buf.WriteByte('\n')

	return buf.Bytes()
}

func crashRoot() string {
	if dir := os.Getenv("NOTELINE_CRASH_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	if home == "" {
		return "."
	}
	return filepath.Join(home, ".noteline", "crash")
}

func writeReport(data []byte) string {
	root := crashRoot()
	_ = os.MkdirAll(root, 0o755)
	ts := time.Now().UTC().Format("20060102-150405")
	path := filepath.Join(root, "crash-"+ts+".log")
	_ = os.WriteFile(path, data, 0o600)
	return path
}

func maybeUpload(data []byte) {
	url := os.Getenv("NOTELINE_CRASH_UPLOAD_URL")
	if url == "" {
		return
	}
	go func() {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "text/plain; charset=utf-8")
		client := &http.Client{Timeout: 3 * time.Second}
		_, _ = client.Do(req)
	}()
}
