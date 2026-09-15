package action

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/i18n"
)

// ManifestSchema changes when the manifest layout changes incompatibly.
const ManifestSchema = 1

// ManifestName is the file written into every quarantine batch folder.
const ManifestName = "manifest.json"

// Problem is an action that could not be carried out; apply continues.
type Problem struct {
	Path string `json:"path"`
	Err  string `json:"error"`
}

// Entry is one moved item, enough to undo it.
type Entry struct {
	Op       Op     `json:"op"`
	From     string `json:"from"`               // where it was
	To       string `json:"to"`                 // where it is now (inside the batch folder)
	Stub     string `json:"stub,omitempty"`     // stub left at From's folder, if any
	Original string `json:"original,omitempty"` // duplicates: the copy that stayed
	Category string `json:"category,omitempty"`
	Size     int64  `json:"size"`
	IsDir    bool   `json:"is_dir,omitempty"`
	Restored bool   `json:"restored,omitempty"`
}

// Manifest describes one quarantine batch: one per scanned root per apply.
type Manifest struct {
	Schema   int        `json:"schema"`
	Tool     string     `json:"tool"`
	Created  time.Time  `json:"created"`
	Plan     string     `json:"plan,omitempty"`
	Report   string     `json:"report,omitempty"`
	Root     string     `json:"root"`
	Dir      string     `json:"dir"` // batch folder: <root>/<quarantine>/<ts>
	Path     string     `json:"-"`   // where this manifest is stored
	DryRun   bool       `json:"dry_run,omitempty"`
	Purged   *time.Time `json:"purged,omitempty"` // set when the copies were deleted for good
	Entries  []Entry    `json:"entries"`
	Problems []Problem  `json:"problems"`
}

// ApplyOptions tunes Apply.
type ApplyOptions struct {
	DryRun         bool
	QuarantineDir  string            // relative to the root ("" = .folder-inspect/quarantine)
	Stubs          bool              // leave "<name>.removed.txt" files
	StubCategories map[string]bool   // categories that get a stub
	StubTexts      map[string]string // custom reason per category (overrides the built-in text)
	Videos         map[string]bool   // extensions explained as video
	Lang           string            // stub language ("" = current)
	Now            time.Time         // batch timestamp ("" = now)
	PlanPath       string            // recorded in the manifest
	// Findings by path gives stubs the rule and threshold of an oversized file.
	Findings map[string]detect.Finding
}

// ApplyResult sums up an apply run.
type ApplyResult struct {
	Manifests []*Manifest
	Moved     int
	Stubs     int
	Problems  []Problem
}

// Apply carries out a validated plan: every path moves into
// <root>/<quarantine>/<ts>/<relative path> and, for the configured
// categories, a stub explaining the removal is left in its place. Nothing
// is deleted. Duplicates are re-verified against their original first; a
// missing or changed original leaves the copy untouched. In dry-run mode
// nothing is written and the manifests describe what would happen.
func Apply(p *Plan, opt ApplyOptions) (*ApplyResult, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	if opt.QuarantineDir == "" {
		opt.QuarantineDir = ".folder-inspect/quarantine"
	}
	if opt.Lang == "" {
		opt.Lang = i18n.Lang()
	}
	if opt.Now.IsZero() {
		opt.Now = time.Now()
	}
	res := &ApplyResult{}
	byRoot := map[string][]Action{}
	for _, a := range p.Actions {
		root := p.rootOf(a.Path)
		byRoot[root] = append(byRoot[root], a)
	}
	roots := make([]string, 0, len(byRoot))
	for r := range byRoot {
		roots = append(roots, r)
	}
	sort.Strings(roots)

	for _, root := range roots {
		batch := filepath.Join(root, filepath.FromSlash(opt.QuarantineDir), opt.Now.Format("2006-01-02_150405"))
		if !opt.DryRun {
			batch = uniqueDir(batch)
		}
		m := &Manifest{
			Schema: ManifestSchema, Tool: "folder-inspect", Created: opt.Now, Plan: opt.PlanPath, Report: p.Report,
			Root: root, Dir: batch, Path: filepath.Join(batch, ManifestName), DryRun: opt.DryRun,
			Entries: []Entry{}, Problems: []Problem{},
		}
		// Deeper paths first, so a folder and a file inside it (both planned)
		// do not trip over each other: the file moves, then the folder.
		actions := byRoot[root]
		sort.Slice(actions, func(i, j int) bool { return depthOf(actions[i].Path) > depthOf(actions[j].Path) })
		for _, a := range actions {
			e, err := applyOne(a, root, batch, m, opt)
			if err != nil {
				m.Problems = append(m.Problems, Problem{Path: a.Path, Err: err.Error()})
				continue
			}
			m.Entries = append(m.Entries, e)
			res.Moved++
			if e.Stub != "" {
				res.Stubs++
			}
		}
		if !opt.DryRun && len(m.Entries) > 0 {
			if err := m.save(); err != nil {
				return res, err
			}
		}
		res.Problems = append(res.Problems, m.Problems...)
		res.Manifests = append(res.Manifests, m)
	}
	return res, nil
}

