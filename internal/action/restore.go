package action

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/wildcar/folder-inspect/internal/i18n"
)

// RestoreOptions tunes Restore.
type RestoreOptions struct {
	DryRun bool
}

// RestoreResult sums up a restore run.
type RestoreResult struct {
	Restored int
	Problems []Problem
}

// Restore moves every not-yet-restored entry of a manifest back to where
// it was and removes the stub. An entry whose original place is occupied
// again is left in quarantine and reported. The manifest is updated so a
// second restore skips what is already back.
func Restore(m *Manifest, opt RestoreOptions) (*RestoreResult, error) {
	res := &RestoreResult{}
	// Shallower paths first: a folder comes back before the files that were
	// planned inside it.
	order := make([]int, 0, len(m.Entries))
	for i := range m.Entries {
		order = append(order, i)
	}
	sortByDepthAsc(order, m.Entries)

	for _, i := range order {
		e := &m.Entries[i]
		if e.Restored {
			continue
		}
		if _, err := os.Lstat(e.To); err != nil {
			res.Problems = append(res.Problems, Problem{Path: e.To, Err: i18n.T("err.not_found")})
			continue
		}
		if _, err := os.Lstat(e.From); err == nil {
			res.Problems = append(res.Problems, Problem{Path: e.From, Err: i18n.T("err.exists_now")})
			continue
		}
		if opt.DryRun {
			res.Restored++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(e.From), 0o755); err != nil {
			res.Problems = append(res.Problems, Problem{Path: e.From, Err: err.Error()})
			continue
		}
		if err := os.Rename(e.To, e.From); err != nil {
			res.Problems = append(res.Problems, Problem{Path: e.To, Err: err.Error()})
			continue
		}
		if e.Stub != "" {
			if err := os.Remove(e.Stub); err != nil && !errors.Is(err, os.ErrNotExist) {
				res.Problems = append(res.Problems, Problem{Path: e.Stub, Err: err.Error()})
			}
		}
		e.Restored = true
		res.Restored++
	}
	if !opt.DryRun && res.Restored > 0 {
		if err := m.save(); err != nil {
			return res, err
		}
		pruneEmptyDirs(m.Dir, m.Path)
	}
	return res, nil
}

func sortByDepthAsc(idx []int, entries []Entry) {
	for i := 1; i < len(idx); i++ {
		for j := i; j > 0 && depthOf(entries[idx[j]].From) < depthOf(entries[idx[j-1]].From); j-- {
			idx[j], idx[j-1] = idx[j-1], idx[j]
		}
	}
}

// pruneEmptyDirs removes now-empty folders inside the batch folder, keeping
// the manifest itself as the record of what happened.
func pruneEmptyDirs(batch, keep string) {
	var dirs []string
	filepath.WalkDir(batch, func(p string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() && p != batch {
			dirs = append(dirs, p)
		}
		return nil
	})
	for i := len(dirs) - 1; i >= 0; i-- {
		os.Remove(dirs[i]) // fails harmlessly when not empty
	}
	_ = keep
}
