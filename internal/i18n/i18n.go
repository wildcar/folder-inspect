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
		"scan.header":          "%s %s — сканирование %s",
		"scan.roots":           "Корни: %d, файлов: %d, папок: %d, объём: %s, время: %s",
		"scan.summary":         "Сводка по категориям",
		"scan.category":        "Категория",
		"scan.count":           "Кол-во",
		"scan.size":            "Объём",
		"scan.nothing":         "Проблем не найдено.",
		"scan.top_files":       "Самые большие файлы",
		"scan.top_dirs":        "Самые тяжёлые папки",
		"scan.errors":          "Ошибки чтения: %d",
		"scan.skipped":         "Пропущено (ссылки, системные папки): %d",
		"scan.more":            "… и ещё %d",
		"scan.written":         "Отчёт сохранён: %s",
		"scan.exported":        "Выгрузка сохранена: %s",
		"scan.hashed":          "Проверка дубликатов: прочитано %d файлов, %s",
		"scan.threshold":       "порог",
		"cat.oversize":         "Большие файлы",
		"cat.archive":          "Архивы",
		"cat.distributive":     "Дистрибутивы",
		"cat.duplicate":        "Дубликаты",
		"cat.dir-duplicate":    "Папки-дубликаты",
		"cat.dir-overlap":      "Папки с общим содержимым",
		"dirdup.folders":       "папок: %d",
		"dirdup.files":         "файлов: %d",
		"overlap.header":       "%s: пар %d, общего содержимого %s",
		"overlap.shared":       "общих файлов: %d, %s, %d%% от меньшей папки",
		"col.related":          "Вторая папка",
		"col.ratio":            "Доля меньшей папки",
		"col.ratio_a":          "Доля первой",
		"col.ratio_b":          "Доля второй",
		"col.shared_files":     "Общих файлов",
		"col.shared_bytes":     "Общий объём, байт",
		"sheet.dir_duplicates": "Папки-дубликаты",
		"sheet.overlaps":       "Общее содержимое",
		"cat.junk":             "Мусор",
		"cat.empty-dir":        "Пустые папки",
		"cat.empty-file":       "Пустые файлы",
		"detail.empty":         "пустая",
		"detail.no-files":      "только пустые подпапки",
		"dup.header":           "%s: групп %d, впустую %s",
		"dup.legend":           "* — предлагаемый оригинал (самый старый по дате изменения); выбор за вами",
		"dup.copies":           "копий: %d",
		"err.exists":           "%s: файл уже существует, добавьте -force для перезаписи",
		"col.category":         "Категория",
		"col.rule":             "Правило",
		"col.group":            "Группа",
		"col.path":             "Путь",
		"col.root":             "Корень",
		"col.size":             "Размер",
		"col.size_bytes":       "Размер, байт",
		"col.mtime":            "Изменён",
		"col.threshold":        "Порог",
		"col.detail":           "Примечание",
		"col.count":            "Кол-во",
		"col.wasted":           "Впустую",
		"col.files":            "Файлов",
		"col.suggested":        "Оригинал",
		"col.metric":           "Показатель",
		"col.value":            "Значение",
		"col.error":            "Ошибка",
		"sheet.summary":        "Сводка",
		"sheet.findings":       "Находки",
		"sheet.duplicates":     "Дубликаты",
		"sheet.top_files":      "Самые большие файлы",
		"sheet.top_dirs":       "Самые тяжёлые папки",
		"sheet.errors":         "Ошибки",
		"sum.tool":             "Инструмент",
		"sum.started":          "Начало сканирования",
		"sum.duration":         "Длительность",
		"sum.roots":            "Папки",
		"sum.files":            "Файлов",
		"sum.dirs":             "Папок",
		"sum.total":            "Общий объём",
		"sum.errors":           "Ошибок чтения",
		"sum.skipped":          "Пропущено",
		"sum.hashed":           "Прочитано для поиска дубликатов",
		"html.title":           "Отчёт folder-inspect",
		"html.generated":       "сформирован %s",
		"html.errors":          "Ошибки чтения",
		"html.skipped":         "Пропущено (ссылки, системные папки)",
		"err.no_reports":       "%s: отчётов нет; сначала выполните folder-inspect scan",
		"scan.fallback_dir":    "не удалось создать %s (%v); отчёт будет записан в текущую папку",
		"scan.ui_hint":         "Открыть в браузере: folder-inspect ui \"%s\"",
		"ui.listening":         "Интерфейс открыт: %s  (Ctrl+C — выход)",
		"ui.open_failed":       "не удалось открыть браузер: %v",
		"ui.lang_switch":       "English",
		"ui.summary":           "Сводка",
		"ui.scanned":           "сканирование %s · %s",
		"ui.export":            "Выгрузка",
		"ui.filter":            "Фильтр по пути…",
		"ui.select_visible":    "Отметить видимые",
		"ui.clear_visible":     "Снять отметки",
		"ui.select_copies":     "Отметить копии",
		"ui.select_all_copies": "Отметить все копии во всех группах",
		"ui.original":          "оригинал",
		"ui.to_quarantine":     "в карантин",
		"ui.reveal":            "Показать в проводнике",
		"ui.no_items":          "Ничего не найдено",
		"ui.plan":              "План действий",
		"ui.plan_items":        "объектов: %d, %s",
		"ui.plan_empty":        "План пуст: отметьте файлы или копии, которые нужно убрать в карантин",
		"ui.plan_save":         "Сохранить план",
		"ui.plan_clear":        "Очистить",
		"ui.plan_saved":        "План сохранён: %s",
		"ui.plan_error":        "План не сохранён: %s",
		"ui.plan_hint":         "Интерфейс ничего не меняет на диске. Отметки собираются в план; его выполняет команда apply, после чего файлы уезжают в карантин с возможностью вернуть.",
		"ui.group_hint":        "Жёлтым выделен предлагаемый оригинал (самый старый). Выберите оригинал сами и отметьте копии, которые убрать в карантин: на их месте останется файл-указатель на оригинал.",
		"ui.overlap_hint":      "Пары папок, у которых общее содержимое составляет заметную долю меньшей папки. Это подсказка для ручного разбора; действий по парам нет.",
		"unit.B":               "Б",
		"unit.KB":              "КБ",
		"unit.MB":              "МБ",
		"unit.GB":              "ГБ",
		"unit.TB":              "ТБ",
	},
	EN: {
		"scan.header":          "%s %s — scan of %s",
		"scan.roots":           "Roots: %d, files: %d, folders: %d, size: %s, time: %s",
		"scan.summary":         "Summary by category",
		"scan.category":        "Category",
		"scan.count":           "Count",
		"scan.size":            "Size",
		"scan.nothing":         "No findings.",
		"scan.top_files":       "Largest files",
		"scan.top_dirs":        "Heaviest folders",
		"scan.errors":          "Read errors: %d",
		"scan.skipped":         "Skipped (links, system folders): %d",
		"scan.more":            "… and %d more",
		"scan.written":         "Report written: %s",
		"scan.exported":        "Export written: %s",
		"scan.hashed":          "Duplicate check: read %d files, %s",
		"scan.threshold":       "threshold",
		"cat.oversize":         "Oversized files",
		"cat.archive":          "Archives",
		"cat.distributive":     "Distributives",
		"cat.duplicate":        "Duplicates",
		"cat.dir-duplicate":    "Duplicate folders",
		"cat.dir-overlap":      "Folders with shared content",
		"dirdup.folders":       "folders: %d",
		"dirdup.files":         "files: %d",
		"overlap.header":       "%s: %d pairs, %s shared",
		"overlap.shared":       "shared files: %d, %s, %d%% of the smaller folder",
		"col.related":          "Second folder",
		"col.ratio":            "Share of smaller",
		"col.ratio_a":          "Share of first",
		"col.ratio_b":          "Share of second",
		"col.shared_files":     "Shared files",
		"col.shared_bytes":     "Shared size, bytes",
		"sheet.dir_duplicates": "Duplicate folders",
		"sheet.overlaps":       "Shared content",
		"cat.junk":             "Junk",
		"cat.empty-dir":        "Empty folders",
		"cat.empty-file":       "Empty files",
		"detail.empty":         "empty",
		"detail.no-files":      "only empty sub-folders",
		"dup.header":           "%s: %d groups, %s wasted",
		"dup.legend":           "* = suggested original (oldest modification time); the choice is yours",
		"dup.copies":           "copies: %d",
		"err.exists":           "%s: file exists, add -force to overwrite",
		"col.category":         "Category",
		"col.rule":             "Rule",
		"col.group":            "Group",
		"col.path":             "Path",
		"col.root":             "Root",
		"col.size":             "Size",
		"col.size_bytes":       "Size, bytes",
		"col.mtime":            "Modified",
		"col.threshold":        "Threshold",
		"col.detail":           "Note",
		"col.count":            "Count",
		"col.wasted":           "Wasted",
		"col.files":            "Files",
		"col.suggested":        "Original",
		"col.metric":           "Metric",
		"col.value":            "Value",
		"col.error":            "Error",
		"sheet.summary":        "Summary",
		"sheet.findings":       "Findings",
		"sheet.duplicates":     "Duplicates",
		"sheet.top_files":      "Largest files",
		"sheet.top_dirs":       "Heaviest folders",
		"sheet.errors":         "Errors",
		"sum.tool":             "Tool",
		"sum.started":          "Scan started",
		"sum.duration":         "Duration",
		"sum.roots":            "Folders",
		"sum.files":            "Files",
		"sum.dirs":             "Folders",
		"sum.total":            "Total size",
		"sum.errors":           "Read errors",
		"sum.skipped":          "Skipped",
		"sum.hashed":           "Read for duplicate check",
		"html.title":           "folder-inspect report",
		"html.generated":       "generated %s",
		"html.errors":          "Read errors",
		"html.skipped":         "Skipped (links, system folders)",
		"err.no_reports":       "%s: no reports yet; run folder-inspect scan first",
		"scan.fallback_dir":    "cannot create %s (%v); the report goes to the current folder",
		"scan.ui_hint":         "Open in the browser: folder-inspect ui \"%s\"",
		"ui.listening":         "UI is at %s  (Ctrl+C to quit)",
		"ui.open_failed":       "could not open the browser: %v",
		"ui.lang_switch":       "Русский",
		"ui.summary":           "Summary",
		"ui.scanned":           "scanned %s · %s",
		"ui.export":            "Export",
		"ui.filter":            "Filter by path…",
		"ui.select_visible":    "Select visible",
		"ui.clear_visible":     "Clear selection",
		"ui.select_copies":     "Select copies",
		"ui.select_all_copies": "Select all copies in all groups",
		"ui.original":          "original",
		"ui.to_quarantine":     "to quarantine",
		"ui.reveal":            "Show in file manager",
		"ui.no_items":          "Nothing found",
		"ui.plan":              "Action plan",
		"ui.plan_items":        "items: %d, %s",
		"ui.plan_empty":        "The plan is empty: tick files or copies to move to quarantine",
		"ui.plan_save":         "Save plan",
		"ui.plan_clear":        "Clear",
		"ui.plan_saved":        "Plan saved: %s",
		"ui.plan_error":        "Plan not saved: %s",
		"ui.plan_hint":         "The UI changes nothing on disk. Ticks are collected into a plan; the apply command executes it, moving files to a restorable quarantine.",
		"ui.group_hint":        "Yellow marks the suggested original (oldest). Pick the original yourself and tick the copies to quarantine: a pointer file naming the original stays in their place.",
		"ui.overlap_hint":      "Folder pairs whose shared content is a notable share of the smaller folder. A hint for manual review; there are no actions on pairs.",
		"unit.B":               "B",
		"unit.KB":              "KB",
		"unit.MB":              "MB",
		"unit.GB":              "GB",
		"unit.TB":              "TB",
	},
}

// All returns a copy of the whole catalog of a language (the web UI takes
// it in one request). Unknown languages give Russian.
func All(l string) map[string]string {
	src, ok := messages[l]
	if !ok {
		src = messages[RU]
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
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
