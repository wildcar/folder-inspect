package action

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Batch describes one quarantine batch for listings.
type Batch struct {
	Manifest *Manifest `json:"manifest"`
	Path     string    `json:"path"`     // manifest path
	Items    int       `json:"items"`    // entries in the manifest
	Pending  int       `json:"pending"`  // entries still in quarantine
	Restored int       `json:"restored"` // entries brought back
	Deleted  int       `json:"deleted"`  // entries deleted outright, not restorable
	Size     int64     `json:"size"`     // bytes of the pending entries
	Status   string    `json:"status"`   // active | restored | partial | purged
}

// ListBatches finds every manifest under <root>/<quarantineDir>, newest
// first.
func ListBatches(root, quarantineDir string) ([]Batch, error) {
	if quarantineDir == "" {
		quarantineDir = ".folder-inspect/quarantine"
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	matches, err := filepath.Glob(filepath.Join(abs, filepath.FromSlash(quarantineDir), "*", ManifestName))
	if err != nil {
		return nil, err
	}
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	var out []Batch
	for _, p := range matches {
		m, err := LoadManifest(p)
		if err != nil {
			continue
		}
		out = append(out, Describe(m))
	}
	return out, nil
}

// Describe summarises a manifest.
func Describe(m *Manifest) Batch {
	b := Batch{Manifest: m, Path: m.Path, Items: len(m.Entries)}
	for _, e := range m.Entries {
		switch {
		case e.Restored:
			b.Restored++
		case e.Op == OpDelete && (e.IsDir || e.Size == 0):
			b.Pending++ // an empty item: restore recreates it
		case e.Op == OpDelete:
			b.Deleted++ // junk with content: gone
		case m.Purged == nil: // purged copies are gone: nothing pending
			b.Pending++
			b.Size += e.Size
		}
	}
	switch {
	case m.Purged != nil:
		b.Status = "purged"
	case b.Pending == 0 && b.Restored == 0 && b.Deleted > 0:
		b.Status = "deleted"
	case b.Pending == 0 && b.Items > 0:
		b.Status = "restored"
	case b.Restored > 0:
		b.Status = "partial"
	default:
		b.Status = "active"
	}
	return b
}

// PurgeResult sums up a purge.
type PurgeResult struct {
	Deleted  int
	Bytes    int64
	Problems []Problem
}

// Purge permanently deletes the quarantined copies of a batch (entries not
// restored). It only ever removes paths inside the batch folder, keeps the
// manifest (marked purged) as the record, and leaves the stubs in place.
// Restore is impossible afterwards. dryRun only reports.
func Purge(m *Manifest, dryRun bool) (*PurgeResult, error) {
	if m.Purged != nil {
		return nil, errors.New("already purged")
	}
	res := &PurgeResult{}
	batch := filepath.Clean(m.Dir)
	for i := range m.Entries {
		e := &m.Entries[i]
		if e.Restored || e.Op == OpDelete { // nothing in the batch folder for deleted items
			continue
		}
		to := filepath.Clean(e.To)
		if !strings.HasPrefix(strings.ToLower(to), strings.ToLower(batch)+string(filepath.Separator)) {
			res.Problems = append(res.Problems, Problem{Path: to, Err: "outside the batch folder, left untouched"})
			continue
		}
		if _, err := os.Lstat(to); err != nil {
			res.Problems = append(res.Problems, Problem{Path: to, Err: "not found"})
			continue
		}
		res.Deleted++
		res.Bytes += e.Size
		if dryRun {
			continue
		}
		if err := os.RemoveAll(to); err != nil {
			res.Problems = append(res.Problems, Problem{Path: to, Err: err.Error()})
			res.Deleted--
			res.Bytes -= e.Size
		}
	}
	if !dryRun {
		now := time.Now()
		m.Purged = &now
		if err := m.save(); err != nil {
			return res, err
		}
		pruneEmptyDirs(m.Dir, m.Path)
	}
	return res, nil
}
