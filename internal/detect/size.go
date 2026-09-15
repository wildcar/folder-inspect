package detect

import (
	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/scan"
)

// Oversized reports files above the threshold of the most specific matching
// size rule. A rule naming the file's extension is tried first; if it does
// not fire, generic ("*") rules are tried in order. A file is reported once.
func Oversized(files []scan.Entry, rules []config.SizeRule) []Finding {
	byExt := map[string]*config.SizeRule{}
	var generic []*config.SizeRule
	for i := range rules {
		r := &rules[i]
		if r.IsGeneric() {
			generic = append(generic, r)
			continue
		}
		for _, e := range r.Extensions {
			if _, dup := byExt[e]; !dup {
				byExt[e] = r
			}
		}
	}

	var out []Finding
	for _, f := range files {
		if r := byExt[Ext(f.Name)]; r != nil && f.Size > int64(r.Threshold) {
			out = append(out, oversize(f, r))
			continue
		}
		for _, g := range generic {
			if f.Size > int64(g.Threshold) {
				out = append(out, oversize(f, g))
				break
			}
		}
	}
	return out
}

func oversize(f scan.Entry, r *config.SizeRule) Finding {
	fd := fromEntry(f, Oversize, r.Name, "")
	fd.Threshold = int64(r.Threshold)
	return fd
}
