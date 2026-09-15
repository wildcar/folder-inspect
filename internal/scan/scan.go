// Package scan walks one or more roots and builds a flat file index with
// per-folder aggregates. It never follows symlinks or junctions, skips
// system folders and the tool's own working folder, and records unreadable
// paths instead of aborting.
package scan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wildcar/folder-inspect/internal/glob"
)

// WorkDir is the tool's own folder inside a scanned root (quarantine etc.).
const WorkDir = ".folder-inspect"

// DefaultSkipDirs are never entered, matched by name, case-insensitively.
var DefaultSkipDirs = []string{"$RECYCLE.BIN", "System Volume Information", WorkDir}

// Entry is a file or folder in the index.
type Entry struct {
	Path    string    `json:"path"`            // absolute, OS-native
	Rel     string    `json:"rel"`             // slash-separated, relative to Root ("" for the root)
	Root    string    `json:"root"`            // the scan root this entry belongs to
	Name    string    `json:"name"`            // base name
	Size    int64     `json:"size"`            // files: own size; folders: total size of files inside
	ModTime time.Time `json:"mtime"`           // last modification
	IsDir   bool      `json:"is_dir"`          // folder
	Files   int       `json:"files,omitempty"` // folders: number of files inside, recursive
	Entries int       `json:"entries,omitempty"`
	// Entries is the number of direct children (files + folders) of a folder.
}

// Error is an unreadable path.
type Error struct {
	Path string `json:"path"`
	Err  string `json:"error"`
}

// Result is the file index of all roots.
type Result struct {
	Roots     []string  `json:"roots"`
	Files     []Entry   `json:"-"`
	Dirs      []Entry   `json:"-"` // excludes the roots themselves
	RootStats []Entry   `json:"root_stats"`
	Errors    []Error   `json:"errors"`
	Skipped   []string  `json:"skipped"` // symlinks, junctions, system folders
	Started   time.Time `json:"started"`
	Finished  time.Time `json:"finished"`
}

// Options tunes the walk.
type Options struct {
	Exclude  []string // glob patterns, see internal/glob
	SkipDirs []string // folder names never entered; nil = DefaultSkipDirs
}

// Walk indexes the roots. It fails only on a root that does not exist or is
// not a folder; problems deeper in the tree land in Result.Errors.
func Walk(roots []string, opt Options) (*Result, error) {
	if len(roots) == 0 {
		return nil, errors.New("no roots given")
	}
	skipDirs := opt.SkipDirs
	if skipDirs == nil {
		skipDirs = DefaultSkipDirs
	}
	skip := make(map[string]bool, len(skipDirs))
	for _, s := range skipDirs {
		skip[strings.ToLower(s)] = true
	}

	res := &Result{Started: time.Now()}
	for _, r := range roots {
		abs, err := filepath.Abs(r)
		if err != nil {
			return nil, err
		}
		real, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return nil, err
		}
		st, err := os.Stat(real)
		if err != nil {
			return nil, err
		}
		if !st.IsDir() {
			return nil, errors.New(r + ": not a folder")
		}
		res.Roots = append(res.Roots, real)
		walkRoot(real, opt.Exclude, skip, res)
	}
	sort.Slice(res.Files, func(i, j int) bool { return less(res.Files[i], res.Files[j]) })
	sort.Slice(res.Dirs, func(i, j int) bool { return less(res.Dirs[i], res.Dirs[j]) })
	res.Finished = time.Now()
	return res, nil
}

func less(a, b Entry) bool {
	if a.Root != b.Root {
		return a.Root < b.Root
	}
	return a.Rel < b.Rel
}

type agg struct {
	size    int64
	files   int
	entries int
}

func walkRoot(root string, exclude []string, skip map[string]bool, res *Result) {
	dirs := map[string]*agg{}
	var dirList []Entry
	rootEntry := Entry{Path: root, Root: root, Name: filepath.Base(root), IsDir: true}

	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			res.Errors = append(res.Errors, Error{Path: p, Err: err.Error()})
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		rel := relSlash(root, p)
		name := d.Name()
		if p != root {
			if d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
				res.Skipped = append(res.Skipped, p)
				return nil
			}
			if d.IsDir() && skip[strings.ToLower(name)] {
				res.Skipped = append(res.Skipped, p)
				return fs.SkipDir
			}
			if glob.MatchAny(exclude, rel, name) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		if parent := dirs[filepath.Dir(p)]; parent != nil && p != root {
			parent.entries++
		}
		if d.IsDir() {
			dirs[p] = &agg{}
			e := Entry{Path: p, Rel: rel, Root: root, Name: name, IsDir: true}
			if info, ierr := d.Info(); ierr == nil {
				e.ModTime = info.ModTime()
			}
			if p == root {
				rootEntry.ModTime = e.ModTime
			} else {
				dirList = append(dirList, e)
			}
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			res.Errors = append(res.Errors, Error{Path: p, Err: ierr.Error()})
			return nil
		}
		e := Entry{Path: p, Rel: rel, Root: root, Name: name, Size: info.Size(), ModTime: info.ModTime()}
		res.Files = append(res.Files, e)
		for dir := filepath.Dir(p); ; {
			if a := dirs[dir]; a != nil {
				a.size += e.Size
				a.files++
			}
			if dir == root {
				break
			}
			next := filepath.Dir(dir)
			if next == dir {
				break
			}
			dir = next
		}
		return nil
	})

	for i := range dirList {
		if a := dirs[dirList[i].Path]; a != nil {
			dirList[i].Size, dirList[i].Files, dirList[i].Entries = a.size, a.files, a.entries
		}
	}
	if a := dirs[root]; a != nil {
		rootEntry.Size, rootEntry.Files, rootEntry.Entries = a.size, a.files, a.entries
	}
	res.Dirs = append(res.Dirs, dirList...)
	res.RootStats = append(res.RootStats, rootEntry)
}

func relSlash(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil || rel == "." {
		return ""
	}
	return filepath.ToSlash(rel)
}

// TotalSize sums the sizes of all roots.
func (r *Result) TotalSize() int64 {
	var n int64
	for _, e := range r.RootStats {
		n += e.Size
	}
	return n
}
