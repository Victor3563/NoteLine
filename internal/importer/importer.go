package importer

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/model"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/store"
)

type FileResult struct {
	Path   string `json:"path"`
	Action string `json:"action"` // created, updated, skipped, error
	Error  string `json:"error,omitempty"`
}

type Report struct {
	Root       string       `json:"root"`
	SourceDir  string       `json:"source_dir"`
	TotalFiles int          `json:"total_files"`
	Parsed     int          `json:"parsed"`
	Created    int          `json:"created"`
	Updated    int          `json:"updated"`
	Skipped    int          `json:"skipped"`
	Errors     int          `json:"errors"`
	Results    []FileResult `json:"results,omitempty"`
}

// информация о сопоставлении внешнего источника и заметки
type sourceInfo struct {
	NoteID      string `json:"note_id"`
	Path        string `json:"path"`          // относительный путь
	ContentHash string `json:"content_hash"`  // sha1(title+tags+body)
	ModTimeUnix int64  `json:"mod_time_unix"` // время изменения файла
}

type importIndex struct {
	Version int                   `json:"version"`
	Sources map[string]sourceInfo `json:"sources"` // key -> info
}

func ImportDir(root, dir string, exts []string, dryRun bool) (*Report, error) {
	if dir == "" {
		return nil, fmt.Errorf("пустой каталог импорта")
	}

	// приведение расширений к нормальному виду
	extSet := make(map[string]bool)
	for _, e := range exts {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		extSet[e] = true
	}
	// если не заданы — используем разумный дефолт
	if len(extSet) == 0 {
		extSet[".md"] = true
		extSet[".markdown"] = true
		extSet[".txt"] = true
	}

	s, err := store.Open(root)
	if err != nil {
		return nil, err
	}
	defer s.Close()

	idx, err := loadIndex(root)
	if err != nil {
		return nil, err
	}

	rep := &Report{
		Root:      root,
		SourceDir: dir,
	}

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			rep.Errors++
			rep.Results = append(rep.Results, FileResult{
				Path:   path,
				Action: "error",
				Error:  walkErr.Error(),
			})
			return nil
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			return nil // пропускаем скрытые файлы
		}
		ext := strings.ToLower(filepath.Ext(name))
		if !extSet[ext] {
			return nil
		}

		rep.TotalFiles++

		data, err := os.ReadFile(path)
		if err != nil {
			rep.Errors++
			rep.Results = append(rep.Results, FileResult{
				Path:   path,
				Action: "error",
				Error:  err.Error(),
			})
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			rel = path
		}

		info, err := d.Info()
		if err != nil {
			rep.Errors++
			rep.Results = append(rep.Results, FileResult{
				Path:   path,
				Action: "error",
				Error:  err.Error(),
			})
			return nil
		}

		meta, body := splitFrontMatter(string(data))
		note := buildNoteFromMarkdown(meta, body, rel, info)
		sourceKey := sourceKeyFor(meta, rel)
		rep.Parsed++

		// хэш содержимого для детекта изменений
		contentHash := hashNoteContent(note)

		// старая запись (если уже импортировали)
		entry, existed := idx.Sources[sourceKey]

		if existed && entry.ContentHash == contentHash {
			// содержимое не изменилось — пропускаем
			rep.Skipped++
			rep.Results = append(rep.Results, FileResult{
				Path:   rel,
				Action: "skipped",
			})
			// можно обновить путь/mtime только в индексе
			if !dryRun {
				entry.Path = rel
				entry.ModTimeUnix = info.ModTime().UTC().Unix()
				idx.Sources[sourceKey] = entry
			}
			return nil
		}

		now := time.Now().UTC()

		if existed {
			// обновление существующей заметки
			old, err := s.GetByID(entry.NoteID)
			if err != nil && !errors.Is(err, store.ErrNotFound) {
				rep.Errors++
				rep.Results = append(rep.Results, FileResult{
					Path:   rel,
					Action: "error",
					Error:  err.Error(),
				})
				return nil
			}

			if old != nil {
				note.ID = old.ID
				// сохраняем оригинальное CreatedAt (из заметки или front matter)
				if note.CreatedAt.IsZero() {
					note.CreatedAt = old.CreatedAt
				}
			}
			if note.CreatedAt.IsZero() {
				note.CreatedAt = now
			}
			if note.UpdatedAt.IsZero() {
				note.UpdatedAt = now
			}

			if !dryRun {
				if err := s.Append(note); err != nil {
					rep.Errors++
					rep.Results = append(rep.Results, FileResult{
						Path:   rel,
						Action: "error",
						Error:  err.Error(),
					})
					return nil
				}
				idx.Sources[sourceKey] = sourceInfo{
					NoteID:      note.ID,
					Path:        rel,
					ContentHash: contentHash,
					ModTimeUnix: info.ModTime().UTC().Unix(),
				}
			}

			rep.Updated++
			rep.Results = append(rep.Results, FileResult{
				Path:   rel,
				Action: "updated",
			})
			return nil
		}

		// новая заметка
		tmp := model.NewNote(note.Title, note.Text, note.Tags)
		if note.ID == "" {
			note.ID = tmp.ID
		}
		if note.CreatedAt.IsZero() {
			note.CreatedAt = tmp.CreatedAt
		}
		if note.UpdatedAt.IsZero() {
			note.UpdatedAt = note.CreatedAt
		}

		if !dryRun {
			if err := s.Append(note); err != nil {
				rep.Errors++
				rep.Results = append(rep.Results, FileResult{
					Path:   rel,
					Action: "error",
					Error:  err.Error(),
				})
				return nil
			}
			idx.Sources[sourceKey] = sourceInfo{
				NoteID:      note.ID,
				Path:        rel,
				ContentHash: contentHash,
				ModTimeUnix: info.ModTime().UTC().Unix(),
			}
		}

		rep.Created++
		rep.Results = append(rep.Results, FileResult{
			Path:   rel,
			Action: "created",
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	if !dryRun {
		if err := saveIndex(root, idx); err != nil {
			return nil, err
		}
	}

	return rep, nil
}

func loadIndex(root string) (*importIndex, error) {
	path := filepath.Join(root, "imports.json")
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &importIndex{
				Version: 1,
				Sources: make(map[string]sourceInfo),
			}, nil
		}
		return nil, err
	}
	var idx importIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		// если файл битый — начинаем с нуля
		return &importIndex{
			Version: 1,
			Sources: make(map[string]sourceInfo),
		}, nil
	}
	if idx.Sources == nil {
		idx.Sources = make(map[string]sourceInfo)
	}
	return &idx, nil
}

