package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mk(t *testing.T, root, rel string, size int) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestWalkIndexesAndAggregates(t *testing.T) {
	root := t.TempDir()
	mk(t, root, "a/one.txt", 10)
	mk(t, root, "a/b/two.txt", 20)
	mk(t, root, "c/three.txt", 5)
	os.MkdirAll(filepath.Join(root, "empty"), 0o755)
	mk(t, root, "$RECYCLE.BIN/junk.tmp", 99)
	mk(t, root, WorkDir+"/quarantine/x.txt", 99)
	mk(t, root, "skipme/skipped.txt", 7)
	mk(t, root, "a/skip.log", 3)

	res, err := Walk([]string{root}, Options{Exclude: []string{"skipme", "*.log"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 3 {
		t.Fatalf("want 3 files, got %d: %+v", len(res.Files), res.Files)
	}
	if got := res.TotalSize(); got != 35 {
		t.Errorf("total size %d want 35", got)
	}
	if len(res.Skipped) != 2 {
		t.Errorf("want 2 skipped system dirs, got %v", res.Skipped)
	}
	byRel := map[string]Entry{}
	for _, d := range res.Dirs {
		byRel[d.Rel] = d
	}
	if a := byRel["a"]; a.Size != 30 || a.Files != 2 || a.Entries != 2 {
		t.Errorf("dir a aggregate wrong: %+v", a)
	}
	if e := byRel["empty"]; e.Files != 0 || e.Entries != 0 {
		t.Errorf("empty dir wrong: %+v", e)
	}
	if _, ok := byRel["skipme"]; ok {
		t.Error("excluded dir must not be indexed")
	}
	if res.RootStats[0].Files != 3 {
		t.Errorf("root stats: %+v", res.RootStats[0])
	}
	for _, f := range res.Files {
		if strings.Contains(f.Rel, "\\") {
			t.Errorf("rel must be slash-separated: %q", f.Rel)
		}
	}
}

func TestWalkDoesNotFollowSymlinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	mk(t, outside, "secret.txt", 1)
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("symlinks not permitted here:", err)
	}
	res, err := Walk([]string{root}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != 0 || len(res.Skipped) != 1 {
		t.Errorf("symlink must be skipped, got files=%d skipped=%v", len(res.Files), res.Skipped)
	}
}

func TestWalkErrors(t *testing.T) {
	if _, err := Walk(nil, Options{}); err == nil {
		t.Error("no roots must fail")
	}
	if _, err := Walk([]string{filepath.Join(t.TempDir(), "missing")}, Options{}); err == nil {
		t.Error("missing root must fail")
	}
	f := filepath.Join(t.TempDir(), "file.txt")
	os.WriteFile(f, nil, 0o644)
	if _, err := Walk([]string{f}, Options{}); err == nil {
		t.Error("file root must fail")
	}
}
