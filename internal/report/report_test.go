package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/scan"
)

func TestHumanSize(t *testing.T) {
	i18n.Set(i18n.EN)
	t.Cleanup(func() { i18n.Set(i18n.RU) })
	cases := map[int64]string{
		0:                               "0 B",
		512:                             "512 B",
		int64(config.KB):                "1.0 KB",
		int64(15 * config.MB):           "15.0 MB",
		int64(1.5 * float64(config.GB)): "1.5 GB",
		int64(2 * config.TB):            "2.0 TB",
	}
	for n, want := range cases {
		if got := HumanSize(n); got != want {
			t.Errorf("HumanSize(%d)=%q want %q", n, got, want)
		}
	}
	i18n.Set(i18n.RU)
	if got := HumanSize(int64(15 * config.MB)); got != "15.0 МБ" {
		t.Errorf("RU units: %q", got)
	}
}

func TestTopItems(t *testing.T) {
	entries := []scan.Entry{
		{Rel: "small", Size: 1}, {Rel: "big", Size: 100, Files: 3}, {Rel: "mid", Size: 50},
	}
	top := topItems(entries, 2)
	if len(top) != 2 || top[0].Rel != "big" || top[0].Files != 3 || top[1].Rel != "mid" {
		t.Errorf("topItems wrong: %+v", top)
	}
	if got := topItems(entries, 10); len(got) != 3 {
		t.Errorf("n larger than input must return all: %d", len(got))
	}
	if got := topItems(nil, 5); len(got) != 0 {
		t.Errorf("nil input: %+v", got)
	}
}

func TestBuildSummarisesDuplicatesAsGroups(t *testing.T) {
	res := &scan.Result{Roots: []string{"R"}, Started: time.Now(), Finished: time.Now()}
	dups := detect.DupResult{
		Groups: []detect.DupGroup{
			{ID: "aaa", Size: 100, Count: 3, Wasted: 200, Files: make([]detect.DupFile, 3)},
			{ID: "bbb", Size: 10, Count: 2, Wasted: 10, Files: make([]detect.DupFile, 2)},
		},
		Errors: []scan.Error{{Path: "x", Err: "denied"}},
		Hashed: 5, HashedBytes: 500,
	}
	dirs := detect.DirDupResult{
		Groups:   []detect.DirGroup{{ID: "d1", Size: 1000, Count: 2, Wasted: 1000, Dirs: make([]detect.DupDir, 2)}},
		Overlaps: []detect.OverlapPair{{SharedBytes: 300, SharedFiles: 2, Ratio: 0.6}, {SharedBytes: 200, SharedFiles: 2, Ratio: 0.5}},
	}
	names := detect.NameResult{Groups: []detect.NameGroup{{ID: "n1", Name: "x.docx", Count: 2, Variants: 1, Size: 40, Files: make([]detect.NameFile, 2)}}}
	findings := append([]detect.Finding{{Category: detect.Junk, Size: 7}}, dups.Findings()...)
	findings = append(findings, dirs.Findings()...)
	findings = append(findings, names.Findings()...)
	r := Build(res, findings, dups, dirs, names, config.Default(), "t")
	if len(r.Summary) != 5 {
		t.Fatalf("summary: %+v", r.Summary)
	}
	if r.Summary[3].Category != detect.SimilarName || r.Summary[3].Count != 1 || r.Summary[3].Size != 40 {
		t.Errorf("similar-name summary must be groups/variant size: %+v", r.Summary[3])
	}
	if r.Summary[0].Category != detect.Duplicate || r.Summary[0].Count != 2 || r.Summary[0].Size != 210 {
		t.Errorf("duplicate summary must be groups/wasted: %+v", r.Summary[0])
	}
	if r.Summary[1].Category != detect.DirDuplicate || r.Summary[1].Count != 1 || r.Summary[1].Size != 1000 {
		t.Errorf("folder duplicate summary must be groups/wasted: %+v", r.Summary[1])
	}
	if r.Summary[2].Category != detect.DirOverlap || r.Summary[2].Count != 2 || r.Summary[2].Size != 500 {
		t.Errorf("overlap summary must be pairs/shared: %+v", r.Summary[2])
	}
	if r.Summary[4].Category != detect.Junk || r.Summary[4].Count != 1 {
		t.Errorf("junk summary: %+v", r.Summary[4])
	}
	if Percent(0.896) != "90%" || Percent(1) != "100%" {
		t.Error("Percent rounding")
	}
	if r.Stats.Errors != 1 || r.Stats.Hashed != 5 || r.Stats.HashedBytes != 500 || len(r.Duplicates) != 2 {
		t.Errorf("stats/dups not carried: %+v", r.Stats)
	}
}

func TestCheckOverwriteAndNames(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "report.json")
	if err := CheckOverwrite(p, false); err != nil {
		t.Errorf("missing file must be writable: %v", err)
	}
	os.WriteFile(p, []byte("{}"), 0o644)
	if err := CheckOverwrite(p, false); err == nil || !strings.Contains(err.Error(), "-force") {
		t.Errorf("existing file must be refused with a -force hint, got %v", err)
	}
	if err := CheckOverwrite(p, true); err != nil {
		t.Errorf("force must allow: %v", err)
	}
	ts := time.Date(2026, 9, 15, 14, 3, 9, 0, time.Local)
	if got := DefaultName(ts); got != "report-2026-09-15_140309.json" {
		t.Errorf("DefaultName: %q", got)
	}
	if got := SiblingPath(`C:\x\report-1.json`, "xlsx"); got != `C:\x\report-1.xlsx` {
		t.Errorf("SiblingPath: %q", got)
	}
	if _, err := ReadJSON(p); err == nil {
		t.Error("a foreign JSON must be rejected")
	}
}

func TestDefaultDirAndNewest(t *testing.T) {
	root := t.TempDir()
	dir := DefaultDir(root)
	if !strings.HasSuffix(dir, filepath.Join(".folder-inspect", "reports")) || !strings.HasPrefix(dir, root) {
		t.Errorf("DefaultDir: %q", dir)
	}
	if _, err := NewestReport(dir); err == nil {
		t.Error("no reports yet must be an error")
	}
	os.MkdirAll(dir, 0o755)
	old := filepath.Join(dir, "report-2026-01-01_000000.json")
	newer := filepath.Join(dir, "report-2026-02-01_000000.json")
	os.WriteFile(old, []byte("{}"), 0o644)
	os.WriteFile(newer, []byte("{}"), 0o644)
	os.Chtimes(old, time.Now().Add(-time.Hour), time.Now().Add(-time.Hour))
	if got, err := NewestReport(dir); err != nil || got != newer {
		t.Errorf("NewestReport = %q, %v", got, err)
	}
	if got := UniquePath(newer); got != strings.TrimSuffix(newer, ".json")+"-2.json" {
		t.Errorf("UniquePath on an existing file: %q", got)
	}
	os.WriteFile(strings.TrimSuffix(newer, ".json")+"-2.json", []byte("{}"), 0o644)
	if got := UniquePath(newer); !strings.HasSuffix(got, "-3.json") {
		t.Errorf("UniquePath must skip taken suffixes: %q", got)
	}
	fresh := filepath.Join(dir, "report-2026-03-01_000000.json")
	if got := UniquePath(fresh); got != fresh {
		t.Errorf("UniquePath on a free name must return it unchanged: %q", got)
	}
}
