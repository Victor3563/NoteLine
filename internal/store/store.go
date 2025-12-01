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
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	fts "github.com/Victor3563/NoteLine/cli-notebook/internal/fulltext"
	"github.com/Victor3563/NoteLine/cli-notebook/internal/i18n"

	"github.com/Victor3563/NoteLine/cli-notebook/internal/lru"
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
var noteCache *lru.LRU

type manifest struct {
	Version          int   `json:"version"`
	SegmentSizeBytes int   `json:"segment_size_bytes"`
	NextSegmentSeq   int   `json:"next_segment_seq"`
	CreatedAtUnix    int64 `json:"created_at_unix"`
}

type Store struct {
	root      string
	man       manifest
	active    *os.File
	activeNo  int
	cacheFile string
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
		root = filepath.Join(home, ".data")
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
		root = filepath.Join(home, ".data")
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
	noteCache = lru.New(4096)
	s.cacheFile = filepath.Join(root, "lru_cache.json") // кеш-файл
	s.loadCacheFromDisk()
	if err := fts.Init(root); err != nil {
		// not fatal: we still want store to work; just log warning to stderr
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("warning.fulltext_init_failed", err))
	}

	if err := s.openActiveSegmentRW(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	var err1 error
	s.saveCacheToDisk()
	if s.active != nil {
		err1 = s.active.Close()
	}
	// close fulltext index (best-effort)
	if err := fts.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("warning.fulltext_close_error", err))
	}
	return err1
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
	if err := enc.Encode(n); err != nil {
		return err
	}
	if noteCache != nil {
		if n.Deleted {
			noteCache.Remove(n.ID)
		} else {
			noteCache.Add(n.ID, n)
		}
	}
	// update fulltext index (best-effort: логируем и не откатываем Append)
	if err := fts.IndexNote(n); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("warning.fulltext_index_update_failed", n.ID, err))
	}
	return nil
}

// GetByID делает линейный поиск по всем сегментам, начиная с последнего.
func (s *Store) GetByID(id string) (*model.Note, error) {
	segDir := filepath.Join(s.root, dirSegments)
	if noteCache != nil {
		if v, ok := noteCache.Get(id); ok {
			if n, ok2 := v.(*model.Note); ok2 {
				if !n.Deleted {
					// log.Printf("lru cache got this")
					return n, nil
				}
				return nil, ErrNotFound
			}
		}
	}

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
					// cache found note
					// log.Printf("lru cache analyze this")
					if noteCache != nil {
						// log.Printf("lru cache added this")
						// store pointer to a heap-allocated copy (n is local; &n is ok as it's escaped)
						noteCache.Add(n.ID, &n)
					}
					f.Close()
					return &n, nil
				}

			}
		}
		f.Close()
	}
	return nil, ErrNotFound
}

func (s *Store) List(filter Filter) ([]model.Note, error) {
	segDir := filepath.Join(s.root, dirSegments)
	files, _ := filepath.Glob(filepath.Join(segDir, "notes-*.ndjson"))
	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	// If Contains is set, try fulltext search first (fast).
	if strings.TrimSpace(filter.Contains) != "" {
		// try fulltext
		ids, err := fts.Search(filter.Contains, filter.Limit)
		// fmt.Fprintf(os.Stderr, "debug: fts.Search('%s') -> len(ids)=%d, err=%v\n", filter.Contains, len(ids), err)
		if err == nil && len(ids) > 0 {
			var out []model.Note
			seen := make(map[string]bool)
			for i, id := range ids {
				if filter.Limit > 0 && i >= filter.Limit {
					break
				}
				if seen[id] {
					continue
				}
				n, err := s.GetByID(id)
				if err != nil {
					continue
				}
				if filter.Tag != "" {
					found := slices.Contains(n.Tags, filter.Tag)
					if !found {
						continue
					}
				}
				out = append(out, *n)
				seen[id] = true
			}
			// sort by created desc
			sort.SliceStable(out, func(i, j int) bool {
				return out[i].CreatedAt.After(out[j].CreatedAt)
			})
			return out, nil
		}
		// else: fallthrough to full scan (index not available or returned nothing)
	}

	// --- existing scan fallback (unchanged) ---
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

func (s *Store) loadCacheFromDisk() {
	f, err := os.Open(s.cacheFile)
	if err != nil {
		return // если файла нет — ничего страшного
	}
	defer f.Close()

	var notes []model.Note
	dec := json.NewDecoder(f)
	if err := dec.Decode(&notes); err != nil {
		return
	}

	for i := range notes {
		n := notes[i]
		if !n.Deleted {
			noteCache.Add(n.ID, &n)
		}
	}
}

func (s *Store) saveCacheToDisk() {
	if noteCache == nil {
		return
	}

	var notes []model.Note
	// сканируем все значения LRU
	for _, e := range noteCache.All() {
		if n, ok := e.(*model.Note); ok && !n.Deleted {
			notes = append(notes, *n)
		}
	}

	b, _ := json.MarshalIndent(notes, "", "  ")
	_ = os.WriteFile(s.cacheFile, b, 0o644)
}
