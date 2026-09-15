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
		"scan.exported":    "Выгрузка сохранена: %s",
		"scan.hashed":      "Проверка дубликатов: прочитано %d файлов, %s",
		"scan.threshold":   "порог",
		"cat.oversize":     "Большие файлы",
		"cat.archive":      "Архивы",
		"cat.distributive": "Дистрибутивы",
		"cat.duplicate":    "Дубликаты",
		"cat.junk":         "Мусор",
		"cat.empty-dir":    "Пустые папки",
		"cat.empty-file":   "Пустые файлы",
		"detail.empty":     "пустая",
		"detail.no-files":  "только пустые подпапки",
		"dup.header":       "%s: групп %d, впустую %s",
		"dup.legend":       "* — предлагаемый оригинал (самый старый по дате изменения); выбор за вами",
		"dup.copies":       "копий: %d",
		"err.exists":       "%s: файл уже существует, добавьте -force для перезаписи",
		"col.category":     "Категория",
		"col.rule":         "Правило",
		"col.group":        "Группа",
		"col.path":         "Путь",
		"col.root":         "Корень",
		"col.size":         "Размер",
		"col.size_bytes":   "Размер, байт",
		"col.mtime":        "Изменён",
		"col.threshold":    "Порог",
		"col.detail":       "Примечание",
		"col.count":        "Кол-во",
		"col.wasted":       "Впустую",
		"col.files":        "Файлов",
		"col.suggested":    "Оригинал",
		"col.metric":       "Показатель",
		"col.value":        "Значение",
		"col.error":        "Ошибка",
		"sheet.summary":    "Сводка",
		"sheet.findings":   "Находки",
		"sheet.duplicates": "Дубликаты",
		"sheet.top_files":  "Самые большие файлы",
		"sheet.top_dirs":   "Самые тяжёлые папки",
		"sheet.errors":     "Ошибки",
		"sum.tool":         "Инструмент",
		"sum.started":      "Начало сканирования",
		"sum.duration":     "Длительность",
		"sum.roots":        "Папки",
		"sum.files":        "Файлов",
		"sum.dirs":         "Папок",
		"sum.total":        "Общий объём",
		"sum.errors":       "Ошибок чтения",
		"sum.skipped":      "Пропущено",
		"sum.hashed":       "Прочитано для поиска дубликатов",
		"html.title":       "Отчёт folder-inspect",
		"html.generated":   "сформирован %s",
		"html.errors":      "Ошибки чтения",
		"html.skipped":     "Пропущено (ссылки, системные папки)",
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
		"scan.exported":    "Export written: %s",
		"scan.hashed":      "Duplicate check: read %d files, %s",
		"scan.threshold":   "threshold",
		"cat.oversize":     "Oversized files",
		"cat.archive":      "Archives",
		"cat.distributive": "Distributives",
		"cat.duplicate":    "Duplicates",
		"cat.junk":         "Junk",
		"cat.empty-dir":    "Empty folders",
		"cat.empty-file":   "Empty files",
		"detail.empty":     "empty",
		"detail.no-files":  "only empty sub-folders",
		"dup.header":       "%s: %d groups, %s wasted",
		"dup.legend":       "* = suggested original (oldest modification time); the choice is yours",
		"dup.copies":       "copies: %d",
		"err.exists":       "%s: file exists, add -force to overwrite",
		"col.category":     "Category",
		"col.rule":         "Rule",
		"col.group":        "Group",
		"col.path":         "Path",
		"col.root":         "Root",
		"col.size":         "Size",
		"col.size_bytes":   "Size, bytes",
		"col.mtime":        "Modified",
		"col.threshold":    "Threshold",
		"col.detail":       "Note",
		"col.count":        "Count",
		"col.wasted":       "Wasted",
		"col.files":        "Files",
		"col.suggested":    "Original",
		"col.metric":       "Metric",
		"col.value":        "Value",
		"col.error":        "Error",
		"sheet.summary":    "Summary",
		"sheet.findings":   "Findings",
		"sheet.duplicates": "Duplicates",
		"sheet.top_files":  "Largest files",
		"sheet.top_dirs":   "Heaviest folders",
		"sheet.errors":     "Errors",
		"sum.tool":         "Tool",
		"sum.started":      "Scan started",
		"sum.duration":     "Duration",
		"sum.roots":        "Folders",
		"sum.files":        "Files",
		"sum.dirs":         "Folders",
		"sum.total":        "Total size",
		"sum.errors":       "Read errors",
		"sum.skipped":      "Skipped",
		"sum.hashed":       "Read for duplicate check",
		"html.title":       "folder-inspect report",
		"html.generated":   "generated %s",
		"html.errors":      "Read errors",
		"html.skipped":     "Skipped (links, system folders)",
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
