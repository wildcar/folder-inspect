package detect

import "github.com/wildcar/folder-inspect/internal/scan"

// ByExtension reports every file whose extension is in exts under the given
// category. Used for archives and installers/distributives: their presence
// in a document repository is the finding, regardless of size.
func ByExtension(files []scan.Entry, exts []string, cat Category) []Finding {
	set := make(map[string]bool, len(exts))
	for _, e := range exts {
		set[e] = true
	}
	var out []Finding
	for _, f := range files {
		if ext := Ext(f.Name); set[ext] {
			out = append(out, fromEntry(f, cat, ext, ""))
		}
	}
	return out
}
