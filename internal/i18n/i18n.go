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

// TL translates key in an explicit language (stubs written on disk must not
// depend on the process-wide setting). Unknown languages fall back to T.
func TL(l, key string) string {
	if s, ok := messages[l][key]; ok {
		return s
	}
	return T(key)
}

// TLf is TL followed by Sprintf.
func TLf(l, key string, args ...any) string { return fmt.Sprintf(TL(l, key), args...) }

var messages = map[string]map[string]string{
	RU: {
		"scan.header":              "%s %s — сканирование %s",
		"scan.roots":               "Корни: %d, файлов: %d, папок: %d, объём: %s, время: %s",
		"scan.summary":             "Сводка по категориям",
		"scan.category":            "Категория",
		"scan.count":               "Кол-во",
		"scan.size":                "Объём",
		"scan.nothing":             "Проблем не найдено.",
		"scan.top_files":           "Самые большие файлы",
		"scan.top_dirs":            "Самые тяжёлые папки",
		"scan.errors":              "Ошибки чтения: %d",
		"scan.skipped":             "Пропущено (ссылки, системные папки): %d",
		"scan.more":                "… и ещё %d",
		"scan.written":             "Отчёт сохранён: %s",
		"scan.exported":            "Выгрузка сохранена: %s",
		"scan.hashed":              "Проверка дубликатов: прочитано %d файлов, %s",
		"scan.threshold":           "порог",
		"cat.oversize":             "Большие файлы",
		"cat.archive":              "Архивы",
		"cat.distributive":         "Дистрибутивы",
		"cat.duplicate":            "Дубликаты",
		"cat.dir-duplicate":        "Папки-дубликаты",
		"cat.dir-overlap":          "Папки с общим содержимым",
		"cat.similar-name":         "Похожие имена",
		"names.header":             "%s: групп %d, вариантов %d, %s",
		"names.legend":             "Файлы с одним именем без пометок вроде «Копия», «(2)», «- копия», «_v2», «_final». Исходный файл (без пометки) отмечен *. Содержимое может отличаться — это подсказка для разбора.",
		"names.base":               "исходный",
		"names.dup":                "дубликат %s",
		"col.name":                 "Имя",
		"col.marker":               "Пометка",
		"col.dup_group":            "Группа дубликатов",
		"sheet.similar_names":      "Похожие имена",
		"stub.similar-name":        "Причина: судя по имени (пометка «%s»), это копия или старая версия файла. Актуальный файл, вероятно, здесь:\n  %s",
		"stub.similar-name-nobase": "Причина: судя по имени (пометка «%s»), это копия или старая версия другого файла.",
		"ui.names_hint":            "Файлы с одним именем без пометок вроде «Копия», «(2)», «_v2». Жёлтым выделен исходный файл (без пометки). Содержимое может отличаться, поэтому автоматических предложений нет: откройте файлы и отметьте лишние.",
		"ui.quarantine":            "Карантин",
		"ui.quarantine_hint":       "Пакеты карантина этого репозитория. «Вернуть» возвращает файлы на место и убирает заглушки. Окончательное удаление доступно только из консоли: folder-inspect quarantine purge -yes <папка пакета>.",
		"ui.q_empty":               "Карантин пуст",
		"ui.q_created":             "Дата",
		"ui.q_items":               "Объектов",
		"ui.q_pending":             "В карантине",
		"ui.q_status":              "Состояние",
		"ui.q_active":              "в карантине",
		"ui.q_restored":            "возвращён",
		"ui.q_partial":             "возвращён частично",
		"ui.q_purged":              "удалён окончательно",
		"ui.q_deleted":             "удалено без карантина",
		"ui.q_restore":             "Вернуть",
		"ui.q_restore_confirm":     "Вернуть %d объектов (%s) из карантина %s на прежние места?",
		"ui.q_restore_done":        "Возвращено: %d, проблем: %d, удалено без карантина (не возвращается): %d",
		"q.list_header":            "Карантин %s: пакетов %d",
		"q.none":                   "Пакетов карантина нет.",
		"q.status.active":          "в карантине",
		"q.status.restored":        "возвращён",
		"q.status.partial":         "возвращён частично",
		"q.status.purged":          "удалён окончательно",
		"q.status.deleted":         "удалено без карантина",
		"q.purge_preview":          "Будет удалено окончательно: %d объектов, %s, из %s",
		"q.purge_hint":             "Это действие нельзя отменить. Для удаления повторите с -yes:\n  folder-inspect quarantine purge -yes \"%s\"",
		"q.purged":                 "Удалено окончательно: %d объектов, %s. Заглушки на местах файлов остались.",
		"q.already_purged":         "%s: пакет уже удалён окончательно",
		"q.nothing_to_purge":       "%s: в пакете нет объектов для удаления (всё возвращено)",
		"dirdup.folders":           "папок: %d",
		"dirdup.files":             "файлов: %d",
		"overlap.header":           "%s: пар %d, общего содержимого %s",
		"overlap.shared":           "общих файлов: %d, %s, %d%% от меньшей папки",
		"col.related":              "Вторая папка",
		"col.ratio":                "Доля меньшей папки",
		"col.ratio_a":              "Доля первой",
		"col.ratio_b":              "Доля второй",
		"col.shared_files":         "Общих файлов",
		"col.shared_bytes":         "Общий объём, байт",
		"sheet.dir_duplicates":     "Папки-дубликаты",
		"sheet.overlaps":           "Общее содержимое",
		"cat.junk":                 "Мусор",
		"cat.empty-dir":            "Пустые папки",
		"cat.empty-file":           "Пустые файлы",
		"detail.empty":             "пустая",
		"detail.no-files":          "только пустые подпапки",
		"dup.header":               "%s: групп %d, впустую %s",
		"dup.legend":               "* — предлагаемый оригинал (самый старый по дате изменения); выбор за вами",
		"dup.copies":               "копий: %d",
		"err.exists":               "%s: файл уже существует, добавьте -force для перезаписи",
		"col.category":             "Категория",
		"col.rule":                 "Правило",
		"col.group":                "Группа",
		"col.path":                 "Путь",
		"col.root":                 "Корень",
		"col.size":                 "Размер",
		"col.size_bytes":           "Размер, байт",
		"col.mtime":                "Изменён",
		"col.threshold":            "Порог",
		"col.detail":               "Примечание",
		"col.count":                "Кол-во",
		"col.wasted":               "Впустую",
		"col.files":                "Файлов",
		"col.suggested":            "Оригинал",
		"col.metric":               "Показатель",
		"col.value":                "Значение",
		"col.error":                "Ошибка",
		"sheet.summary":            "Сводка",
		"sheet.findings":           "Находки",
		"sheet.duplicates":         "Дубликаты",
		"sheet.top_files":          "Самые большие файлы",
		"sheet.top_dirs":           "Самые тяжёлые папки",
		"sheet.errors":             "Ошибки",
		"sum.tool":                 "Инструмент",
		"sum.started":              "Начало сканирования",
		"sum.duration":             "Длительность",
		"sum.roots":                "Папки",
		"sum.files":                "Файлов",
		"sum.dirs":                 "Папок",
		"sum.total":                "Общий объём",
		"sum.errors":               "Ошибок чтения",
		"sum.skipped":              "Пропущено",
		"sum.hashed":               "Прочитано для поиска дубликатов",
		"html.title":               "Отчёт folder-inspect",
		"html.generated":           "сформирован %s",
		"html.errors":              "Ошибки чтения",
		"html.skipped":             "Пропущено (ссылки, системные папки)",
		"err.no_reports":           "%s: отчётов нет; сначала выполните folder-inspect scan",
		"scan.fallback_dir":        "не удалось создать %s (%v); отчёт будет записан в текущую папку",
		"scan.ui_hint":             "Открыть в браузере: folder-inspect ui \"%s\"",
		"ui.listening":             "Интерфейс открыт: %s  (Ctrl+C — выход)",
		"ui.open_failed":           "не удалось открыть браузер: %v",
		"ui.lang_switch":           "English",
		"ui.summary":               "Сводка",
		"ui.scanned":               "сканирование %s · %s",
		"ui.export":                "Выгрузка",
		"ui.filter":                "Фильтр по пути…",
		"ui.select_visible":        "Отметить видимые",
		"ui.clear_visible":         "Снять отметки",
		"ui.select_copies":         "Отметить копии",
		"ui.select_all_copies":     "Отметить все копии во всех группах",
		"ui.original":              "оригинал",
		"ui.to_quarantine":         "в карантин",
		"ui.reveal":                "Показать в проводнике",
		"ui.no_items":              "Ничего не найдено",
		"ui.plan":                  "План действий",
		"ui.plan_items":            "объектов: %d, %s",
		"ui.plan_empty":            "План пуст: отметьте файлы или копии, которые нужно убрать в карантин",
		"ui.plan_save":             "Сохранить план",
		"ui.plan_clear":            "Очистить",
		"ui.plan_saved":            "План сохранён: %s",
		"ui.plan_error":            "План не сохранён: %s",
		"ui.plan_hint":             "Отметки собираются в план. Ничего не меняется, пока вы не нажмёте «Применить план»: тогда отмеченные файлы переезжают в карантин внутри репозитория, на их месте остаются файлы-заглушки с пояснением, а вернуть всё можно командой restore.",
		"ui.group_hint":            "Жёлтым выделен предлагаемый оригинал (самый старый). Выберите оригинал сами и отметьте копии, которые убрать в карантин: на их месте останется файл-указатель на оригинал.",
		"ui.overlap_hint":          "Пары папок, у которых общее содержимое составляет заметную долю меньшей папки. Это подсказка для ручного разбора; действий по парам нет.",
		"ui.rescan":                "Пересканировать",
		"ui.rescanning":            "Сканирование…",
		"ui.rescan_done":           "Готово: новый отчёт %s",
		"ui.apply_dry":             "Проверить план",
		"ui.apply":                 "Применить план",
		"ui.apply_confirm":         "Убрать %d объектов (%s)?\n\nБольшие файлы, архивы, дистрибутивы и дубликаты переезжают в папку .folder-inspect/quarantine внутри репозитория, на их месте остаются файлы-заглушки с пояснением; вернуть всё можно командой restore. Мусор и пустые файлы и папки удаляются сразу, без карантина и заглушек.",
		"ui.apply_running":         "Выполняю…",
		"ui.apply_result":          "Результат применения",
		"ui.apply_summary":         "в карантин: %d, удалено: %d, заглушек: %d, проблем: %d",
		"ui.apply_dry_note":        "Пробный запуск: на диске ничего не изменилось.",
		"ui.apply_manifest":        "Манифест карантина (для восстановления):",
		"ui.apply_restore":         "Вернуть всё: folder-inspect restore \"%s\"",
		"ui.apply_back":            "К отчёту",
		"ui.problems":              "Проблемы",
		"ui.col_action":            "Действие",
		"ui.col_from":              "Откуда",
		"ui.col_to":                "Куда",
		"ui.col_stub":              "Заглушка",
		"op.quarantine":            "в карантин",
		"op.quarantine-duplicate":  "дубликат в карантин",
		"op.quarantine-dir":        "папка-дубликат в карантин",
		"op.delete":                "удалено без карантина",
		"apply.header":             "План %s: действий %d, %s",
		"apply.dry":                "Пробный запуск: на диске ничего не изменено.",
		"apply.summary":            "Перемещено в карантин: %d, удалено: %d, заглушек: %d, проблем: %d",
		"apply.manifest":           "Манифест карантина: %s",
		"apply.restore_hint":       "Вернуть всё: folder-inspect restore \"%s\"",
		"apply.problems":           "Проблемы:",
		"plan.header":              "Отчёт %s (корней: %d)",
		"plan.rules":               "Правила: категории — %s; дубликаты — %s; папки-дубликаты — %s",
		"plan.filters":             "Фильтры: включить [%s], исключить [%s], не меньше %s",
		"plan.off":                 "нет",
		"plan.keep.oldest":         "оставить самую старую копию",
		"plan.keep.newest":         "оставить самую новую копию",
		"plan.keep.shallowest":     "оставить копию с самым коротким путём",
		"plan.original":            "оригинал: %s",
		"plan.summary":             "Действий в плане: %d (%s) — находок %d, дубликатов %d, папок-дубликатов %d; пропущено %d, групп без действий %d",
		"plan.empty":               "По этим правилам действий нет, план не сохранён.",
		"plan.dry":                 "Пробный запуск: план не сохранён.",
		"plan.saved":               "План сохранён: %s",
		"plan.apply_hint":          "Проверить: folder-inspect apply -dry-run \"%s\"\nВыполнить: folder-inspect apply \"%s\"",
		"restore.header":           "Манифест %s: записей %d, карантин %s",
		"restore.summary":          "Возвращено: %d, проблем: %d",
		"restore.gone":             "Удалено без карантина как мусор, не возвращается: %d (отмечено x)",
		"restore.dry":              "Пробный запуск: на диске ничего не изменено.",
		"err.not_found":            "не найден",
		"err.is_link":              "это ссылка, ссылки не трогаем",
		"err.original_missing":     "оригинал не найден, копия оставлена: %s",
		"err.original_differs":     "оригинал отличается по содержимому, копия оставлена: %s",
		"err.exists_now":           "по этому пути уже что-то есть",
		"err.deleted":              "удалён без карантина как мусор, вернуть нельзя",
		"err.no_manifest":          "%s: манифест карантина не найден",
		"stub.file":                "Файл «%s» перемещён в карантин инструментом folder-inspect %s.",
		"stub.dir":                 "Папка «%s» перемещена в карантин инструментом folder-inspect %s.",
		"stub.duplicate":           "Причина: точная копия другого файла. Оригинал находится здесь:\n  %s",
		"stub.dir-duplicate":       "Причина: содержимое папки полностью совпадает с другой папкой. Оригинал находится здесь:\n  %s",
		"stub.archive":             "Причина: архив. Репозиторий предназначен для документов в прямом доступе, хранить в нём архивы и старые версии файлов не нужно: старые версии доступны в истории изменений.",
		"stub.distributive":        "Причина: дистрибутив. Репозиторий документов — неподходящее место для дистрибутивов. Используйте ссылки на официальные источники или облачное хранилище.",
		"stub.video":               "Причина: видео (%s). Репозиторий документов — неподходящее место для видео и других очень больших файлов. Используйте облачное хранилище и оставляйте ссылку.",
		"stub.oversize":            "Причина: файл слишком большой для репозитория документов (%s, порог %s). Храните такие файлы в облаке или на файловом сервере и оставляйте ссылку.",
		"stub.junk":                "Причина: служебный или временный файл.",
		"stub.generic":             "Причина: %s.",
		"stub.quarantine":          "Карантин: %s",
		"stub.restore":             "Вернуть: folder-inspect restore \"%s\"",
		"unit.B":                   "Б",
		"unit.KB":                  "КБ",
		"unit.MB":                  "МБ",
		"unit.GB":                  "ГБ",
		"unit.TB":                  "ТБ",
	},
	EN: {
		"scan.header":              "%s %s — scan of %s",
		"scan.roots":               "Roots: %d, files: %d, folders: %d, size: %s, time: %s",
		"scan.summary":             "Summary by category",
		"scan.category":            "Category",
		"scan.count":               "Count",
		"scan.size":                "Size",
		"scan.nothing":             "No findings.",
		"scan.top_files":           "Largest files",
		"scan.top_dirs":            "Heaviest folders",
		"scan.errors":              "Read errors: %d",
		"scan.skipped":             "Skipped (links, system folders): %d",
		"scan.more":                "… and %d more",
		"scan.written":             "Report written: %s",
		"scan.exported":            "Export written: %s",
		"scan.hashed":              "Duplicate check: read %d files, %s",
		"scan.threshold":           "threshold",
		"cat.oversize":             "Oversized files",
		"cat.archive":              "Archives",
		"cat.distributive":         "Distributives",
		"cat.duplicate":            "Duplicates",
		"cat.dir-duplicate":        "Duplicate folders",
		"cat.dir-overlap":          "Folders with shared content",
		"cat.similar-name":         "Similar names",
		"names.header":             "%s: %d groups, %d variants, %s",
		"names.legend":             "Files sharing a name once markers like \"Copy\", \"(2)\", \"- copy\", \"_v2\", \"_final\" are stripped. The base file (no marker) is marked *. Contents may differ — a hint for review.",
		"names.base":               "base",
		"names.dup":                "duplicate %s",
		"col.name":                 "Name",
		"col.marker":               "Marker",
		"col.dup_group":            "Duplicate group",
		"sheet.similar_names":      "Similar names",
		"stub.similar-name":        "Reason: judging by the name (marker \"%s\") this is a copy or an old version of a file. The current file is probably here:\n  %s",
		"stub.similar-name-nobase": "Reason: judging by the name (marker \"%s\") this is a copy or an old version of another file.",
		"ui.names_hint":            "Files sharing a name once markers like \"Copy\", \"(2)\", \"_v2\" are stripped. Yellow marks the base file (no marker). Contents may differ, so nothing is pre-selected: open the files and tick the redundant ones.",
		"ui.quarantine":            "Quarantine",
		"ui.quarantine_hint":       "Quarantine batches of this repository. \"Restore\" brings the files back and removes the stubs. Permanent deletion is only available from the console: folder-inspect quarantine purge -yes <batch folder>.",
		"ui.q_empty":               "Quarantine is empty",
		"ui.q_created":             "Date",
		"ui.q_items":               "Items",
		"ui.q_pending":             "In quarantine",
		"ui.q_status":              "Status",
		"ui.q_active":              "in quarantine",
		"ui.q_restored":            "restored",
		"ui.q_partial":             "partially restored",
		"ui.q_purged":              "permanently deleted",
		"ui.q_deleted":             "deleted, no quarantine",
		"ui.q_restore":             "Restore",
		"ui.q_restore_confirm":     "Bring %d items (%s) back from quarantine %s to their original places?",
		"ui.q_restore_done":        "Restored: %d, problems: %d, deleted outright (not restorable): %d",
		"q.list_header":            "Quarantine %s: %d batches",
		"q.none":                   "No quarantine batches.",
		"q.status.active":          "in quarantine",
		"q.status.restored":        "restored",
		"q.status.partial":         "partially restored",
		"q.status.purged":          "permanently deleted",
		"q.status.deleted":         "deleted, no quarantine",
		"q.purge_preview":          "To be deleted permanently: %d items, %s, from %s",
		"q.purge_hint":             "This cannot be undone. To delete, repeat with -yes:\n  folder-inspect quarantine purge -yes \"%s\"",
		"q.purged":                 "Permanently deleted: %d items, %s. The stubs at the files' places remain.",
		"q.already_purged":         "%s: batch already deleted permanently",
		"q.nothing_to_purge":       "%s: nothing to delete in this batch (everything was restored)",
		"dirdup.folders":           "folders: %d",
		"dirdup.files":             "files: %d",
		"overlap.header":           "%s: %d pairs, %s shared",
		"overlap.shared":           "shared files: %d, %s, %d%% of the smaller folder",
		"col.related":              "Second folder",
		"col.ratio":                "Share of smaller",
		"col.ratio_a":              "Share of first",
		"col.ratio_b":              "Share of second",
		"col.shared_files":         "Shared files",
		"col.shared_bytes":         "Shared size, bytes",
		"sheet.dir_duplicates":     "Duplicate folders",
		"sheet.overlaps":           "Shared content",
		"cat.junk":                 "Junk",
		"cat.empty-dir":            "Empty folders",
		"cat.empty-file":           "Empty files",
		"detail.empty":             "empty",
		"detail.no-files":          "only empty sub-folders",
		"dup.header":               "%s: %d groups, %s wasted",
		"dup.legend":               "* = suggested original (oldest modification time); the choice is yours",
		"dup.copies":               "copies: %d",
		"err.exists":               "%s: file exists, add -force to overwrite",
		"col.category":             "Category",
		"col.rule":                 "Rule",
		"col.group":                "Group",
		"col.path":                 "Path",
		"col.root":                 "Root",
		"col.size":                 "Size",
		"col.size_bytes":           "Size, bytes",
		"col.mtime":                "Modified",
		"col.threshold":            "Threshold",
		"col.detail":               "Note",
		"col.count":                "Count",
		"col.wasted":               "Wasted",
		"col.files":                "Files",
		"col.suggested":            "Original",
		"col.metric":               "Metric",
		"col.value":                "Value",
		"col.error":                "Error",
		"sheet.summary":            "Summary",
		"sheet.findings":           "Findings",
		"sheet.duplicates":         "Duplicates",
		"sheet.top_files":          "Largest files",
		"sheet.top_dirs":           "Heaviest folders",
		"sheet.errors":             "Errors",
		"sum.tool":                 "Tool",
		"sum.started":              "Scan started",
		"sum.duration":             "Duration",
		"sum.roots":                "Folders",
		"sum.files":                "Files",
		"sum.dirs":                 "Folders",
		"sum.total":                "Total size",
		"sum.errors":               "Read errors",
		"sum.skipped":              "Skipped",
		"sum.hashed":               "Read for duplicate check",
		"html.title":               "folder-inspect report",
		"html.generated":           "generated %s",
		"html.errors":              "Read errors",
		"html.skipped":             "Skipped (links, system folders)",
		"err.no_reports":           "%s: no reports yet; run folder-inspect scan first",
		"scan.fallback_dir":        "cannot create %s (%v); the report goes to the current folder",
		"scan.ui_hint":             "Open in the browser: folder-inspect ui \"%s\"",
		"ui.listening":             "UI is at %s  (Ctrl+C to quit)",
		"ui.open_failed":           "could not open the browser: %v",
		"ui.lang_switch":           "Русский",
		"ui.summary":               "Summary",
		"ui.scanned":               "scanned %s · %s",
		"ui.export":                "Export",
		"ui.filter":                "Filter by path…",
		"ui.select_visible":        "Select visible",
		"ui.clear_visible":         "Clear selection",
		"ui.select_copies":         "Select copies",
		"ui.select_all_copies":     "Select all copies in all groups",
		"ui.original":              "original",
		"ui.to_quarantine":         "to quarantine",
		"ui.reveal":                "Show in file manager",
		"ui.no_items":              "Nothing found",
		"ui.plan":                  "Action plan",
		"ui.plan_items":            "items: %d, %s",
		"ui.plan_empty":            "The plan is empty: tick files or copies to move to quarantine",
		"ui.plan_save":             "Save plan",
		"ui.plan_clear":            "Clear",
		"ui.plan_saved":            "Plan saved: %s",
		"ui.plan_error":            "Plan not saved: %s",
		"ui.plan_hint":             "Ticks are collected into a plan. Nothing changes until you press \"Apply plan\": then the ticked files move to a quarantine folder inside the repository, stub files with an explanation stay in their place, and the restore command brings everything back.",
		"ui.group_hint":            "Yellow marks the suggested original (oldest). Pick the original yourself and tick the copies to quarantine: a pointer file naming the original stays in their place.",
		"ui.overlap_hint":          "Folder pairs whose shared content is a notable share of the smaller folder. A hint for manual review; there are no actions on pairs.",
		"ui.rescan":                "Rescan",
		"ui.rescanning":            "Scanning…",
		"ui.rescan_done":           "Done: new report %s",
		"ui.apply_dry":             "Check plan",
		"ui.apply":                 "Apply plan",
		"ui.apply_confirm":         "Remove %d items (%s)?\n\nOversized files, archives, distributives and duplicates move to the .folder-inspect/quarantine folder inside the repository, with a short note left in their place; everything can be brought back with restore. Junk and empty files and folders are deleted right away, without quarantine or notes.",
		"ui.apply_running":         "Applying…",
		"ui.apply_result":          "Apply result",
		"ui.apply_summary":         "to quarantine: %d, deleted: %d, stubs: %d, problems: %d",
		"ui.apply_dry_note":        "Dry run: nothing changed on disk.",
		"ui.apply_manifest":        "Quarantine manifest (for restore):",
		"ui.apply_restore":         "Bring everything back: folder-inspect restore \"%s\"",
		"ui.apply_back":            "Back to the report",
		"ui.problems":              "Problems",
		"ui.col_action":            "Action",
		"ui.col_from":              "From",
		"ui.col_to":                "To",
		"ui.col_stub":              "Stub",
		"op.quarantine":            "to quarantine",
		"op.quarantine-duplicate":  "duplicate to quarantine",
		"op.quarantine-dir":        "duplicate folder to quarantine",
		"op.delete":                "deleted, no quarantine",
		"apply.header":             "Plan %s: %d actions, %s",
		"apply.dry":                "Dry run: nothing changed on disk.",
		"apply.summary":            "Moved to quarantine: %d, deleted: %d, stubs: %d, problems: %d",
		"apply.manifest":           "Quarantine manifest: %s",
		"apply.restore_hint":       "Bring everything back: folder-inspect restore \"%s\"",
		"apply.problems":           "Problems:",
		"plan.header":              "Report %s (%d roots)",
		"plan.rules":               "Rules: categories — %s; duplicates — %s; duplicate folders — %s",
		"plan.filters":             "Filters: include [%s], exclude [%s], at least %s",
		"plan.off":                 "off",
		"plan.keep.oldest":         "keep the oldest copy",
		"plan.keep.newest":         "keep the newest copy",
		"plan.keep.shallowest":     "keep the copy with the shortest path",
		"plan.original":            "original: %s",
		"plan.summary":             "Actions in the plan: %d (%s) — findings %d, duplicates %d, duplicate folders %d; skipped %d, groups without actions %d",
		"plan.empty":               "No actions under these rules, plan not saved.",
		"plan.dry":                 "Dry run: plan not saved.",
		"plan.saved":               "Plan saved: %s",
		"plan.apply_hint":          "Check: folder-inspect apply -dry-run \"%s\"\nRun:   folder-inspect apply \"%s\"",
		"restore.header":           "Manifest %s: %d entries, quarantine %s",
		"restore.summary":          "Restored: %d, problems: %d",
		"restore.gone":             "Deleted outright as junk, not restorable: %d (marked x)",
		"restore.dry":              "Dry run: nothing changed on disk.",
		"err.not_found":            "not found",
		"err.is_link":              "it is a link; links are never touched",
		"err.original_missing":     "original not found, copy left in place: %s",
		"err.original_differs":     "original differs in content, copy left in place: %s",
		"err.exists_now":           "something already exists at this path",
		"err.deleted":              "deleted outright as junk, cannot be restored",
		"err.no_manifest":          "%s: quarantine manifest not found",
		"stub.file":                "File \"%s\" was moved to quarantine by folder-inspect on %s.",
		"stub.dir":                 "Folder \"%s\" was moved to quarantine by folder-inspect on %s.",
		"stub.duplicate":           "Reason: exact copy of another file. The original is here:\n  %s",
		"stub.dir-duplicate":       "Reason: the folder content is identical to another folder. The original is here:\n  %s",
		"stub.archive":             "Reason: archive. It makes no sense to store old file versions in the repository: all old versions are available in the change history. The repository is for plain documents and is not intended to store archives.",
		"stub.distributive":        "Reason: distributive. Distributives were removed due to an inappropriate place to store. Better use links to official sources or cloud storage to keep distributives and other huge files.",
		"stub.video":               "Reason: video (%s). Videos were removed due to an inappropriate place to store. Better use links to official sources or cloud storage to keep videos and other huge files.",
		"stub.oversize":            "Reason: the file is too large for a document repository (%s, threshold %s). Keep such files in cloud storage or on a file server and leave a link.",
		"stub.junk":                "Reason: service or temporary file.",
		"stub.generic":             "Reason: %s.",
		"stub.quarantine":          "Quarantine: %s",
		"stub.restore":             "Bring it back: folder-inspect restore \"%s\"",
		"unit.B":                   "B",
		"unit.KB":                  "KB",
		"unit.MB":                  "MB",
		"unit.GB":                  "GB",
		"unit.TB":                  "TB",
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
