package detect

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/wildcar/folder-inspect/internal/scan"
)

// Copy and version markers people add to file names, RU + EN. Applied to the
// stem (name without extension), case-insensitively, until nothing matches.
var (
	namePrefixes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(?:копия|copy of)\s+`),
	}
	nameSuffixes = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\s*[-–—]\s*(?:копия|copy)(?:\s*\(\d+\))?$`),
		regexp.MustCompile(`(?i)\s*\(\d+\)$`),
		regexp.MustCompile(`(?i)\s*\((?:восстановлен[оа]?|recovered|копия|copy|final|финал|итог|old|старый|new|новый|draft|черновик)\)$`),
		regexp.MustCompile(`(?i)[\s_.-]+(?:v|ver|версия|version)\.?\s*\d+(?:[._]\d+)*$`),
		regexp.MustCompile(`(?i)[\s_-]+(?:final|финал|итог|итоговый|итоговая|old|старый|старая|старое|new|новый|новая|новое|копия|copy|backup|бэкап|bak|draft|черновик|last|latest|последний|последняя)$`),
	}
)

// SplitName strips copy/version markers from a file stem. It returns the
// base stem and the markers removed (outermost first); markers is empty
// when the stem carries none.
func SplitName(stem string) (base string, markers []string) {
	base = strings.TrimSpace(stem)
	for i := 0; i < 10; i++ {
		changed := false
		for _, re := range namePrefixes {
			if m := re.FindString(base); m != "" {
				markers = append(markers, strings.TrimSpace(m))
				base = strings.TrimSpace(base[len(m):])
				changed = true
			}
		}
		for _, re := range nameSuffixes {
			if loc := re.FindStringIndex(base); loc != nil && loc[0] > 0 {
				markers = append(markers, strings.Trim(base[loc[0]:], " _-–—."))
				base = strings.TrimSpace(base[:loc[0]])
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	return base, markers
}

// NameFile is a member of a similar-name group.
type NameFile struct {
	Path     string    `json:"path"`
	Rel      string    `json:"rel"`
	Root     string    `json:"root"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mtime"`
	Marker   string    `json:"marker,omitempty"`    // what was stripped: "копия (2)", "v2"…
	IsBase   bool      `json:"is_base"`             // carries no marker
	DupGroup string    `json:"dup_group,omitempty"` // exact-duplicate group it belongs to, if any
}

// NameGroup is a set of files that share a base name once copy/version
// markers are stripped: "Отчёт.docx", "Копия Отчёт.docx", "Отчёт (2).docx".
type NameGroup struct {
	ID       string     `json:"id"`
	Name     string     `json:"name"`           // base name with extension, display form
	Count    int        `json:"count"`          // files in the group
	Variants int        `json:"variants"`       // files carrying a marker
	Size     int64      `json:"size"`           // total size of the variants
	Base     string     `json:"base,omitempty"` // newest file without a marker, if any
	Files    []NameFile `json:"files"`          // base files first, then newest first
}

// NameResult is the outcome of SimilarNames.
type NameResult struct {
	Groups []NameGroup `json:"groups"`
}

// SimilarNames groups files that sit in the same folder by base name
// (markers stripped) and extension, and reports groups of two or more
// files where at least one carries a marker. Grouping is per folder by the
// owner's decision (2026-09-15): a copy usually lands next to its source,
// and repository-wide grouping was too noisy. Purely informational:
// contents may differ.
func SimilarNames(files []scan.Entry, dups DupResult) NameResult {
	dupOf := map[string]string{}
	for _, g := range dups.Groups {
		for _, f := range g.Files {
			dupOf[f.Path] = g.ID
		}
	}
	type member struct {
		file   scan.Entry
		base   string
		marker string
	}
	groups := map[string][]member{}
	for _, f := range files {
		ext := filepath.Ext(f.Name)
		stem := strings.TrimSuffix(f.Name, ext)
		base, markers := SplitName(stem)
		if base == "" {
			continue
		}
		key := strings.ToLower(filepath.Dir(f.Path)) + "\x00" + strings.ToLower(base) + "\x00" + strings.ToLower(ext)
		groups[key] = append(groups[key], member{f, base, strings.Join(markers, ", ")})
	}

	var out NameResult
	for key, ms := range groups {
		if len(ms) < 2 {
			continue
		}
		variants := 0
		for _, m := range ms {
			if m.marker != "" {
				variants++
			}
		}
		if variants == 0 {
			continue
		}
		sum := sha256.Sum256([]byte(key))
		g := NameGroup{ID: hex.EncodeToString(sum[:])[:12], Count: len(ms), Variants: variants}
		ext := filepath.Ext(ms[0].file.Name)
		for _, m := range ms {
			nf := NameFile{Path: m.file.Path, Rel: m.file.Rel, Root: m.file.Root, Size: m.file.Size, ModTime: m.file.ModTime,
				Marker: m.marker, IsBase: m.marker == "", DupGroup: dupOf[m.file.Path]}
			if nf.IsBase {
				g.Name = m.base + ext
			} else {
				g.Size += m.file.Size
			}
			g.Files = append(g.Files, nf)
		}
		if g.Name == "" {
			g.Name = ms[0].base + ext
		}
		sort.Slice(g.Files, func(a, b int) bool {
			fa, fb := g.Files[a], g.Files[b]
			if fa.IsBase != fb.IsBase {
				return fa.IsBase
			}
			if !fa.ModTime.Equal(fb.ModTime) {
				return fa.ModTime.After(fb.ModTime)
			}
			return fa.Path < fb.Path
		})
		if g.Files[0].IsBase {
			g.Base = g.Files[0].Path
		}
		out.Groups = append(out.Groups, g)
	}
	sort.Slice(out.Groups, func(a, b int) bool {
		if out.Groups[a].Size != out.Groups[b].Size {
			return out.Groups[a].Size > out.Groups[b].Size
		}
		return out.Groups[a].Name < out.Groups[b].Name
	})
	return out
}

// Findings flattens the groups: one finding per member; Rule carries the
// marker ("" for the base file), Group the group id.
func (r NameResult) Findings() []Finding {
	var out []Finding
	for _, g := range r.Groups {
		for _, f := range g.Files {
			out = append(out, Finding{
				Category: SimilarName, Rule: f.Marker, Group: g.ID,
				Path: f.Path, Rel: f.Rel, Root: f.Root, Size: f.Size, ModTime: f.ModTime,
			})
		}
	}
	return out
}
