package detect

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wildcar/folder-inspect/internal/scan"
)

func write(t *testing.T, root, rel string, content []byte, mtime time.Time) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func TestDuplicates(t *testing.T) {
	root := t.TempDir()
	t0 := time.Date(2026, 1, 10, 12, 0, 0, 0, time.Local)
	small := bytes.Repeat([]byte("a"), 2048)
	other := bytes.Repeat([]byte("b"), 2048)
	big := bytes.Repeat([]byte("xyz"), 70000) // 210 000 B > head size: forces the full pass
	bigTail := append(bytes.Clone(big[:len(big)-1]), 'Q')

	write(t, root, "p/a.docx", small, t0.Add(2*time.Hour)) // copy
	write(t, root, "q/b.docx", small, t0)                  // oldest → suggested
	write(t, root, "r/c.docx", small, t0.Add(time.Hour))   // copy
	write(t, root, "p/d.docx", other, t0)                  // same size, different content
	write(t, root, "p/tiny.txt", []byte("hi"), t0)         // below min size
	write(t, root, "p/tiny2.txt", []byte("hi"), t0)
	write(t, root, "v/big1.bin", big, t0.Add(time.Hour))
	write(t, root, "v/big2.bin", big, t0)
	write(t, root, "v/big3.bin", bigTail, t0) // same head, different tail

	res, err := scan.Walk([]string{root}, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	dr := Duplicates(res.Files, DupOptions{MinSize: 1024, HeadSize: 64 << 10, Workers: 2})
	if len(dr.Errors) != 0 {
		t.Fatalf("errors: %+v", dr.Errors)
	}
	if len(dr.Groups) != 2 {
		t.Fatalf("want 2 groups, got %d: %+v", len(dr.Groups), dr.Groups)
	}
	// sorted by wasted size: the big pair (210 000) before the small triple (4096)
	g0, g1 := dr.Groups[0], dr.Groups[1]
	if g0.Size != int64(len(big)) || g0.Count != 2 || g0.Wasted != int64(len(big)) {
		t.Errorf("big group wrong: %+v", g0)
	}
	if filepath.Base(g0.Suggested) != "big2.bin" {
		t.Errorf("suggested should be the oldest: %s", g0.Suggested)
	}
	if g1.Size != 2048 || g1.Count != 3 || g1.Wasted != 4096 || filepath.Base(g1.Suggested) != "b.docx" {
		t.Errorf("small group wrong: %+v", g1)
	}
	if len(g0.ID) != 12 || g0.Hash[:12] != g0.ID {
		t.Errorf("id must be the hash prefix: %+v", g0)
	}
	// head pass: 4 small + 3 big candidates; full pass: 3 big files
	if dr.Hashed != 10 {
		t.Errorf("hashed %d files, want 10", dr.Hashed)
	}
	wantBytes := int64(4*2048 + 3*(64<<10) + 3*len(big))
	if dr.HashedBytes != wantBytes {
		t.Errorf("hashed %d bytes, want %d", dr.HashedBytes, wantBytes)
	}

	fs := dr.Findings()
	if len(fs) != 5 {
		t.Fatalf("want 5 duplicate findings, got %d", len(fs))
	}
	for _, f := range fs {
		if f.Category != Duplicate || f.Group == "" || f.Rule != f.Group {
			t.Errorf("bad finding %+v", f)
		}
	}
}

func TestDuplicatesNoCandidates(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", []byte("aaaa"), time.Now())
	write(t, root, "b.txt", []byte("bbbbb"), time.Now())
	res, _ := scan.Walk([]string{root}, scan.Options{})
	dr := Duplicates(res.Files, DupOptions{})
	if len(dr.Groups) != 0 || dr.Hashed != 0 {
		t.Errorf("nothing should be hashed: %+v", dr)
	}
}

func TestDuplicatesUnreadable(t *testing.T) {
	files := []scan.Entry{
		{Path: filepath.Join(t.TempDir(), "missing1"), Size: 5000},
		{Path: filepath.Join(t.TempDir(), "missing2"), Size: 5000},
	}
	dr := Duplicates(files, DupOptions{})
	if len(dr.Errors) != 2 || len(dr.Groups) != 0 {
		t.Errorf("missing files must be reported as errors: %+v", dr)
	}
}
