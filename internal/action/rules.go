package action

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/glob"
	"github.com/wildcar/folder-inspect/internal/report"
)

// Rules describe a plan assembled without the UI (`plan` command): which
// finding categories go to quarantine wholesale, how duplicate groups pick
// the copy that stays, and path/size filters. Categories where a person must
// look first (similar names, overlapping folders) cannot be selected.
type Rules struct {
	Categories    []detect.Category // findings to quarantine: junk, archive, distributive, oversize, empty-dir, empty-file
	Include       []string          // globs on the relative path or name; empty = everything
	Exclude       []string          // globs on the relative path or name
	MinSize       int64             // skip items (or duplicate groups) smaller than this
	Duplicates    KeepPolicy        // "" = leave duplicate files alone
	DirDuplicates KeepPolicy        // "" = leave identical folders alone
}

// KeepPolicy says which copy of a duplicate group stays when a plan is built
// from rules. It is the caller's explicit choice, never a default.
type KeepPolicy string

const (
	KeepOldest     KeepPolicy = "oldest"     // the copy with the oldest modification time (the UI's suggestion)
	KeepNewest     KeepPolicy = "newest"     // the most recently modified copy
	KeepShallowest KeepPolicy = "shallowest" // the copy with the fewest path components, then the shortest path
)

// RuleCategories lists the finding categories `plan` may quarantine by rule,
// in report order. "all" in the CLI expands to this list.
var RuleCategories = []detect.Category{
	detect.Junk, detect.Archive, detect.Distributive, detect.Oversize, detect.EmptyDir, detect.EmptyFile,
}

// ParseKeepPolicy validates a -duplicates / -dir-duplicates value.
func ParseKeepPolicy(s string) (KeepPolicy, error) {
	switch KeepPolicy(strings.ToLower(strings.TrimSpace(s))) {
	case "":
		return "", nil
	case KeepOldest:
		return KeepOldest, nil
	case KeepNewest:
		return KeepNewest, nil
	case KeepShallowest:
		return KeepShallowest, nil
	}
	return "", fmt.Errorf("unknown keep policy %q (oldest, newest, shallowest)", s)
}

// ParseCategories parses "junk,archive" or "all" into rule categories.
func ParseCategories(s string) ([]detect.Category, error) {
	var out []detect.Category
	seen := map[detect.Category]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		if part == "all" {
			for _, c := range RuleCategories {
				if !seen[c] {
					seen[c] = true
					out = append(out, c)
				}
			}
			continue
		}
		c := detect.Category(part)
		if !ruleCategory(c) {
			if c == detect.Duplicate || c == detect.DirDuplicate {
				return nil, fmt.Errorf("%s: use -duplicates / -dir-duplicates with a keep policy", part)
			}
			if c == detect.SimilarName || c == detect.DirOverlap {
				return nil, fmt.Errorf("%s: needs a person's judgement, pick items in the UI", part)
			}
			return nil, fmt.Errorf("unknown category %q", part)
		}
		if !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	return out, nil
}

func ruleCategory(c detect.Category) bool {
	for _, r := range RuleCategories {
		if r == c {
			return true
		}
	}
	return false
}

// RuleStats says what FromRules did and did not take.
type RuleStats struct {
	Findings      int // finding actions added
	Duplicates    int // duplicate copies added
	DirDuplicates int // duplicate folders added
	Skipped       int // items dropped by filters or nesting
	GroupsSkipped int // duplicate groups with nothing left to quarantine
}