func applyOne(a Action, root, batch string, m *Manifest, opt ApplyOptions) (Entry, error) {
	st, err := os.Lstat(a.Path)
	if err != nil {
		return Entry{}, errors.New(i18n.TL(opt.Lang, "err.not_found"))
	}
	if st.Mode()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
		return Entry{}, errors.New(i18n.TL(opt.Lang, "err.is_link"))
	}
	if a.Op == OpQuarantineDuplicate || a.Op == OpQuarantineDir {
		ost, err := os.Lstat(a.Original)
		if err != nil {
			return Entry{}, errors.New(i18n.TLf(opt.Lang, "err.original_missing", a.Original))
		}
		if st.IsDir() != ost.IsDir() {
			return Entry{}, errors.New(i18n.TLf(opt.Lang, "err.original_differs", a.Original))
		}
		same, err := sameContent(a.Path, a.Original, st.IsDir())
		if err != nil {
			return Entry{}, err
		}
		if !same {
			return Entry{}, errors.New(i18n.TLf(opt.Lang, "err.original_differs", a.Original))
		}
	}

	rel, err := filepath.Rel(root, a.Path)
	if err != nil {
		return Entry{}, err
	}
	dest := filepath.Join(batch, rel)
	e := Entry{Op: a.Op, From: a.Path, To: dest, Original: a.Original, Category: a.Category, Size: a.Size, IsDir: st.IsDir()}
	if a.Size == 0 && !st.IsDir() {
		e.Size = st.Size()
	}

	wantStub := opt.Stubs && opt.StubCategories[a.Category]
	if wantStub {
		e.Stub = filepath.Join(filepath.Dir(a.Path), filepath.Base(a.Path)+".removed.txt")
	}
	if opt.DryRun {
		return e, nil
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return Entry{}, err
	}
	if _, err := os.Lstat(dest); err == nil {
		dest = uniquePath(dest)
		e.To = dest
	}
	if err := os.Rename(a.Path, dest); err != nil {
		return Entry{}, err
	}
	if wantStub {
		e.Stub = uniquePath(e.Stub)
		text := stubText(a, e, m, opt)
		if err := os.WriteFile(e.Stub, []byte(text), 0o644); err != nil {
			// the move succeeded; report the stub problem but keep the entry
			m.Problems = append(m.Problems, Problem{Path: e.Stub, Err: err.Error()})
			e.Stub = ""
		}
	}
	return e, nil
}

// sameContent re-verifies a duplicate against its original right before
// moving it: files by size and SHA-256, folders by the set of relative
// paths with sizes and hashes.
func sameContent(a, b string, isDir bool) (bool, error) {
	if !isDir {
		ha, err := fileHash(a)
		if err != nil {
			return false, err
		}
		hb, err := fileHash(b)
		if err != nil {
			return false, err
		}
		return ha == hb, nil
	}
	sa, err := dirSignature(a)
	if err != nil {
		return false, err
	}
	sb, err := dirSignature(b)
	if err != nil {
		return false, err
	}
	return sa == sb, nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func dirSignature(dir string) (string, error) {
	var parts []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		h, err := fileHash(p)
		if err != nil {
			return err
		}
		parts = append(parts, filepath.ToSlash(rel)+"\x00"+h)
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(parts)
	return strings.Join(parts, "\n"), nil
}

func (m *Manifest) save() error {
	if err := os.MkdirAll(m.Dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.Path, data, 0o644)
}

// LoadManifest reads a manifest written by Apply.
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Tool != "folder-inspect" || m.Schema == 0 {
		return nil, errors.New(path + ": not a folder-inspect quarantine manifest")
	}
	m.Path = path
	return &m, nil
}

// FindManifest resolves a manifest path from what the user typed: the
// manifest itself, a batch folder, or a scanned root (newest batch under
// the default quarantine folder).
func FindManifest(arg, quarantineDir string) (string, error) {
	if quarantineDir == "" {
		quarantineDir = ".folder-inspect/quarantine"
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return abs, nil
	}
	if _, err := os.Stat(filepath.Join(abs, ManifestName)); err == nil {
		return filepath.Join(abs, ManifestName), nil
	}
	matches, _ := filepath.Glob(filepath.Join(abs, filepath.FromSlash(quarantineDir), "*", ManifestName))
	if len(matches) == 0 {
		return "", errors.New(i18n.Tf("err.no_manifest", abs))
	}
	sort.Strings(matches) // batch folders are timestamped: last is newest
	return matches[len(matches)-1], nil
}

func (p *Plan) rootOf(path string) string {
	lp := strings.ToLower(filepath.Clean(path))
	best := ""
	for _, r := range p.Roots {
		lr := strings.ToLower(filepath.Clean(r))
		if strings.HasPrefix(lp, lr+string(filepath.Separator)) && len(lr) > len(best) {
			best = filepath.Clean(r)
		}
	}
	return best
}

func depthOf(p string) int { return strings.Count(p, string(filepath.Separator)) }

func uniquePath(path string) string {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 2; ; i++ {
		p := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Lstat(p); errors.Is(err, os.ErrNotExist) {
			return p
		}
	}
}

func uniqueDir(path string) string {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	for i := 2; ; i++ {
		p := fmt.Sprintf("%s-%d", path, i)
		if _, err := os.Lstat(p); errors.Is(err, os.ErrNotExist) {
			return p
		}
	}
}
