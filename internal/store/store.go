//Низкоуровневая запись заметок, то на что ссылается comands
//Удалить после прочтения

package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
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
	dirSegments          = "segments"
	filenameManifest     = "manifest.json"
	defaultSegSize   int = 8 * 1024 * 1024 // 8 MiB
	maxScanLine          = 10 * 1024 * 1024
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

// Ensure создаёт каталог хранилища.
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

func Open(root string) (*Store, error) {
	if strings.TrimSpace(root) == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".noteline")
	}
	if err := Ensure(root); err != nil {
		return nil, err
	}
	// read manifest
	var man manifest
	b, err := os.ReadFile(filepath.Join(root, filenameManifest))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &man); err != nil {
		return nil, err
	}
	s := &Store{root: root, man: man}
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
		// create first
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
	// обновляем manifest
	b, _ := json.MarshalIndent(s.man, "", "  ")
	return os.WriteFile(filepath.Join(s.root, filenameManifest), b, 0o644) //0o644 = (rw-r--r--)
}

// Append добавляет запись в конец активного сегмента (NDJSON).
func (s *Store) Append(n *model.Note) error {
	if s.active == nil {
		if err := s.openActiveSegmentRW(); err != nil {
			return err
		}
	}
	// проверка что файл нужного размера и открытие нового иначе
	st, err := s.active.Stat()
	if err == nil && st.Size() >= int64(s.man.SegmentSizeBytes) {
		if err := s.rotate(); err != nil {
			return err
		}
	}
	enc := json.NewEncoder(s.active)
	return enc.Encode(n)
}

// GetByID делает линейный поиск по всем сегментам, начиная с последнего.
func (s *Store) GetByID(id string) (*model.Note, error) {
	segDir := filepath.Join(s.root, dirSegments)
	files, _ := filepath.Glob(filepath.Join(segDir, "notes-*.ndjson"))
	if len(files) == 0 {
		return nil, ErrNotFound
	}
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		buf := make([]byte, 0, 1024*1024)
		sc.Buffer(buf, maxScanLine)
		for sc.Scan() {
			line := sc.Bytes()
			var n model.Note
			if err := json.Unmarshal(line, &n); err == nil {
				if n.ID == id && !n.Deleted {
					f.Close()
					return &n, nil
				}
			}
		}
		f.Close()
	}
	return nil, ErrNotFound
}

// List возвращает заметки, отфильтрованные без индекса.
func (s *Store) List(filter Filter) ([]model.Note, error) {
	segDir := filepath.Join(s.root, dirSegments)
	files, _ := filepath.Glob(filepath.Join(segDir, "notes-*.ndjson"))
	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	var out []model.Note
	seen := make(map[string]bool) // если в будущих версиях появятся обновления — брать последний
	for _, path := range files {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		buf := make([]byte, 0, 1024*1024)
		sc.Buffer(buf, maxScanLine)
		for sc.Scan() {
			var n model.Note
			if err := json.Unmarshal(sc.Bytes(), &n); err != nil {
				continue
			}
			if n.Deleted {
				seen[n.ID] = true
				continue
			}
			if seen[n.ID] {
				continue
			}
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
			if filter.Contains != "" {
				if !strings.Contains(strings.ToLower(n.Title+" "+n.Text), strings.ToLower(filter.Contains)) {
					continue
				}
			}
			out = append(out, n)
			seen[n.ID] = true
			if filter.Limit > 0 && len(out) >= filter.Limit {
				f.Close()
				return out, nil
			}
		}
		f.Close()
	}
	// стабильно отсортируем по времени (новые сверху)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}
