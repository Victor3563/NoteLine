package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

const (
	dirSegments      = "segments"
	filenameManifest = "manifest.json"

	defaultSegSize int = 8 * 1024 * 1024
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

func Ensure(root string) error {
	if strings.TrimSpace(root) == "" {
		home, _ := os.UserHomeDir()

		root = filepath.Join(home, ".noteline")
	}
	if err := os.MkdirAll(filepath.Join(root, dirSegments), 0o755); err != nil {
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

	noteCache = lru.New(4096)
	s.cacheFile = filepath.Join(root, "lru_cache.json")
	s.loadCacheFromDisk()

	if err := fts.Init(root); err != nil {
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

	if err := fts.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("warning.fulltext_close_error", err))
	}
	return err1
}

func (s *Store) openActiveSegmentRW() error {
	segDir := filepath.Join(s.root, dirSegments)
	files, _ := filepath.Glob(filepath.Join(segDir, "notes-*.ndjson"))
	if len(files) == 0 {

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

	b, err := json.MarshalIndent(s.man, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.root, filenameManifest), b, 0o644)
}

func (s *Store) Append(n *model.Note) error {
	if s.active == nil {
		if err := s.openActiveSegmentRW(); err != nil {
			return err
		}
	}

	b, err := json.Marshal(n)
	if err != nil {
		return err
	}

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

	if noteCache != nil {
		if n.Deleted {
			noteCache.Remove(n.ID)
		} else {
			noteCopy := *n
			noteCache.Add(n.ID, &noteCopy)
		}
	}

	if err := fts.IndexNote(n); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("warning.fulltext_index_update_failed", n.ID, err))
	}

	return nil
}

func (s *Store) loadAllNotes() (map[string]model.Note, error) {
	segDir := filepath.Join(s.root, dirSegments)
	files, _ := filepath.Glob(filepath.Join(segDir, "notes-*.ndjson"))
	sort.Strings(files)

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

				continue
			}

			if n.Deleted {

				delete(notes, n.ID)
				continue
			}

			notes[n.ID] = n
		}

		_ = f.Close()
	}

	return notes, nil
}

func (s *Store) GetByID(id string) (*model.Note, error) {
	if noteCache != nil {
		if v, ok := noteCache.Get(id); ok {
			if n, ok2 := v.(*model.Note); ok2 {
				if !n.Deleted {
					nCopy := *n
					return &nCopy, nil
				}
				return nil, ErrNotFound
			}
		}
	}

	notes, err := s.loadAllNotes()
	if err != nil {
		return nil, err
	}
	n, ok := notes[id]
	if !ok {
		return nil, ErrNotFound
	}

	if noteCache != nil {
		nCopy := n
		noteCache.Add(id, &nCopy)
	}

	nCopy := n
	return &nCopy, nil
}

func (s *Store) List(filter Filter) ([]model.Note, error) {
	filter.Tag = strings.TrimSpace(filter.Tag)
	filter.Contains = strings.TrimSpace(filter.Contains)

	if filter.Contains != "" {
		ids, err := fts.Search(filter.Contains, filter.Limit)
		if err == nil && len(ids) > 0 {
			var out []model.Note
			seen := make(map[string]bool)

			for _, id := range ids {
				if filter.Limit > 0 && len(out) >= filter.Limit {
					break
				}
				if seen[id] {
					continue
				}
				n, err := s.GetByID(id)
				if err != nil {
					continue
				}
				if filter.Tag != "" && !slices.Contains(n.Tags, filter.Tag) {
					continue
				}
				out = append(out, *n)
				seen[id] = true
			}

			sort.SliceStable(out, func(i, j int) bool {
				return out[i].CreatedAt.After(out[j].CreatedAt)
			})
			return out, nil
		}

	}

	notes, err := s.loadAllNotes()
	if err != nil {
		return nil, err
	}

	var out []model.Note
	for _, n := range notes {

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
			hay := strings.ToLower(n.Title + " " + n.Text)
			needle := strings.ToLower(filter.Contains)
			if !strings.Contains(hay, needle) {
				continue
			}
		}

		out = append(out, n)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})

	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}

	return out, nil
}

func (s *Store) loadCacheFromDisk() {
	if noteCache == nil {
		return
	}

	f, err := os.Open(s.cacheFile)
	if err != nil {
		return
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
			noteCopy := n
			noteCache.Add(n.ID, &noteCopy)
		}
	}
}

func (s *Store) saveCacheToDisk() {
	if noteCache == nil {
		return
	}

	var notes []model.Note
	for _, e := range noteCache.All() {
		if n, ok := e.(*model.Note); ok && !n.Deleted {
			notes = append(notes, *n)
		}
	}

	b, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(s.cacheFile, b, 0o644)
}
