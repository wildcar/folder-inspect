package detect

import (
	"path/filepath"

	"github.com/wildcar/folder-inspect/internal/scan"
)

// EmptyEntries reports folders that contain no files (only the top-most one
// of a nested empty chain) and files of zero size.
func EmptyEntries(res *scan.Result) []Finding {
	empty := map[string]bool{}
	for _, d := range res.Dirs {
		if d.Files == 0 {
			empty[d.Path] = true
		}
	}
	var out []Finding
	for _, d := range res.Dirs {
		if d.Files != 0 || empty[filepath.Dir(d.Path)] {
			continue
		}
		detail := "no-files" // only empty sub-folders inside
		if d.Entries == 0 {
			detail = "empty"
		}
		out = append(out, fromEntry(d, EmptyDir, "", detail))
	}
	for _, f := range res.Files {
		if f.Size == 0 {
			out = append(out, fromEntry(f, EmptyFile, "", ""))
		}
	}
	return out
}
