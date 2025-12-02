package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/model"
)

// Я погуглил и решил что будем хранить заметки в файлах по 8мб, чтобы если файл повреждался
// мы не теряли все заметки, и при этом не было слишком много файлов
const (
	dirSegments      = "segments"
	filenameManifest = "manifest.json"
	// размер сегмента по умолчанию (в байтах)
	defaultSegSize int = 8 * 1024 * 1024 // 8 MiB
)

var ErrNotFound = errors.New("note not found")

type manifest struct {
	Version          int   `json:"version"`
	SegmentSizeBytes int   `json:"segment_size_bytes"`
	NextSegmentSeq   int   `json:"next_segment_seq"`
	CreatedAtUnix    int64 `json:"created_at_unix"`
}

type Store struct {
	root     string
	man      manifest
	active   *os.File
	activeNo int
}

type Filter struct {
	Tag      string
	Contains string
	Limit    int
}

// Ensure создаёт каталог хранилища и manifest.json, если их ещё нет.
func Ensure(root string) error {
	if strings.TrimSpace(root) == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".noteline")
	}
	if err := os.MkdirAll(filepath.Join(root, dirSegments), 0o755); err != nil { // Права 0o755 (rwxr-xr-x)
		return err
	}

	manPath := filepath.Join(root, filenameManifest)
	if _, err := os.Stat(manPath); errors.Is(err, os.ErrNotExist) {
		m := manifest{
			Version:          1,
			SegmentSizeBytes: defaultSegSize,
			NextSegmentSeq:   1,
			CreatedAtUnix:    time.Now().UTC().Unix(),
		}
		f, err := os.Create(manPath)
		if err != nil {
			return err
		}
		defer f.Close()

		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(m); err != nil {
			return err
		}
	}
	return nil
}

// Open открывает хранилище и активный сегмент для записи.
func Open(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".noteline")
	}
	if err := Ensure(root); err != nil {
		return nil, err
	}

	// читаем manifest
	var man manifest
	b, err := os.ReadFile(filepath.Join(root, filenameManifest))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &man); err != nil {
		return nil, err
	}

	s := &Store{
		root: root,
		man:  man,
	}

	if err := s.openActiveSegmentRW(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	if s.active != nil {
		return s.active.Close()
	}
	return nil
}

func (s *Store) openActiveSegmentRW() error {
	segDir := filepath.Join(s.root, dirSegments)
	files, _ := filepath.Glob(filepath.Join(segDir, "notes-*.ndjson"))
	if len(files) == 0 {
		// первый сегмент
		return s.rotate()
	}

	sort.Strings(files)
	last := files[len(files)-1]
	f, err := os.OpenFile(last, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}
	s.active = f
	no, _ := parseSeqFromName(filepath.Base(last))
	s.activeNo = no
	return nil
}

func parseSeqFromName(name string) (int, error) {
	// notes-00000001.ndjson
	name = strings.TrimPrefix(name, "notes-")
	name = strings.TrimSuffix(name, ".ndjson")
	name = strings.TrimLeft(name, "0")
	if name == "" {
		return 0, nil
	}
	return strconv.Atoi(name)
}

func (s *Store) rotate() error {
	segDir := filepath.Join(s.root, dirSegments)
	name := fmt.Sprintf("notes-%08d.ndjson", s.man.NextSegmentSeq)
	path := filepath.Join(segDir, name)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o644)
	if err != nil {
		return err
	}

	if s.active != nil {
		_ = s.active.Close()
	}

	s.active = f
	s.activeNo = s.man.NextSegmentSeq
	s.man.NextSegmentSeq++

	// сохраняем manifest
	b, err := json.MarshalIndent(s.man, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.root, filenameManifest), b, 0o644)
}

// Append добавляет запись в конец активного сегмента (NDJSON).
func (s *Store) Append(n *model.Note) error {
	if s.active == nil {
		if err := s.openActiveSegmentRW(); err != nil {
			return err
		}
	}

	// кодируем заметку в JSON, чтобы знать размер
	b, err := json.Marshal(n)
	if err != nil {
		return err
	}

	// ротация по size (учитываем \n)
	if st, err := s.active.Stat(); err == nil {
		if st.Size()+int64(len(b)+1) > int64(s.man.SegmentSizeBytes) {
		if err := s.rotate(); err != nil {
			return err
		}
	}
	}

	if _, err := s.active.Write(b); err != nil {
		return err
	}
	if _, err := s.active.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

// loadAllNotes читает все сегменты и собирает последнюю версию каждой заметки.
func (s *Store) loadAllNotes() (map[string]model.Note, error) {
	segDir := filepath.Join(s.root, dirSegments)
	files, _ := filepath.Glob(filepath.Join(segDir, "notes-*.ndjson"))
	sort.Strings(files) // от старого к новому

	notes := make(map[string]model.Note)

	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			continue
		}

		dec := json.NewDecoder(f)
		for {
			var n model.Note
			if err := dec.Decode(&n); err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				// битую запись просто пропускаем
				continue
			}

			if n.Deleted {
				// "мягкое" удаление
				delete(notes, n.ID)
				continue
			}

			// последняя версия по ID перезаписывает предыдущие
			notes[n.ID] = n
		}

		_ = f.Close()
	}

	return notes, nil
}

// GetByID возвращает последнюю версию заметки по ID.
func (s *Store) GetByID(id string) (*model.Note, error) {
	notes, err := s.loadAllNotes()
	if err != nil {
		return nil, err
	}
	n, ok := notes[id]
	if !ok {
		return nil, ErrNotFound
	}
	nCopy := n
	return &nCopy, nil
}

// List возвращает заметки, отфильтрованные без постоянного индекса.
func (s *Store) List(filter Filter) ([]model.Note, error) {
	notes, err := s.loadAllNotes()
	if err != nil {
		return nil, err
	}

	var out []model.Note

	for _, n := range notes {
		// фильтр по тегу
			if filter.Tag != "" {
				found := false
				for _, t := range n.Tags {
					if t == filter.Tag {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

		// фильтр по подстроке (title + text)
			if filter.Contains != "" {
			hay := strings.ToLower(n.Title + " " + n.Text)
			needle := strings.ToLower(filter.Contains)
			if !strings.Contains(hay, needle) {
					continue
				}
			}

			out = append(out, n)
	}

	// сортируем: новые сверху (по времени создания)
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})

	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}

	return out, nil
}