func saveIndex(root string, idx *importIndex) error {
	path := filepath.Join(root, "imports.json")
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// splitFrontMatter отделяет front matter в формате:
//
// ---
// key: value
// tags: a, b, c
// ---
// тело markdown...
func splitFrontMatter(content string) (map[string]string, string) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return map[string]string{}, ""
	}

	if strings.TrimSpace(lines[0]) != "---" {
		// нет front matter
		return map[string]string{}, content
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		// нет закрывающего ---
		return map[string]string{}, content
	}

	metaLines := lines[1:end]
	bodyLines := []string{}
	if end+1 < len(lines) {
		bodyLines = lines[end+1:]
	}
	body := strings.Join(bodyLines, "\n")

	meta := make(map[string]string)
	for _, line := range metaLines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		meta[key] = value
	}

	return meta, body
}

func buildNoteFromMarkdown(meta map[string]string, body, relpath string, info fs.FileInfo) *model.Note {
	title := strings.TrimSpace(meta["title"])
	tags := parseTags(meta["tags"])

	var created time.Time
	if v := strings.TrimSpace(meta["created"]); v != "" {
		if t, err := parseTimeFlexible(v); err == nil {
			created = t.UTC()
		}
	}
	var updated time.Time
	if v := strings.TrimSpace(meta["updated"]); v != "" {
		if t, err := parseTimeFlexible(v); err == nil {
			updated = t.UTC()
		}
	}

	if created.IsZero() && info != nil {
		created = info.ModTime().UTC()
	}
	if created.IsZero() {
		created = time.Now().UTC()
	}
	if updated.IsZero() {
		updated = created
	}

	// sourceKey будет собран отдельно, здесь только Note
	return &model.Note{
		// ID заполним позже
		Title:     title,
		Text:      body,
		Tags:      tags,
		CreatedAt: created,
		UpdatedAt: updated,
		Deleted:   false,
	}
}

// sourceKey строится как:
//
//	если есть meta["id"] -> "id:<значение>"
//	иначе                -> "path:<relpath>"
func sourceKeyFor(meta map[string]string, relpath string) string {
	if id := strings.TrimSpace(meta["id"]); id != "" {
		return "id:" + id
	}
	return "path:" + filepath.ToSlash(relpath)
}

func hashNoteContent(n *model.Note) string {
	h := sha1.Sum([]byte(n.Title + "\n" + strings.Join(n.Tags, ",") + "\n" + n.Text))
	return hex.EncodeToString(h[:])
}

func parseTags(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	// убираем возможные скобки [a, b]
	if strings.HasPrefix(raw, "[") && strings.HasSuffix(raw, "]") {
		raw = strings.TrimSpace(raw[1 : len(raw)-1])
	}
	parts := strings.Split(raw, ",")
	var tags []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		t = strings.Trim(t, `"`)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func parseTimeFlexible(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time %q", s)
}
