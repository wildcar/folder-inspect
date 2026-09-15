// Package report builds the native scan result (report.json) and renders
// it. Every presentation — console, exports, later the web UI — derives
// from Report, never from a second scan.
package report

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/scan"
)

// SchemaVersion changes when the JSON layout changes incompatibly.
// 2: added duplicates, findings[].group, stats.hashed*.
// 3: added dir_duplicates, dir_overlaps.
const SchemaVersion = 3

// Item is a path with a size, used for the top lists.
type Item struct {
	Path  string `json:"path"`
	Rel   string `json:"rel"`
	Root  string `json:"root"`
	Size  int64  `json:"size"`
	Files int    `json:"files,omitempty"` // folders only
}

// CategorySummary is the count and total size of one category. For
// duplicates (files and folders) Count is the number of groups and Size the
// wasted bytes; for folder overlaps Count is pairs and Size the shared bytes.
type CategorySummary struct {
	Category detect.Category `json:"category"`
	Count    int             `json:"count"`
	Size     int64           `json:"size"`
}

// Stats describes the scanned tree as a whole.
type Stats struct {
	Files       int   `json:"files"`
	Dirs        int   `json:"dirs"`
	TotalSize   int64 `json:"total_size"`
	Errors      int   `json:"errors"`
	Skipped     int   `json:"skipped"`
	Hashed      int   `json:"hashed"`       // files read for duplicate detection
	HashedBytes int64 `json:"hashed_bytes"` // bytes read for duplicate detection
}

// Report is the native scan result.
type Report struct {
	Schema        int                  `json:"schema"`
	Tool          string               `json:"tool"`
	Version       string               `json:"version"`
	Started       time.Time            `json:"started"`
	Finished      time.Time            `json:"finished"`
	Roots         []string             `json:"roots"`
	Config        *config.Config       `json:"config"`
	Stats         Stats                `json:"stats"`
	Summary       []CategorySummary    `json:"summary"`
	Findings      []detect.Finding     `json:"findings"`
	Duplicates    []detect.DupGroup    `json:"duplicates"`
	DirDuplicates []detect.DirGroup    `json:"dir_duplicates"`
	DirOverlaps   []detect.OverlapPair `json:"dir_overlaps"`
	TopFiles      []Item               `json:"top_files"`
	TopDirs       []Item               `json:"top_dirs"`
	Errors        []scan.Error         `json:"errors"`
	Skipped       []string             `json:"skipped"`
}

// Build assembles the report from a scan, its findings (which should
// already include dups.Findings() and dirs.Findings()) and the duplicate
// detection results.
func Build(res *scan.Result, findings []detect.Finding, dups detect.DupResult, dirs detect.DirDupResult, cfg *config.Config, version string) *Report {
	r := &Report{
		Schema:        SchemaVersion,
		Tool:          "folder-inspect",
		Version:       version,
		Started:       res.Started,
		Finished:      res.Finished,
		Roots:         res.Roots,
		Config:        cfg,
		Findings:      findings,
		Duplicates:    dups.Groups,
		DirDuplicates: dirs.Groups,
		DirOverlaps:   dirs.Overlaps,
		Errors:        append(append([]scan.Error{}, res.Errors...), dups.Errors...),
		Skipped:       res.Skipped,
	}
	if r.Findings == nil {
		r.Findings = []detect.Finding{}
	}
	if r.Duplicates == nil {
		r.Duplicates = []detect.DupGroup{}
	}
	if r.DirDuplicates == nil {
		r.DirDuplicates = []detect.DirGroup{}
	}
	if r.DirOverlaps == nil {
		r.DirOverlaps = []detect.OverlapPair{}
	}
	if r.Skipped == nil {
		r.Skipped = []string{}
	}
	r.Stats = Stats{
		Files: len(res.Files), Dirs: len(res.Dirs), TotalSize: res.TotalSize(),
		Errors: len(r.Errors), Skipped: len(res.Skipped),
		Hashed: dups.Hashed, HashedBytes: dups.HashedBytes,
	}

	byCat := map[detect.Category]*CategorySummary{}
	for _, f := range findings {
		if detect.GroupCategories[f.Category] {
			continue
		}
		s := byCat[f.Category]
		if s == nil {
			s = &CategorySummary{Category: f.Category}
			byCat[f.Category] = s
		}
		s.Count++
		s.Size += f.Size
	}
	if len(dups.Groups) > 0 {
		s := &CategorySummary{Category: detect.Duplicate, Count: len(dups.Groups)}
		for _, g := range dups.Groups {
			s.Size += g.Wasted
		}
		byCat[detect.Duplicate] = s
	}
	if len(dirs.Groups) > 0 {
		s := &CategorySummary{Category: detect.DirDuplicate, Count: len(dirs.Groups)}
		for _, g := range dirs.Groups {
			s.Size += g.Wasted
		}
		byCat[detect.DirDuplicate] = s
	}
	if len(dirs.Overlaps) > 0 {
		s := &CategorySummary{Category: detect.DirOverlap, Count: len(dirs.Overlaps)}
		for _, o := range dirs.Overlaps {
			s.Size += o.SharedBytes
		}
		byCat[detect.DirOverlap] = s
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

// Duration of the scan including duplicate hashing.
func (r *Report) Duration() time.Duration { return r.Finished.Sub(r.Started) }

// ByCategory returns the findings of one category in report order.
func (r *Report) ByCategory(c detect.Category) []detect.Finding {
	var out []detect.Finding
	for _, f := range r.Findings {
		if f.Category == c {
			out = append(out, f)
		}
	}
	return out
}

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
	if r.Tool != "folder-inspect" || r.Schema == 0 {
		return nil, errors.New(path + ": not a folder-inspect report")
	}
	return &r, nil
}

// DefaultName is the report file name for a scan started at t; it never
// collides with a previous scan, so nothing is overwritten by accident.
func DefaultName(t time.Time) string {
	return "report-" + t.Format("2006-01-02_150405") + ".json"
}

// UniquePath returns path if nothing exists there, else path with a "-2",
// "-3"… suffix before the extension. Used for default names, so two scans
// within one second do not collide; explicit -out paths still need -force.
func UniquePath(path string) string {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		p := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
			return p
		}
	}
}

