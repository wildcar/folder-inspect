// Package detect turns the file index into findings. Detectors are pure
// functions over scan.Result; they do no I/O.
package detect

import (
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/scan"
)

// Category of a finding. The order of Categories is the display order.
type Category string

const (
	Oversize     Category = "oversize"
	Archive      Category = "archive"
	Distributive Category = "distributive"
	Duplicate    Category = "duplicate"
	DirDuplicate Category = "dir-duplicate" // folders with identical content
	DirOverlap   Category = "dir-overlap"   // folder pairs sharing part of their content
	SimilarName  Category = "similar-name"  // copy / version candidates by name
	Junk         Category = "junk"
	EmptyDir     Category = "empty-dir"
	EmptyFile    Category = "empty-file"
)

// Categories in display order.
var Categories = []Category{Oversize, Archive, Distributive, Duplicate, DirDuplicate, DirOverlap, SimilarName, Junk, EmptyDir, EmptyFile}

// GroupCategories are rendered as groups/pairs, not as flat finding lists.
var GroupCategories = map[Category]bool{Duplicate: true, DirDuplicate: true, DirOverlap: true, SimilarName: true}

// Finding is one flagged file or folder.
type Finding struct {
	Category Category  `json:"category"`
	Rule     string    `json:"rule"` // size rule name, matched pattern or extension
	Path     string    `json:"path"`
	Rel      string    `json:"rel"`
	Root     string    `json:"root"`
	Size     int64     `json:"size"`
	ModTime  time.Time `json:"mtime"`
	IsDir    bool      `json:"is_dir,omitempty"`
	// Threshold is the size limit that was exceeded (oversize only).
	Threshold int64 `json:"threshold,omitempty"`
	// Group links the copies of one duplicate group (duplicate only).
	Group string `json:"group,omitempty"`
	// Detail is a machine-readable qualifier, translated for display via
	// the i18n key "detail.<value>" (e.g. "empty", "no-files").
	Detail string `json:"detail,omitempty"`
}

func fromEntry(e scan.Entry, cat Category, rule, detail string) Finding {
	return Finding{
		Category: cat, Rule: rule, Path: e.Path, Rel: e.Rel, Root: e.Root,
		Size: e.Size, ModTime: e.ModTime, IsDir: e.IsDir, Detail: detail,
	}
}

// Ext returns the lower-case extension without the dot ("" if none).
func Ext(name string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
}

// Run applies every I/O-free detector and returns findings sorted by
// category (display order), then size descending, then path. Duplicates
// need file access and are run separately (see Duplicates).
func Run(res *scan.Result, cfg *config.Config) []Finding {
	var out []Finding
	out = append(out, Oversized(res.Files, cfg.SizeRules)...)
	out = append(out, ByExtension(res.Files, cfg.Archives, Archive)...)
	out = append(out, ByExtension(res.Files, cfg.Distributives, Distributive)...)
	out = append(out, JunkFiles(res, cfg.Junk)...)
	out = append(out, EmptyEntries(res)...)
	Sort(out)
	return out
}

var catOrder = func() map[Category]int {
	m := map[Category]int{}
	for i, c := range Categories {
		m[c] = i
	}
	return m
}()

// Sort orders findings for display: category, size descending, path.
func Sort(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.Category != b.Category {
			return catOrder[a.Category] < catOrder[b.Category]
		}
		if a.Size != b.Size {
			return a.Size > b.Size
		}
		return a.Path < b.Path
	})
}
