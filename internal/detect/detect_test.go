package detect

import (
	"path/filepath"
	"testing"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/scan"
)

func file(rel string, size int64) scan.Entry {
	return scan.Entry{Path: filepath.Join("R", filepath.FromSlash(rel)), Rel: rel, Root: "R", Name: filepath.Base(rel), Size: size}
}

func dir(rel string, size int64, files, entries int) scan.Entry {
	e := file(rel, size)
	e.IsDir, e.Files, e.Entries = true, files, entries
	return e
}

func TestOversizedPicksMostSpecificRule(t *testing.T) {
	rules := config.Default().SizeRules
	files := []scan.Entry{
		file("a/small.docx", 1*int64(config.MB)),
		file("a/big.docx", 16*int64(config.MB)),
		file("a/huge.docx", 200*int64(config.MB)),
		file("v/meeting.MP4", 101*int64(config.MB)),
		file("v/short.mp4", 99*int64(config.MB)),
		file("p/photo.JPG", 6*int64(config.MB)),
		file("p/deck.pptx", 15*int64(config.MB)), // exactly at threshold: not over
	}
	got := Oversized(files, rules)
	want := map[string]string{"a/big.docx": "documents", "a/huge.docx": "documents", "v/meeting.MP4": "huge", "p/photo.JPG": "images"}
	if len(got) != len(want) {
		t.Fatalf("got %d findings, want %d: %+v", len(got), len(want), got)
	}
	for _, f := range got {
		if want[f.Rel] != f.Rule {
			t.Errorf("%s: rule %q want %q", f.Rel, f.Rule, want[f.Rel])
		}
		if f.Category != Oversize || f.Threshold <= 0 || f.Size <= f.Threshold {
			t.Errorf("%s: bad finding %+v", f.Rel, f)
		}
	}
}

func TestOversizedFallsBackToGenericWhenSpecificIsLarger(t *testing.T) {
	rules := []config.SizeRule{
		{Name: "any", Extensions: []string{"*"}, Threshold: 100 * config.MB},
		{Name: "docs", Extensions: []string{"docx"}, Threshold: 500 * config.MB},
	}
	got := Oversized([]scan.Entry{file("x.docx", 150*int64(config.MB))}, rules)
	if len(got) != 1 || got[0].Rule != "any" {
		t.Errorf("want fallback to generic rule, got %+v", got)
	}
}

func TestByExtension(t *testing.T) {
	cfg := config.Default()
	files := []scan.Entry{
		file("old/backup 2024.ZIP", 1), file("dist/setup.exe", 1), file("doc/report.docx", 1),
		file("dist/app.tar.gz", 1), file("lib/app.jar", 1),
	}
	arch := ByExtension(files, cfg.Archives, Archive)
	if len(arch) != 2 || arch[0].Rule != "zip" || arch[1].Rule != "gz" {
		t.Errorf("archives: %+v", arch)
	}
	dist := ByExtension(files, cfg.Distributives, Distributive)
	if len(dist) != 1 || dist[0].Rel != "dist/setup.exe" {
		t.Errorf("distributives: %+v", dist)
	}
}

func TestJunkFilesCollapsesFolders(t *testing.T) {
	res := &scan.Result{
		Dirs: []scan.Entry{
			dir("p", 100, 3, 3),
			dir("p/.Trashes", 50, 2, 2),
			dir("p/.Trashes/sub", 20, 1, 1),
		},
		Files: []scan.Entry{
			file("p/Thumbs.db", 5), file("p/~$Отчёт.docx", 1), file("p/Отчёт.docx", 30),
			file("p/.Trashes/a.tmp", 30), file("p/.Trashes/sub/b.bak", 20), file("p/notes.bak.txt", 1),
		},
	}
	got := JunkFiles(res, config.Default().Junk)
	rels := map[string]Finding{}
	for _, f := range got {
		rels[f.Rel] = f
	}
	if len(got) != 3 {
		t.Fatalf("want 3 findings (dir + 2 files), got %d: %+v", len(got), got)
	}
	if d, ok := rels["p/.Trashes"]; !ok || !d.IsDir || d.Size != 50 || d.Rule != ".Trashes" {
		t.Errorf("junk dir finding wrong: %+v", d)
	}
	if _, ok := rels["p/.Trashes/a.tmp"]; ok {
		t.Error("files inside a junk folder must not be reported separately")
	}
	if _, ok := rels["p/Thumbs.db"]; !ok {
		t.Error("Thumbs.db missing")
	}
	if _, ok := rels["p/~$Отчёт.docx"]; !ok {
		t.Error("office lock file missing")
	}
}

func TestEmptyEntries(t *testing.T) {
	res := &scan.Result{
		Dirs: []scan.Entry{
			dir("full", 10, 1, 1),
			dir("e1", 0, 0, 1),
			dir("e1/e2", 0, 0, 0),
			dir("e3", 0, 0, 0),
		},
		Files: []scan.Entry{file("full/a.txt", 10), file("full/zero.txt", 0)},
	}
	got := EmptyEntries(res)
	if len(got) != 3 {
		t.Fatalf("want 3 findings, got %+v", got)
	}
	seen := map[string]Finding{}
	for _, f := range got {
		seen[f.Rel] = f
	}
	if seen["e1"].Detail != "no-files" || seen["e3"].Detail != "empty" {
		t.Errorf("details: %+v", seen)
	}
	if _, ok := seen["e1/e2"]; ok {
		t.Error("nested empty dir must be collapsed into its parent")
	}
	if seen["full/zero.txt"].Category != EmptyFile {
		t.Error("zero-size file missing")
	}
}

func TestSortOrder(t *testing.T) {
	fs := []Finding{
		{Category: Junk, Size: 1, Path: "b"},
		{Category: Oversize, Size: 1, Path: "z"},
		{Category: Oversize, Size: 9, Path: "a"},
		{Category: Junk, Size: 1, Path: "a"},
	}
	Sort(fs)
	want := []string{"a", "z", "a", "b"}
	for i, f := range fs {
		if f.Path != want[i] {
			t.Fatalf("order %v", fs)
		}
	}
}
