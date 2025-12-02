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

// InitFromEnv ищет JSON с локалью.
// Использует NOTELINE_I18N_DIR если задан, иначе последовательность:
// ./internal/i18n/locals, ./internal/i18n, ./i18n/locals, ./i18n, $HOME/NoteLine/i18n/locals, $HOME/NoteLine/i18n
// Логика локали: NOTELINE_LANG -> LANG -> "en". Если LANG == "C" || "POSIX" -> treat as "en".
func InitFromEnv() error {
	dirEnv := os.Getenv("NOTELINE_I18N_DIR")

	// determine lang, normalize
	lang := os.Getenv("NOTELINE_LANG")
	if lang == "" {
		lang = os.Getenv("LANG")
	}
	if lang == "" {
		lang = "en"
	}
	// normalize locales like "en_US.UTF-8" -> "en_US"
	lang = strings.SplitN(lang, ".", 2)[0]
	lang = strings.SplitN(lang, "@", 2)[0]
	lang = strings.TrimSpace(lang)

	// treat "C" and "POSIX" as default "en"
	if lang == "C" || lang == "POSIX" || lang == "" {
		lang = "en"
	}

	// build candidate directories (prefer project-relative 'internal/i18n/locals')
	candidates := make([]string, 0, 8)
	if dirEnv != "" {
		candidates = append(candidates, dirEnv, filepath.Join(dirEnv, "locals"))
	} else {
		wd, _ := os.Getwd()
		home, _ := os.UserHomeDir()

		// preferred: relative to project root (where you run the binary)
		if wd != "" {
			candidates = append(candidates,
				filepath.Join(wd, "internal", "i18n", "locals"),
				filepath.Join(wd, "internal", "i18n"),
				filepath.Join(wd, "i18n", "locals"),
				filepath.Join(wd, "i18n"),
			)
		}
		// fallback: home/NoteLine
		if home != "" {
			candidates = append(candidates,
				filepath.Join(home, "NoteLine", "i18n", "locals"),
				filepath.Join(home, "NoteLine", "i18n"),
			)
		}
	}

	// dedupe
	candidates = uniqueStrings(candidates)

	// try exact lang and short lang per each candidate dir
	var errs []error
	for _, d := range candidates {
		if d == "" {
			continue
		}
		// try exact (e.g. en_US)
		if err := tryLoad(filepath.Join(d, lang+".json"), lang); err == nil {
			return nil
		} else {
			errs = append(errs, err)
		}
		// try short code (e.g. "en")
		if len(lang) > 2 {
			short := lang[:2]
			if err := tryLoad(filepath.Join(d, short+".json"), short); err == nil {
				return nil
			} else {
				errs = append(errs, err)
			}
		}
	}

	// as last resort, try "en" files in candidates
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

	// nothing loaded
	lastErr = fmt.Errorf("i18n: failed to load locale %q; tried %d paths; lastErr: %v", lang, len(lastTried), errsToString(errs))
	// print short diagnostic so user understands and can fix path
	fmt.Fprintf(os.Stderr, "i18n: load failed: %v\n", lastErr)
	fmt.Fprintf(os.Stderr, "i18n: set NOTELINE_I18N_DIR or put locale json into one of:\n")
	for _, p := range candidates {
		fmt.Fprintf(os.Stderr, "  - %s\n", p)
	}
	return lastErr
}

// tryLoad attempts to load file fullPath which should be <dir>/<loc>.json
func tryLoad(fullPath, loc string) error {
	mu.Lock()
	lastTried = append(lastTried, fullPath)
	mu.Unlock()
	// call LoadLocale which reads file and sets msgs
	if err := LoadLocale(fullPath, loc); err != nil {
		return err
	}
	mu.Lock()
	loadedFrom = fullPath
	lastErr = nil
	mu.Unlock()
	return nil
}

// LoadLocale loads file at fullPath and sets locale to loc.
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

// T возвращает локализованную строку, форматируя через fmt.Sprintf.
// Если msgs не загружены — возвращает ключ (и не пытается Sprintf ключа).
func T(key string, args ...interface{}) string {
	mu.RLock()
	defer mu.RUnlock()
	if msgs == nil {
		// msgs не загружены: возвращаем ключ или ключ + args для диагностики
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

// Locale возвращает текущую локаль (для отладки)
func Locale() string {
	mu.RLock()
	defer mu.RUnlock()
	return locale
}

// LastLoadInfo возвращает from, tried paths and lastErr (для отладки)
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