// FromRules builds a plan from a report according to r. Nothing is touched on
// disk; the result still has to be saved and applied. Folder actions win over
// items inside them; a duplicate group whose kept copy would sit inside a
// quarantined folder keeps another copy instead.
func FromRules(rep *report.Report, reportPath string, r Rules) (*Plan, RuleStats, error) {
	var st RuleStats
	if len(r.Categories) == 0 && r.Duplicates == "" && r.DirDuplicates == "" {
		return nil, st, errors.New("nothing selected: pass categories, -duplicates or -dir-duplicates")
	}
	for _, c := range r.Categories {
		if !ruleCategory(c) {
			return nil, st, fmt.Errorf("category %s cannot be planned by rule", c)
		}
	}
	p := New(reportPath, rep.Roots)
	p.Created = time.Now()

	// planned folders (lower-cased, cleaned) — items inside them are redundant;
	// kept folders — a file inside one must stay, or the kept folder loses content.
	var dirs, kept []string
	under := func(path string, parents []string) bool {
		clean := strings.ToLower(filepath.Clean(path))
		for _, d := range parents {
			if strings.HasPrefix(clean, d+string(filepath.Separator)) {
				return true
			}
		}
		return false
	}
	inPlannedDir := func(path string) bool { return under(path, dirs) }
	allowed := func(rel, name string, size int64) bool {
		if size < r.MinSize {
			return false
		}
		if len(r.Include) > 0 && !glob.MatchAny(r.Include, rel, name) {
			return false
		}
		return !glob.MatchAny(r.Exclude, rel, name)
	}

	// 1. Identical folders: decide first so file actions can defer to them.
	if r.DirDuplicates != "" {
		for _, g := range rep.DirDuplicates {
			if g.Size < r.MinSize {
				st.GroupsSkipped++
				continue
			}
			var cands []dupItem
			for _, d := range g.Dirs {
				if inPlannedDir(d.Path) {
					st.Skipped++
					continue
				}
				cands = append(cands, dupItem{Path: d.Path, Rel: d.Rel, ModTime: d.ModTime})
			}
			keep, copies := pick(cands, r.DirDuplicates)
			if keep == nil || len(copies) == 0 {
				st.GroupsSkipped++
				continue
			}
			kept = append(kept, strings.ToLower(filepath.Clean(keep.Path)))
			added := 0
			for _, c := range copies {
				if !allowed(c.Rel, filepath.Base(c.Path), g.Size) {
					st.Skipped++
					continue
				}
				p.Actions = append(p.Actions, Action{Op: OpQuarantineDir, Path: c.Path, Original: keep.Path, Group: g.ID, Category: string(detect.DirDuplicate), Size: g.Size, IsDir: true})
				dirs = append(dirs, strings.ToLower(filepath.Clean(c.Path)))
				added++
			}
			if added == 0 {
				st.GroupsSkipped++
			}
			st.DirDuplicates += added
		}
	}

	// 2. Findings by category, in report order.
	want := map[detect.Category]bool{}
	for _, c := range r.Categories {
		want[c] = true
	}
	planned := map[string]bool{}
	for _, a := range p.Actions {
		planned[strings.ToLower(a.Path)] = true
	}
	for _, f := range rep.Findings {
		if !want[f.Category] {
			continue
		}
		if planned[strings.ToLower(f.Path)] || inPlannedDir(f.Path) {
			st.Skipped++
			continue
		}
		if !allowed(f.Rel, filepath.Base(f.Path), f.Size) {
			st.Skipped++
			continue
		}
		p.Actions = append(p.Actions, Action{Op: OpQuarantine, Path: f.Path, Category: string(f.Category), Size: f.Size, IsDir: f.IsDir})
		planned[strings.ToLower(f.Path)] = true
		if f.IsDir {
			dirs = append(dirs, strings.ToLower(filepath.Clean(f.Path)))
		}
		st.Findings++
	}

	// 3. Duplicate files: the kept copy must stay outside everything planned;
	// copies inside a kept folder are protected and stay too.
	if r.Duplicates != "" {
		for _, g := range rep.Duplicates {
			if g.Size < r.MinSize {
				st.GroupsSkipped++
				continue
			}
			var cands, protected []dupItem
			for _, f := range g.Files {
				if planned[strings.ToLower(f.Path)] || inPlannedDir(f.Path) {
					st.Skipped++
					continue
				}
				it := dupItem{Path: f.Path, Rel: f.Rel, ModTime: f.ModTime}
				if under(f.Path, kept) {
					protected = append(protected, it)
				} else {
					cands = append(cands, it)
				}
			}
			var keep *dupItem
			var copies []dupItem
			if len(protected) > 0 {
				keep, copies = &protected[0], cands
				if len(protected) > 1 {
					keep, _ = pick(protected, r.Duplicates)
				}
			} else {
				keep, copies = pick(cands, r.Duplicates)
			}
			if keep == nil || len(copies) == 0 {
				st.GroupsSkipped++
				continue
			}
			added := 0
			for _, c := range copies {
				if !allowed(c.Rel, filepath.Base(c.Path), g.Size) {
					st.Skipped++
					continue
				}
				p.Actions = append(p.Actions, Action{Op: OpQuarantineDuplicate, Path: c.Path, Original: keep.Path, Group: g.ID, Category: string(detect.Duplicate), Size: g.Size})
				planned[strings.ToLower(c.Path)] = true
				added++
			}
			if added == 0 {
				st.GroupsSkipped++
			}
			st.Duplicates += added
		}
	}

	if err := p.Validate(); err != nil {
		return nil, st, err
	}
	return p, st, nil
}

type dupItem struct {
	Path    string
	Rel     string
	ModTime time.Time
}

// pick chooses the copy that stays per policy; the rest are copies to
// quarantine, in a stable order. Fewer than two candidates → nothing.
func pick(items []dupItem, policy KeepPolicy) (*dupItem, []dupItem) {
	if len(items) < 2 {
		return nil, nil
	}
	sorted := make([]dupItem, len(items))
	copy(sorted, items)
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		switch policy {
		case KeepNewest:
			if !a.ModTime.Equal(b.ModTime) {
				return a.ModTime.After(b.ModTime)
			}
		case KeepShallowest:
			da, db := strings.Count(a.Path, string(filepath.Separator)), strings.Count(b.Path, string(filepath.Separator))
			if da != db {
				return da < db
			}
			if len(a.Path) != len(b.Path) {
				return len(a.Path) < len(b.Path)
			}
		default: // KeepOldest
			if !a.ModTime.Equal(b.ModTime) {
				return a.ModTime.Before(b.ModTime)
			}
		}
		return strings.ToLower(a.Path) < strings.ToLower(b.Path)
	})
	keep := sorted[0]
	return &keep, sorted[1:]
}
