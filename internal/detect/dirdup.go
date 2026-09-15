package detect

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wildcar/folder-inspect/internal/scan"
)

// DupDir is a folder taking part in a duplicate group or an overlap pair.
type DupDir struct {
	Path    string    `json:"path"`
	Rel     string    `json:"rel"`
	Root    string    `json:"root"`
	ModTime time.Time `json:"mtime"`
	Size    int64     `json:"size"`  // total bytes of files inside
	Files   int       `json:"files"` // files inside, recursive
}

// DirGroup is a set of folders with identical content: same relative file
// paths, same file contents.
type DirGroup struct {
	ID        string   `json:"id"`
	Hash      string   `json:"hash"`
	Size      int64    `json:"size"`
	Files     int      `json:"files"`
	Count     int      `json:"count"`
	Wasted    int64    `json:"wasted"`
	Suggested string   `json:"suggested"` // oldest folder by mtime, pre-selected in the UI
	Dirs      []DupDir `json:"dirs"`
}

// OverlapPair is a pair of folders that share part of their content.
type OverlapPair struct {
	A           DupDir  `json:"a"`
	B           DupDir  `json:"b"`
	SharedFiles int     `json:"shared_files"`
	SharedBytes int64   `json:"shared_bytes"`
	Ratio       float64 `json:"ratio"`   // SharedBytes / size of the smaller folder
	RatioA      float64 `json:"ratio_a"` // share of A's bytes that also exist in B
	RatioB      float64 `json:"ratio_b"`
}

// DirDupOptions tunes folder comparison.
type DirDupOptions struct {
	MinOverlap      float64 // report a pair when Ratio >= MinOverlap (0 = 0.5)
	MinFiles        int     // and at least this many shared files (0 = 2)
	MaxGroupMembers int     // ignore file groups with more copies than this for overlaps (0 = 64)
}

// DirDupResult is the outcome of DuplicateDirs.
type DirDupResult struct {
	Groups   []DirGroup    `json:"groups"`
	Overlaps []OverlapPair `json:"overlaps"`
}

type dirInfo struct {
	entry  scan.Entry
	parts  []string // "<rel path inside folder>\x00<content hash>" per file
	unique bool     // holds a file without a byte-identical twin → cannot be a duplicate
	hash   string
}

// DuplicateDirs derives duplicate folders from the file duplicate groups.
// It does no I/O: every file of an identical folder pair necessarily has a
// same-size twin and was therefore hashed by Duplicates. Files below the
// duplicate min-size are treated as unique, so folders of tiny files are
// never reported.
func DuplicateDirs(res *scan.Result, dups DupResult, opt DirDupOptions) DirDupResult {
	if opt.MinOverlap <= 0 {
		opt.MinOverlap = 0.5
	}
	if opt.MinFiles <= 0 {
		opt.MinFiles = 2
	}
	if opt.MaxGroupMembers <= 0 {
		opt.MaxGroupMembers = 64
	}
	var out DirDupResult

	content := map[string]string{}
	for _, g := range dups.Groups {
		for _, f := range g.Files {
			content[f.Path] = g.Hash
		}
	}

	dirs := map[string]*dirInfo{}
	for _, d := range res.Dirs {
		dirs[d.Path] = &dirInfo{entry: d}
	}
	for _, r := range res.RootStats {
		dirs[r.Path] = &dirInfo{entry: r}
	}

	for _, f := range res.Files {
		cid := content[f.Path]
		for _, anc := range ancestors(f.Path, f.Root) {
			d := dirs[anc]
			if d == nil {
				continue
			}
			if cid == "" {
				d.unique = true
				continue
			}
			rel := filepath.ToSlash(strings.TrimPrefix(f.Path, anc+string(filepath.Separator)))
			d.parts = append(d.parts, rel+"\x00"+cid)
		}
	}

	// Full duplicates: identical signatures.
	byHash := map[string][]*dirInfo{}
	for _, d := range dirs {
		if d.unique || len(d.parts) == 0 {
			continue
		}
		sort.Strings(d.parts)
		h := sha256.New()
		for _, p := range d.parts {
			h.Write([]byte(p))
			h.Write([]byte{0})
		}
		d.hash = hex.EncodeToString(h.Sum(nil))
		byHash[d.hash] = append(byHash[d.hash], d)
	}
	for hash, members := range byHash {
		if len(members) < 2 {
			continue
		}
		if impliedByParents(members, dirs, byHash) {
			continue
		}
		g := DirGroup{ID: hash[:12], Hash: hash, Count: len(members)}
		for _, m := range members {
			g.Dirs = append(g.Dirs, toDupDir(m.entry))
		}
		sort.Slice(g.Dirs, func(a, b int) bool {
			da, db := g.Dirs[a], g.Dirs[b]
			if !da.ModTime.Equal(db.ModTime) {
				return da.ModTime.Before(db.ModTime)
			}
			return da.Path < db.Path
		})
		g.Size, g.Files = g.Dirs[0].Size, g.Dirs[0].Files
		g.Wasted = g.Size * int64(g.Count-1)
		g.Suggested = g.Dirs[0].Path
		out.Groups = append(out.Groups, g)
	}
	sort.Slice(out.Groups, func(a, b int) bool {
		if out.Groups[a].Wasted != out.Groups[b].Wasted {
			return out.Groups[a].Wasted > out.Groups[b].Wasted
		}
		return out.Groups[a].Hash < out.Groups[b].Hash
	})

	out.Overlaps = overlaps(dups, dirs, out.Groups, opt)
	return out
}

