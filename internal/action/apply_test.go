package action

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/i18n"
)

func mk(t *testing.T, root, rel string, content string) string {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

func defaultOpts(now time.Time) ApplyOptions {
	return ApplyOptions{
		Stubs: true, Lang: i18n.RU, Now: now,
		StubCategories: map[string]bool{"oversize": true, "archive": true, "distributive": true, "duplicate": true, "dir-duplicate": true},
		Videos:         map[string]bool{"mp4": true},
	}
}

func TestApplyAndRestore(t *testing.T) {
	root := t.TempDir()
	in := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	video := mk(t, root, "Проект A/Записи/Встреча.mp4", strings.Repeat("v", 5000))
	archive := mk(t, root, "Проект B/Старое/Архив 2024.zip", "zip")
	junk := mk(t, root, "Проект A/Thumbs.db", "thumbs")
	orig := mk(t, root, "Проект A/Договоры/Договор.docx", "contract-content")
	copy1 := mk(t, root, "Проект B/Копия Договор.docx", "contract-content")
	mk(t, root, "Проект A/Приложения/1.pdf", "p1")
	mk(t, root, "Проект A/Приложения/2.pdf", "p2")
	mk(t, root, "Проект B/Старое/Приложения (копия)/1.pdf", "p1")
	mk(t, root, "Проект B/Старое/Приложения (копия)/2.pdf", "p2")
	copyDir := in("Проект B/Старое/Приложения (копия)")
	origDir := in("Проект A/Приложения")
	changed := mk(t, root, "Проект B/Изменённый.docx", "contract-content-CHANGED")

	p := New("r.json", []string{root})
	p.Actions = []Action{
		{Op: OpQuarantine, Path: video, Category: "oversize", Size: 5000},
		{Op: OpQuarantine, Path: archive, Category: "archive", Size: 3},
		{Op: OpQuarantine, Path: junk, Category: "junk", Size: 6},
		{Op: OpQuarantineDuplicate, Path: copy1, Original: orig, Group: "g1", Category: "duplicate", Size: 16},
		{Op: OpQuarantineDir, Path: copyDir, Original: origDir, Group: "d1", Category: "dir-duplicate", Size: 4, IsDir: true},
		{Op: OpQuarantineDuplicate, Path: changed, Original: orig, Group: "g1", Category: "duplicate", Size: 24}, // content differs → must stay
		{Op: OpQuarantine, Path: in("Проект A/нет такого.txt"), Category: "junk"},                                // missing → problem
	}
	now := time.Date(2026, 9, 15, 16, 0, 0, 0, time.Local)
	opt := defaultOpts(now)
	opt.Findings = map[string]detect.Finding{video: {Threshold: 100 << 20}}

	// dry run: nothing moves, nothing written
	opt.DryRun = true
	dry, err := Apply(p, opt)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Moved != 5 || len(dry.Problems) != 2 || len(dry.Manifests) != 1 || len(dry.Manifests[0].Entries) != 5 {
		t.Errorf("dry run: moved=%d problems=%d manifests=%+v", dry.Moved, len(dry.Problems), dry.Manifests)
	}
	if !exists(video) || exists(dry.Manifests[0].Path) || exists(in(".folder-inspect")) {
		t.Error("dry run must not touch the disk")
	}

	// real run
	opt.DryRun = false
	res, err := Apply(p, opt)
	if err != nil {
		t.Fatal(err)
	}
	if res.Moved != 5 || res.Stubs != 4 || len(res.Problems) != 2 {
		t.Fatalf("apply: moved=%d stubs=%d problems=%+v", res.Moved, res.Stubs, res.Problems)
	}
	m := res.Manifests[0]
	batch := filepath.Join(root, ".folder-inspect", "quarantine", "2026-09-15_160000")
	if m.Dir != batch || !exists(m.Path) {
		t.Errorf("batch dir/manifest: %s %s", m.Dir, m.Path)
	}
	// moved with the relative path preserved
	if exists(video) || !exists(filepath.Join(batch, "Проект A", "Записи", "Встреча.mp4")) {
		t.Error("video not moved into the batch folder")
	}
	if exists(copyDir) || !exists(filepath.Join(batch, "Проект B", "Старое", "Приложения (копия)", "2.pdf")) {
		t.Error("duplicate folder not moved")
	}
	if !exists(orig) || !exists(origDir) || !exists(changed) {
		t.Error("originals and the changed copy must stay")
	}
	// stubs: named after the file, none for junk
	stub := video + ".removed.txt"
	body, err := os.ReadFile(stub)
	if err != nil {
		t.Fatalf("video stub missing: %v", err)
	}
	s := string(body)
	for _, want := range []string{"«Встреча.mp4»", "2026-09-15 16:00", "видео", "4.9 КБ", "Карантин:", "restore", ".folder-inspect"} {
		if !strings.Contains(s, want) {
			t.Errorf("video stub lacks %q:\n%s", want, s)
		}
	}
	if exists(junk + ".removed.txt") {
		t.Error("junk must not get a stub")
	}
	dupStub, _ := os.ReadFile(copy1 + ".removed.txt")
	if !strings.Contains(string(dupStub), "Оригинал") || !strings.Contains(string(dupStub), filepath.Join("..", "Проект A", "Договоры", "Договор.docx")) {
		t.Errorf("duplicate stub must name the original by relative path:\n%s", dupStub)
	}
	dirStub, _ := os.ReadFile(copyDir + ".removed.txt")
	if !strings.Contains(string(dirStub), "Папка «Приложения (копия)»") || !strings.Contains(string(dirStub), "полностью совпадает") {
		t.Errorf("folder stub wrong:\n%s", dirStub)
	}
	archStub, _ := os.ReadFile(archive + ".removed.txt")
	if !strings.Contains(string(archStub), "архив") || !strings.Contains(string(archStub), "истории изменений") {
		t.Errorf("archive stub wrong:\n%s", archStub)
	}
	// problems name the reasons
	joined := res.Problems[0].Err + res.Problems[1].Err
	if !strings.Contains(joined, "не найден") || !strings.Contains(joined, "отличается") {
		t.Errorf("problems: %+v", res.Problems)
	}

	// second apply in the same second gets its own batch folder
	res2, err := Apply(&Plan{Schema: PlanSchema, Tool: "folder-inspect", Roots: []string{root},
		Actions: []Action{{Op: OpQuarantine, Path: junk + ".x", Category: "junk"}}}, opt)
	if err != nil || len(res2.Manifests) != 1 || res2.Manifests[0].Dir == batch {
		t.Errorf("second batch must not reuse the folder: %v %+v", err, res2.Manifests)
	}

	// restore: dry, then real
	loaded, err := LoadManifest(m.Path)
	if err != nil {
		t.Fatal(err)
	}
	rd, _ := Restore(loaded, RestoreOptions{DryRun: true})
	if rd.Restored != 5 || exists(video) {
		t.Errorf("restore dry run: %+v", rd)
	}
	rr, err := Restore(loaded, RestoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if rr.Restored != 5 || len(rr.Problems) != 0 {
		t.Errorf("restore: %+v", rr)
	}
	for _, p := range []string{video, archive, junk, copy1, filepath.Join(copyDir, "1.pdf")} {
		if !exists(p) {
			t.Errorf("%s not restored", p)
		}
	}
	if exists(stub) || exists(copy1+".removed.txt") || exists(copyDir+".removed.txt") {
		t.Error("stubs must be removed on restore")
	}
	again, _ := LoadManifest(m.Path)
	for _, e := range again.Entries {
		if !e.Restored {
			t.Errorf("entry not marked restored: %+v", e)
		}
	}
	rr2, _ := Restore(again, RestoreOptions{})
	if rr2.Restored != 0 {
		t.Error("second restore must be a no-op")
	}
	if !exists(m.Path) {
		t.Error("manifest must stay as the record")
	}
}

func TestRestoreRefusesOccupiedPath(t *testing.T) {
	root := t.TempDir()
	f := mk(t, root, "a/x.zip", "zip")
	p := New("r.json", []string{root})
	p.Actions = []Action{{Op: OpQuarantine, Path: f, Category: "archive", Size: 3}}
	res, err := Apply(p, defaultOpts(time.Now()))
	if err != nil || res.Moved != 1 {
		t.Fatal(err, res)
	}
	mk(t, root, "a/x.zip", "new file at the old place")
	rr, _ := Restore(res.Manifests[0], RestoreOptions{})
	if rr.Restored != 0 || len(rr.Problems) != 1 {
		t.Errorf("occupied path must be a problem: %+v", rr)
	}
	if b, _ := os.ReadFile(f); string(b) != "new file at the old place" {
		t.Error("the new file must not be overwritten")
	}
}

func TestFindManifest(t *testing.T) {
	root := t.TempDir()
	f := mk(t, root, "x.zip", "zip")
	p := New("r.json", []string{root})
	p.Actions = []Action{{Op: OpQuarantine, Path: f, Category: "archive"}}
	res, err := Apply(p, defaultOpts(time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)))
	if err != nil {
		t.Fatal(err)
	}
	want := res.Manifests[0].Path
	for _, arg := range []string{want, filepath.Dir(want), root} {
		got, err := FindManifest(arg, "")
		if err != nil || got != want {
			t.Errorf("FindManifest(%q) = %q, %v", arg, got, err)
		}
	}
	if _, err := FindManifest(t.TempDir(), ""); err == nil {
		t.Error("root without quarantine must fail")
	}
}

func TestCustomStubText(t *testing.T) {
	root := t.TempDir()
	f := mk(t, root, "setup.exe", "exe")
	p := New("r.json", []string{root})
	p.Actions = []Action{{Op: OpQuarantine, Path: f, Category: "distributive", Size: 3}}
	opt := defaultOpts(time.Now())
	opt.Lang = i18n.EN
	opt.StubTexts = map[string]string{"distributive": "Distributives were removed due to inappropriate place to store."}
	if _, err := Apply(p, opt); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(f + ".removed.txt")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.HasPrefix(s, `File "setup.exe" was moved to quarantine`) || !strings.Contains(s, "inappropriate place") || !strings.Contains(s, "Bring it back") {
		t.Errorf("custom EN stub:\n%s", s)
	}
}