// DefaultDir is where reports (and plans, exports) of a root live:
// <root>/.folder-inspect/reports. The scanner never enters that folder.
func DefaultDir(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		abs = root
	}
	return filepath.Join(abs, scan.WorkDir, "reports")
}

// NewestReport returns the most recently modified report-*.json in dir.
func NewestReport(dir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "report-*.json"))
	if err != nil {
		return "", err
	}
	var best string
	var bestTime time.Time
	for _, m := range matches {
		st, err := os.Stat(m)
		if err != nil || st.IsDir() {
			continue
		}
		if best == "" || st.ModTime().After(bestTime) {
			best, bestTime = m, st.ModTime()
		}
	}
	if best == "" {
		return "", errors.New(i18n.Tf("err.no_reports", dir))
	}
	return best, nil
}

// CheckOverwrite fails when path exists and force is false. Every file the
// tool writes on the user's behalf goes through this.
func CheckOverwrite(path string, force bool) error {
	if force {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		return errors.New(i18n.Tf("err.exists", path))
	}
	return nil
}

// SiblingPath swaps the extension of a report path: report-x.json + "html"
// → report-x.html. Used to name exports next to the JSON.
func SiblingPath(reportPath, ext string) string {
	return strings.TrimSuffix(reportPath, filepath.Ext(reportPath)) + "." + strings.TrimPrefix(ext, ".")
}

// FormatTime renders a timestamp for people; empty for the zero time.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04")
}

// Qualifier is the text after a finding: rule, exceeded threshold and the
// translated detail, comma-separated; "" when there is nothing to say.
func Qualifier(f detect.Finding) string {
	var parts []string
	if f.Rule != "" {
		parts = append(parts, f.Rule)
	}
	if f.Threshold > 0 {
		parts = append(parts, "> "+HumanSize(f.Threshold))
	}
	if f.Detail != "" {
		parts = append(parts, i18n.T("detail."+f.Detail))
	}
	return strings.Join(parts, ", ")
}

// CategoryName is the translated display name of a category.
func CategoryName(c detect.Category) string { return i18n.T("cat." + string(c)) }

// Percent renders a 0..1 ratio as a whole percentage.
func Percent(ratio float64) string { return fmt.Sprintf("%d%%", int(ratio*100+0.5)) }

// OverlapLine describes a folder pair: shared files, bytes and share.
func OverlapLine(o detect.OverlapPair) string {
	return i18n.Tf("overlap.shared", o.SharedFiles, HumanSize(o.SharedBytes), int(o.Ratio*100+0.5))
}
