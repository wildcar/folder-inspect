// Package report builds the native scan result (report.json) and renders
// it. Every presentation — console, later the web UI and exports — derives
// from Report, never from a second scan.
package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/scan"
)

// SchemaVersion changes when the JSON layout changes incompatibly.
const SchemaVersion = 1

// Item is a path with a size, used for the top lists.
type Item struct {
	Path  string `json:"path"`
	Rel   string `json:"rel"`
	Root  string `json:"root"`
	Size  int64  `json:"size"`
	Files int    `json:"files,omitempty"` // folders only
}

// CategorySummary is the count and total size of one category.
type CategorySummary struct {
	Category detect.Category `json:"category"`
	Count    int             `json:"count"`
	Size     int64           `json:"size"`
}

// Stats describes the scanned tree as a whole.
type Stats struct {
	Files     int   `json:"files"`
	Dirs      int   `json:"dirs"`
	TotalSize int64 `json:"total_size"`
	Errors    int   `json:"errors"`
	Skipped   int   `json:"skipped"`
}

// Report is the native scan result.
type Report struct {
	Schema   int               `json:"schema"`
	Tool     string            `json:"tool"`
	Version  string            `json:"version"`
	Started  time.Time         `json:"started"`
	Finished time.Time         `json:"finished"`
	Roots    []string          `json:"roots"`
	Config   *config.Config    `json:"config"`
	Stats    Stats             `json:"stats"`
	Summary  []CategorySummary `json:"summary"`
	Findings []detect.Finding  `json:"findings"`
	TopFiles []Item            `json:"top_files"`
	TopDirs  []Item            `json:"top_dirs"`
	Errors   []scan.Error      `json:"errors"`
	Skipped  []string          `json:"skipped"`
}

// Build assembles the report from a scan and its findings.
func Build(res *scan.Result, findings []detect.Finding, cfg *config.Config, version string) *Report {
	r := &Report{
		Schema:   SchemaVersion,
		Tool:     "folder-inspect",
		Version:  version,
		Started:  res.Started,
		Finished: res.Finished,
		Roots:    res.Roots,
		Config:   cfg,
		Findings: findings,
		Errors:   res.Errors,
		Skipped:  res.Skipped,
	}
	if r.Findings == nil {
		r.Findings = []detect.Finding{}
	}
	if r.Errors == nil {
		r.Errors = []scan.Error{}
	}
	if r.Skipped == nil {
		r.Skipped = []string{}
	}
	r.Stats = Stats{
		Files: len(res.Files), Dirs: len(res.Dirs), TotalSize: res.TotalSize(),
		Errors: len(res.Errors), Skipped: len(res.Skipped),
	}

	byCat := map[detect.Category]*CategorySummary{}
	for _, f := range findings {
		s := byCat[f.Category]
		if s == nil {
			s = &CategorySummary{Category: f.Category}
			byCat[f.Category] = s
		}
		s.Count++
		s.Size += f.Size
	}
	r.Summary = []CategorySummary{}
	for _, c := range detect.Categories {
		if s := byCat[c]; s != nil {
			r.Summary = append(r.Summary, *s)
		}
	}

	r.TopFiles = topItems(res.Files, cfg.TopN)
	r.TopDirs = topItems(res.Dirs, cfg.TopN)
	return r
}

func topItems(entries []scan.Entry, n int) []Item {
	idx := make([]int, len(entries))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return entries[idx[a]].Size > entries[idx[b]].Size })
	if n > len(idx) {
		n = len(idx)
	}
	out := make([]Item, 0, n)
	for _, i := range idx[:n] {
		e := entries[i]
		out = append(out, Item{Path: e.Path, Rel: e.Rel, Root: e.Root, Size: e.Size, Files: e.Files})
	}
	return out
}

// Duration of the scan.
func (r *Report) Duration() time.Duration { return r.Finished.Sub(r.Started) }

// WriteJSON saves the report, creating parent folders as needed.
func (r *Report) WriteJSON(path string) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ReadJSON loads a report written by WriteJSON.
func ReadJSON(path string) (*Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Report
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}