// impliedByParents reports whether every member's parent folder belongs to
// one identical-folder group as well — then this group is just a
// consequence of the parents being duplicates and is not reported.
func impliedByParents(members []*dirInfo, dirs map[string]*dirInfo, byHash map[string][]*dirInfo) bool {
	parentHash := ""
	for _, m := range members {
		p := dirs[filepath.Dir(m.entry.Path)]
		if p == nil || p.hash == "" || len(byHash[p.hash]) < 2 {
			return false
		}
		if parentHash == "" {
			parentHash = p.hash
		} else if p.hash != parentHash {
			return false
		}
	}
	return true
}

type overlapAgg struct {
	files int
	bytes int64
}

func overlaps(dups DupResult, dirs map[string]*dirInfo, groups []DirGroup, opt DirDupOptions) []OverlapPair {
	agg := map[[2]string]*overlapAgg{}
	for _, g := range dups.Groups {
		if len(g.Files) > opt.MaxGroupMembers {
			continue
		}
		cnt := map[string]int{}
		for _, f := range g.Files {
			for _, anc := range ancestors(f.Path, f.Root) {
				cnt[anc]++
			}
		}
		keys := make([]string, 0, len(cnt))
		for k := range cnt {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i := 0; i < len(keys); i++ {
			for j := i + 1; j < len(keys); j++ {
				a, b := keys[i], keys[j]
				if isUnder(a, b) || isUnder(b, a) {
					continue
				}
				m := min(cnt[a], cnt[b])
				k := [2]string{a, b}
				o := agg[k]
				if o == nil {
					o = &overlapAgg{}
					agg[k] = o
				}
				o.files += m
				o.bytes += g.Size * int64(m)
			}
		}
	}

	type cand struct {
		OverlapPair
		depth int
	}
	var cands []cand
	for k, o := range agg {
		da, db := dirs[k[0]], dirs[k[1]]
		if da == nil || db == nil {
			continue
		}
		if da.hash != "" && da.hash == db.hash { // identical folders: reported as a group
			continue
		}
		smaller := min(da.entry.Size, db.entry.Size)
		if smaller <= 0 || o.files < opt.MinFiles {
			continue
		}
		ratio := float64(o.bytes) / float64(smaller)
		if ratio < opt.MinOverlap {
			continue
		}
		ov := OverlapPair{
			A: toDupDir(da.entry), B: toDupDir(db.entry),
			SharedFiles: o.files, SharedBytes: o.bytes, Ratio: ratio,
			RatioA: float64(o.bytes) / float64(da.entry.Size),
			RatioB: float64(o.bytes) / float64(db.entry.Size),
		}
		cands = append(cands, cand{ov, depth(k[0]) + depth(k[1])})
	}
	// Most specific pairs first; a pair whose shared bytes are fully
	// explained by a deeper kept pair (or by an identical-folder group) is
	// noise and dropped.
	sort.Slice(cands, func(a, b int) bool {
		if cands[a].depth != cands[b].depth {
			return cands[a].depth > cands[b].depth
		}
		return cands[a].A.Path+cands[a].B.Path < cands[b].A.Path+cands[b].B.Path
	})
	type kept struct {
		a, b  string
		bytes int64
	}
	var keep []kept
	for _, g := range groups {
		for i := 0; i < len(g.Dirs); i++ {
			for j := i + 1; j < len(g.Dirs); j++ {
				keep = append(keep, kept{g.Dirs[i].Path, g.Dirs[j].Path, g.Size})
			}
		}
	}
	var out []OverlapPair
	for _, c := range cands {
		explained := false
		for _, k := range keep {
			if k.bytes != c.SharedBytes {
				continue
			}
			if (isUnderOrEqual(k.a, c.A.Path) && isUnderOrEqual(k.b, c.B.Path)) ||
				(isUnderOrEqual(k.b, c.A.Path) && isUnderOrEqual(k.a, c.B.Path)) {
				explained = true
				break
			}
		}
		if explained {
			continue
		}
		keep = append(keep, kept{c.A.Path, c.B.Path, c.SharedBytes})
		out = append(out, c.OverlapPair)
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].SharedBytes != out[b].SharedBytes {
			return out[a].SharedBytes > out[b].SharedBytes
		}
		return out[a].A.Path+out[a].B.Path < out[b].A.Path+out[b].B.Path
	})
	return out
}

// Findings flattens the identical-folder groups: one finding per folder.
// Overlap pairs are not findings; they are rendered from DirOverlaps.
func (r DirDupResult) Findings() []Finding {
	var out []Finding
	for _, g := range r.Groups {
		for _, d := range g.Dirs {
			out = append(out, Finding{
				Category: DirDuplicate, Rule: g.ID, Group: g.ID, IsDir: true,
				Path: d.Path, Rel: d.Rel, Root: d.Root, Size: d.Size, ModTime: d.ModTime,
			})
		}
	}
	return out
}

func toDupDir(e scan.Entry) DupDir {
	return DupDir{Path: e.Path, Rel: e.Rel, Root: e.Root, ModTime: e.ModTime, Size: e.Size, Files: e.Files}
}

// ancestors lists the folders containing path, from its parent up to and
// including root.
func ancestors(path, root string) []string {
	var out []string
	for dir := filepath.Dir(path); ; {
		out = append(out, dir)
		if dir == root {
			break
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return out
}

func isUnder(parent, p string) bool {
	return strings.HasPrefix(p, parent+string(filepath.Separator))
}

func isUnderOrEqual(p, parent string) bool { return p == parent || isUnder(parent, p) }

func depth(p string) int { return strings.Count(p, string(filepath.Separator)) }
