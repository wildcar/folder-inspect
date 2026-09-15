package detect

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/wildcar/folder-inspect/internal/scan"
)

// DefaultHeadSize is how much of a file is hashed in the first pass.
const DefaultHeadSize = 64 << 10

// DupFile is one member of a duplicate group.
type DupFile struct {
	Path    string    `json:"path"`
	Rel     string    `json:"rel"`
	Root    string    `json:"root"`
	ModTime time.Time `json:"mtime"`
}

// DupGroup is a set of byte-identical files.
type DupGroup struct {
	ID        string    `json:"id"`        // short hash prefix, stable across scans
	Hash      string    `json:"hash"`      // full SHA-256, hex
	Size      int64     `json:"size"`      // size of one file
	Count     int       `json:"count"`     // number of copies
	Wasted    int64     `json:"wasted"`    // Size × (Count − 1)
	Suggested string    `json:"suggested"` // path the UI pre-selects as the original: oldest mtime
	Files     []DupFile `json:"files"`     // sorted by mtime, then path
}

// DupOptions tunes duplicate detection.
type DupOptions struct {
	MinSize  int64 // files smaller than this are ignored
	HeadSize int64 // bytes hashed in the first pass (0 = DefaultHeadSize)
	Workers  int   // parallel hashers (0 = NumCPU)
}

// DupResult is the outcome of Duplicates.
type DupResult struct {
	Groups      []DupGroup   `json:"groups"`
	Errors      []scan.Error `json:"errors"`
	Hashed      int          `json:"hashed"`       // files read (both passes)
	HashedBytes int64        `json:"hashed_bytes"` // bytes read
}

// Duplicates finds byte-identical files. This is the one detector that does
// I/O: candidates are grouped by size, then by a hash of the first HeadSize
// bytes, and only the survivors are hashed in full. Hashing runs in
// parallel; unreadable files land in Errors.
func Duplicates(files []scan.Entry, opt DupOptions) DupResult {
	if opt.HeadSize <= 0 {
		opt.HeadSize = DefaultHeadSize
	}
	if opt.Workers <= 0 {
		opt.Workers = runtime.NumCPU()
	}
	var res DupResult

	bySize := map[int64][]int{}
	for i, f := range files {
		if f.Size > 0 && f.Size >= opt.MinSize {
			bySize[f.Size] = append(bySize[f.Size], i)
		}
	}
	var cand []int
	for _, idx := range bySize {
		if len(idx) > 1 {
			cand = append(cand, idx...)
		}
	}
	if len(cand) == 0 {
		return res
	}

	type key struct {
		size int64
		hash string
	}
	head := hashAll(files, cand, opt.HeadSize, opt.Workers, &res)
	byHead := map[key][]int{}
	for _, i := range cand {
		if h, ok := head[i]; ok {
			k := key{files[i].Size, h}
			byHead[k] = append(byHead[k], i)
		}
	}

	final := map[key][]int{}
	var full []int
	for k, idx := range byHead {
		if len(idx) < 2 {
			continue
		}
		if k.size <= opt.HeadSize { // the head is the whole file
			final[k] = idx
			continue
		}
		full = append(full, idx...)
	}
	if len(full) > 0 {
		fh := hashAll(files, full, -1, opt.Workers, &res)
		for _, i := range full {
			if h, ok := fh[i]; ok {
				k := key{files[i].Size, h}
				final[k] = append(final[k], i)
			}
		}
	}

	for k, idx := range final {
		if len(idx) < 2 {
			continue
		}
		g := DupGroup{ID: k.hash[:12], Hash: k.hash, Size: k.size, Count: len(idx), Wasted: k.size * int64(len(idx)-1)}
		for _, i := range idx {
			f := files[i]
			g.Files = append(g.Files, DupFile{Path: f.Path, Rel: f.Rel, Root: f.Root, ModTime: f.ModTime})
		}
		sort.Slice(g.Files, func(a, b int) bool {
			fa, fb := g.Files[a], g.Files[b]
			if !fa.ModTime.Equal(fb.ModTime) {
				return fa.ModTime.Before(fb.ModTime)
			}
			return fa.Path < fb.Path
		})
		g.Suggested = g.Files[0].Path
		res.Groups = append(res.Groups, g)
	}
	sort.Slice(res.Groups, func(a, b int) bool {
		if res.Groups[a].Wasted != res.Groups[b].Wasted {
			return res.Groups[a].Wasted > res.Groups[b].Wasted
		}
		return res.Groups[a].Hash < res.Groups[b].Hash
	})
	sort.Slice(res.Errors, func(a, b int) bool { return res.Errors[a].Path < res.Errors[b].Path })
	return res
}

// Findings flattens the groups: one finding per copy, Rule and Group set
// to the group id, so the flat lists and exports can show duplicates too.
func (r DupResult) Findings() []Finding {
	var out []Finding
	for _, g := range r.Groups {
		for _, f := range g.Files {
			out = append(out, Finding{
				Category: Duplicate, Rule: g.ID, Group: g.ID,
				Path: f.Path, Rel: f.Rel, Root: f.Root, Size: g.Size, ModTime: f.ModTime,
			})
		}
	}
	return out
}

func hashAll(files []scan.Entry, idx []int, limit int64, workers int, res *DupResult) map[int]string {
	out := make(map[int]string, len(idx))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	for _, i := range idx {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			h, n, err := hashFile(files[i].Path, limit)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				res.Errors = append(res.Errors, scan.Error{Path: files[i].Path, Err: err.Error()})
				return
			}
			out[i] = h
			res.Hashed++
			res.HashedBytes += n
		}(i)
	}
	wg.Wait()
	return out
}

// hashFile returns the SHA-256 of the first limit bytes (whole file if
// limit < 0) and how many bytes were read.
func hashFile(path string, limit int64) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	var r io.Reader = f
	if limit > 0 {
		r = io.LimitReader(f, limit)
	}
	h := sha256.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}
