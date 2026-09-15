package action

import (
	"path/filepath"
	"strings"

	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
)

// stubText composes the "<name>.removed.txt" left where a file or folder
// was: what was removed and when, the reason by category (or the owner's
// custom text), for duplicates the original by relative path, where the
// item is now and how to bring it back.
func stubText(a Action, e Entry, m *Manifest, opt ApplyOptions) string {
	lang := opt.Lang
	name := filepath.Base(a.Path)
	date := opt.Now.Format("2006-01-02 15:04")
	var b strings.Builder
	if e.IsDir {
		b.WriteString(i18n.TLf(lang, "stub.dir", name, date))
	} else {
		b.WriteString(i18n.TLf(lang, "stub.file", name, date))
	}
	b.WriteString("\n\n")

	stubDir := filepath.Dir(e.Stub)
	if custom, ok := opt.StubTexts[a.Category]; ok && strings.TrimSpace(custom) != "" {
		b.WriteString(strings.TrimSpace(custom))
		if a.Original != "" {
			b.WriteString("\n  " + relativeTo(stubDir, a.Original))
		}
	} else {
		b.WriteString(reason(a, e, opt, stubDir))
	}
	b.WriteString("\n\n")
	b.WriteString(i18n.TLf(lang, "stub.quarantine", relativeTo(stubDir, e.To)))
	b.WriteString("\n")
	b.WriteString(i18n.TLf(lang, "stub.restore", m.Path))
	b.WriteString("\n")
	return b.String()
}

func reason(a Action, e Entry, opt ApplyOptions, stubDir string) string {
	lang := opt.Lang
	switch a.Category {
	case "duplicate":
		return i18n.TLf(lang, "stub.duplicate", relativeTo(stubDir, a.Original))
	case "dir-duplicate":
		return i18n.TLf(lang, "stub.dir-duplicate", relativeTo(stubDir, a.Original))
	case "archive":
		return i18n.TL(lang, "stub.archive")
	case "distributive":
		return i18n.TL(lang, "stub.distributive")
	case "junk":
		return i18n.TL(lang, "stub.junk")
	case "similar-name":
		marker := ""
		if f, ok := opt.Findings[a.Path]; ok {
			marker = f.Rule
		}
		if marker == "" {
			marker = "—"
		}
		if a.Original != "" {
			return i18n.TLf(lang, "stub.similar-name", marker, relativeTo(stubDir, a.Original))
		}
		return i18n.TLf(lang, "stub.similar-name-nobase", marker)
	case "oversize":
		size := report.HumanSizeLang(e.Size, lang)
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(a.Path), "."))
		if opt.Videos[ext] {
			return i18n.TLf(lang, "stub.video", size)
		}
		threshold := "—"
		if f, ok := opt.Findings[a.Path]; ok && f.Threshold > 0 {
			threshold = report.HumanSizeLang(f.Threshold, lang)
		}
		return i18n.TLf(lang, "stub.oversize", size, threshold)
	}
	label := a.Category
	if a.Category != "" {
		label = i18n.TL(lang, "cat."+a.Category)
	}
	return i18n.TLf(lang, "stub.generic", label)
}

// relativeTo renders target relative to dir when possible (same volume),
// otherwise absolute.
func relativeTo(dir, target string) string {
	rel, err := filepath.Rel(dir, target)
	if err != nil || strings.HasPrefix(rel, "..") && filepath.VolumeName(dir) != filepath.VolumeName(target) {
		return target
	}
	return rel
}
