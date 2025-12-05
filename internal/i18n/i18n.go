package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	mu         sync.RWMutex
	msgs       map[string]string
	locale     string
	lastTried  []string
	lastErr    error
	loadedFrom string
)

func InitFromEnv() error {
	dirEnv := os.Getenv("NOTELINE_I18N_DIR")

	lang := os.Getenv("NOTELINE_LANG")
	if lang == "" {
		lang = os.Getenv("LANG")
	}
	if lang == "" {
		lang = "en"
	}

	lang = strings.SplitN(lang, ".", 2)[0]
	lang = strings.SplitN(lang, "@", 2)[0]
	lang = strings.TrimSpace(lang)

	if lang == "C" || lang == "POSIX" || lang == "" {
		lang = "en"
	}

	candidates := make([]string, 0, 8)
	if dirEnv != "" {
		candidates = append(candidates, dirEnv, filepath.Join(dirEnv, "locals"))
	} else {
		wd, _ := os.Getwd()
		home, _ := os.UserHomeDir()

		if wd != "" {
			candidates = append(candidates,
				filepath.Join(wd, "internal", "i18n", "locals"),
				filepath.Join(wd, "internal", "i18n"),
				filepath.Join(wd, "i18n", "locals"),
				filepath.Join(wd, "i18n"),
			)
		}

		if home != "" {
			candidates = append(candidates,
				filepath.Join(home, "NoteLine", "i18n", "locals"),
				filepath.Join(home, "NoteLine", "i18n"),
			)
		}
	}

	candidates = uniqueStrings(candidates)

	var errs []error
	for _, d := range candidates {
		if d == "" {
			continue
		}

		if err := tryLoad(filepath.Join(d, lang+".json"), lang); err == nil {
			return nil
		} else {
			errs = append(errs, err)
		}

		if len(lang) > 2 {
			short := lang[:2]
			if err := tryLoad(filepath.Join(d, short+".json"), short); err == nil {
				return nil
			} else {
				errs = append(errs, err)
			}
		}
	}

	for _, d := range candidates {
		if d == "" {
			continue
		}
		if err := tryLoad(filepath.Join(d, "en.json"), "en"); err == nil {
			return nil
		} else {
			errs = append(errs, err)
		}
	}

	lastErr = fmt.Errorf("i18n: failed to load locale %q; tried %d paths; lastErr: %v", lang, len(lastTried), errsToString(errs))

	fmt.Fprintf(os.Stderr, "i18n: load failed: %v\n", lastErr)
	fmt.Fprintf(os.Stderr, "i18n: set NOTELINE_I18N_DIR or put locale json into one of:\n")
	for _, p := range candidates {
		fmt.Fprintf(os.Stderr, "  - %s\n", p)
	}
	return lastErr
}

func tryLoad(fullPath, loc string) error {
	mu.Lock()
	lastTried = append(lastTried, fullPath)
	mu.Unlock()

	if err := LoadLocale(fullPath, loc); err != nil {
		return err
	}
	mu.Lock()
	loadedFrom = fullPath
	lastErr = nil
	mu.Unlock()
	return nil
}

func LoadLocale(fullPath, loc string) error {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	mu.Lock()
	msgs = m
	locale = loc
	lastErr = nil
	loadedFrom = fullPath
	mu.Unlock()
	return nil
}

func T(key string, args ...interface{}) string {
	mu.RLock()
	defer mu.RUnlock()
	if msgs == nil {

		if len(args) == 0 {
			return key
		}
		return fmt.Sprintf("%s %v", key, args)
	}
	val, ok := msgs[key]
	if !ok || val == "" {
		if len(args) == 0 {
			return key
		}
		return fmt.Sprintf("%s %v", key, args)
	}
	if len(args) == 0 {
		return val
	}
	return fmt.Sprintf(val, args...)
}

func Locale() string {
	mu.RLock()
	defer mu.RUnlock()
	return locale
}

func LastLoadInfo() (loadedFromPath string, triedPaths []string, err error) {
	mu.RLock()
	defer mu.RUnlock()
	cpy := make([]string, len(lastTried))
	copy(cpy, lastTried)
	return loadedFrom, cpy, lastErr
}

func uniqueStrings(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func errsToString(errs []error) string {
	if len(errs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(errs))
	for _, e := range errs {
		parts = append(parts, e.Error())
	}
	return strings.Join(parts, " ; ")
}
