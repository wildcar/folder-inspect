// Package i18n holds the user-facing strings in Russian (default) and
// English. Keep every key in both maps; TestAllKeysTranslated enforces it.
package i18n

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// Supported languages.
const (
	RU = "ru"
	EN = "en"
)

var (
	mu   sync.RWMutex
	lang = RU
)

// Set selects the language; unknown values fall back to Russian.
func Set(l string) {
	l = strings.ToLower(strings.TrimSpace(l))
	if l != EN {
		l = RU
	}
	mu.Lock()
	lang = l
	mu.Unlock()
}

// Lang returns the current language.
func Lang() string {
	mu.RLock()
	defer mu.RUnlock()
	return lang
}

// Detect guesses the language from LANG / LC_ALL / LANGUAGE. Anything that
// does not start with "en" (including unset, the usual case on Windows)
// means Russian. Proper OS-locale detection is a later step.
func Detect() string {
	for _, k := range []string{"LC_ALL", "LANGUAGE", "LANG"} {
		if v := strings.ToLower(os.Getenv(k)); strings.HasPrefix(v, "en") {
			return EN
		}
	}
	return RU
}

// T returns the translation of key, or the key itself when missing.
func T(key string) string {
	mu.RLock()
	l := lang
	mu.RUnlock()
	if s, ok := messages[l][key]; ok {
		return s
	}
	if s, ok := messages[RU][key]; ok {
		return s
	}
	return key
}

// Tf is T followed by Sprintf.
func Tf(key string, args ...any) string { return fmt.Sprintf(T(key), args...) }

var messages = map[string]map[string]string{
	RU: {
		"scan.header":      "%s %s — сканирование %s",
		"scan.roots":       "Корни: %d, файлов: %d, папок: %d, объём: %s, время: %s",
		"scan.summary":     "Сводка по категориям",
		"scan.category":    "Категория",
		"scan.count":       "Кол-во",
		"scan.size":        "Объём",
		"scan.nothing":     "Проблем не найдено.",
		"scan.top_files":   "Самые большие файлы",
		"scan.top_dirs":    "Самые тяжёлые папки",
		"scan.errors":      "Ошибки чтения: %d",
		"scan.skipped":     "Пропущено (ссылки, системные папки): %d",
		"scan.more":        "… и ещё %d",
		"scan.written":     "Отчёт сохранён: %s",
		"scan.threshold":   "порог",
		"cat.oversize":     "Большие файлы",
		"cat.archive":      "Архивы",
		"cat.distributive": "Дистрибутивы",
		"cat.junk":         "Мусор",
		"cat.empty-dir":    "Пустые папки",
		"cat.empty-file":   "Пустые файлы",
		"detail.empty":     "пустая",
		"detail.no-files":  "только пустые подпапки",
		"unit.B":           "Б",
		"unit.KB":          "КБ",
		"unit.MB":          "МБ",
		"unit.GB":          "ГБ",
		"unit.TB":          "ТБ",
	},
	EN: {
		"scan.header":      "%s %s — scan of %s",
		"scan.roots":       "Roots: %d, files: %d, folders: %d, size: %s, time: %s",
		"scan.summary":     "Summary by category",
		"scan.category":    "Category",
		"scan.count":       "Count",
		"scan.size":        "Size",
		"scan.nothing":     "No findings.",
		"scan.top_files":   "Largest files",
		"scan.top_dirs":    "Heaviest folders",
		"scan.errors":      "Read errors: %d",
		"scan.skipped":     "Skipped (links, system folders): %d",
		"scan.more":        "… and %d more",
		"scan.written":     "Report written: %s",
		"scan.threshold":   "threshold",
		"cat.oversize":     "Oversized files",
		"cat.archive":      "Archives",
		"cat.distributive": "Distributives",
		"cat.junk":         "Junk",
		"cat.empty-dir":    "Empty folders",
		"cat.empty-file":   "Empty files",
		"detail.empty":     "empty",
		"detail.no-files":  "only empty sub-folders",
		"unit.B":           "B",
		"unit.KB":          "KB",
		"unit.MB":          "MB",
		"unit.GB":          "GB",
		"unit.TB":          "TB",
	},
}

// Keys returns all message keys of the default language (for tests).
func Keys() []string {
	out := make([]string, 0, len(messages[RU]))
	for k := range messages[RU] {
		out = append(out, k)
	}
	return out
}

// Has reports whether the language defines the key (for tests).
func Has(l, key string) bool {
	_, ok := messages[l][key]
	return ok
}
