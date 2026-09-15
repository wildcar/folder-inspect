package detect

import (
	"path/filepath"
	"strings"

	"github.com/wildcar/folder-inspect/internal/glob"
	"github.com/wildcar/folder-inspect/internal/scan"
)

// JunkFiles reports files and folders whose name matches a junk pattern.
// A matching folder is one finding with its total size; files inside it are
// not reported again.
func JunkFiles(res *scan.Result, patterns []string) []Finding {
	var out []Finding
	var junkDirs []string
	for _, d := range res.Dirs {
		if under(junkDirs, d.Path) {
			continue
		}
		if p := firstMatch(patterns, d.Name); p != "" {
			out = append(out, fromEntry(d, Junk, p, ""))
			junkDirs = append(junkDirs, d.Path)
		}
	}
	for _, f := range res.Files {
		if under(junkDirs, f.Path) {
			continue
		}
		if p := firstMatch(patterns, f.Name); p != "" {
			out = append(out, fromEntry(f, Junk, p, ""))
		}
	}
	return out
}

func firstMatch(patterns []string, name string) string {
	for _, p := range patterns {
		if glob.Match(p, name) {
			return p
		}
	}
	return ""
}

func under(dirs []string, p string) bool {
	for _, d := range dirs {
		if strings.HasPrefix(p, d+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
