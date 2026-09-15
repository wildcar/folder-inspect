package detect

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/wildcar/folder-inspect/internal/scan"
)

// layout:
//
//	A/Приложения/{1.pdf,2.pdf}            identical to
//	B/Старое/Приложения (копия)/{1.pdf,2.pdf}   → full duplicate group
//	B/Для отправки/{1.pdf,2.pdf,Письмо.docx}    → contains all of Приложения → overlap, ratio 1.0
//	A/Договоры/Договор.docx  ≡  B/Старое/Договор (2).docx   (single file: not enough for a pair)
//	A/Уникальное/readme.txt  unique
func buildDirFixture(t *testing.T) (*scan.Result, DupResult) {
	root := t.TempDir()
	t0 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.Local)
	p1 := bytes.Repeat([]byte("p1"), 3000)
	p2 := bytes.Repeat([]byte("p2"), 3500)
	letter := bytes.Repeat([]byte("l"), 1500)
	contract := bytes.Repeat([]byte("c"), 2000)

	write(t, root, "A/Приложения/1.pdf", p1, t0)
	write(t, root, "A/Приложения/2.pdf", p2, t0)
	write(t, root, "B/Старое/Приложения (копия)/1.pdf", p1, t0.Add(48*time.Hour))
	write(t, root, "B/Старое/Приложения (копия)/2.pdf", p2, t0.Add(48*time.Hour))
	write(t, root, "B/Для отправки/1.pdf", p1, t0.Add(time.Hour))
	write(t, root, "B/Для отправки/2.pdf", p2, t0.Add(time.Hour))
	write(t, root, "B/Для отправки/Письмо.docx", letter, t0)
	write(t, root, "A/Договоры/Договор.docx", contract, t0)
	write(t, root, "B/Старое/Договор (2).docx", contract, t0)
	write(t, root, "A/Уникальное/readme.txt", []byte("unique content here"), t0)

	res, err := scan.Walk([]string{root}, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	dups := Duplicates(res.Files, DupOptions{MinSize: 1})
	return res, dups
}

func TestDuplicateDirsFull(t *testing.T) {
	res, dups := buildDirFixture(t)
	r := DuplicateDirs(res, dups, DirDupOptions{})
	if len(r.Groups) != 1 {
		t.Fatalf("want 1 identical-folder group, got %d: %+v", len(r.Groups), r.Groups)
	}
	g := r.Groups[0]
	if g.Count != 2 || g.Files != 2 || g.Size != 13000 || g.Wasted != 13000 {
		t.Errorf("group metrics wrong: %+v", g)
	}
	if filepath.Base(g.Suggested) != "Приложения" {
		t.Errorf("suggested must be the oldest folder: %s", g.Suggested)
	}
	names := map[string]bool{}
	for _, d := range g.Dirs {
		names[filepath.Base(d.Path)] = true
	}
	if !names["Приложения"] || !names["Приложения (копия)"] {
		t.Errorf("members: %+v", g.Dirs)
	}
	fs := r.Findings()
	if len(fs) != 2 || !fs[0].IsDir || fs[0].Category != DirDuplicate || fs[0].Group != g.ID {
		t.Errorf("findings: %+v", fs)
	}
}

func TestDuplicateDirsOverlaps(t *testing.T) {
	res, dups := buildDirFixture(t)
	r := DuplicateDirs(res, dups, DirDupOptions{})
	type pair struct{ a, b string }
	got := map[pair]OverlapPair{}
	for _, o := range r.Overlaps {
		got[pair{filepath.Base(o.A.Path), filepath.Base(o.B.Path)}] = o
		got[pair{filepath.Base(o.B.Path), filepath.Base(o.A.Path)}] = o
	}
	o, ok := got[pair{"Приложения", "Для отправки"}]
	if !ok {
		t.Fatalf("expected overlap Приложения ~ Для отправки, got %+v", r.Overlaps)
	}
	if o.SharedFiles != 2 || o.SharedBytes != 13000 || o.Ratio != 1.0 {
		t.Errorf("overlap metrics: %+v", o)
	}
	// Для отправки (14500 B) holds the 13000 shared bytes → RatioB ≈ 0.897
	small, big := o.RatioA, o.RatioB
	if o.A.Files == 3 {
		small, big = big, small
	}
	if small != 1.0 || big < 0.89 || big > 0.9 {
		t.Errorf("per-folder ratios: %+v", o)
	}
	if _, ok := got[pair{"Приложения (копия)", "Для отправки"}]; !ok {
		t.Error("the copy must overlap with Для отправки as well")
	}
	// Identical folders are a group, never an overlap pair.
	if _, ok := got[pair{"Приложения", "Приложения (копия)"}]; ok {
		t.Error("identical folders must not be reported as an overlap")
	}
	// A ~ B/Старое share 13000 (explained by the identical group) + 2000 (contract):
	// 15000 / min(21000, 15000) = 1.0 → reported, because the contract is extra.
	if _, ok := got[pair{"A", "Старое"}]; !ok {
		t.Errorf("A ~ Старое should be reported (extra shared contract), got %+v", r.Overlaps)
	}
	// but A ~ B is only the same content again at a higher level: 15000 vs min(21000, 29500)=21000 → 0.71,
	// bytes equal to the kept A~Старое pair → explained → dropped.
	if _, ok := got[pair{"A", "B"}]; ok {
		t.Error("A ~ B is fully explained by A ~ Старое and must be dropped")
	}
	for _, ov := range r.Overlaps {
		if isUnder(ov.A.Path, ov.B.Path) || isUnder(ov.B.Path, ov.A.Path) {
			t.Errorf("ancestor pairs must never be reported: %+v", ov)
		}
	}
}

func TestDuplicateDirsThresholds(t *testing.T) {
	res, dups := buildDirFixture(t)
	r := DuplicateDirs(res, dups, DirDupOptions{MinFiles: 3})
	// only A ~ Старое shares 3 files (two attachments + the contract)
	if len(r.Overlaps) != 1 || r.Overlaps[0].SharedFiles != 3 {
		t.Errorf("min_files 3 must leave only the 3-file pair: %+v", r.Overlaps)
	}
	if r := DuplicateDirs(res, dups, DirDupOptions{MinFiles: 4}); len(r.Overlaps) != 0 {
		t.Errorf("min_files 4 must suppress every pair: %+v", r.Overlaps)
	}
	r = DuplicateDirs(res, DupResult{}, DirDupOptions{})
	if len(r.Groups) != 0 || len(r.Overlaps) != 0 {
		t.Errorf("no file duplicates → no folder results: %+v", r)
	}
}

func TestAncestors(t *testing.T) {
	root := filepath.Join("R", "root")
	got := ancestors(filepath.Join(root, "a", "b", "f.txt"), root)
	want := []string{filepath.Join(root, "a", "b"), filepath.Join(root, "a"), root}
	if len(got) != len(want) {
		t.Fatalf("ancestors: %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ancestors[%d] = %q want %q", i, got[i], want[i])
		}
	}
}
