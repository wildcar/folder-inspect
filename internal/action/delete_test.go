package action

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/i18n"
)

// TestDeleteCategories: with the default config junk and empty items are
// deleted outright (no quarantine copy, no stub) and recorded in the
// manifest; empty ones come back on restore, junk does not. An "empty"
// folder that is not empty any more is quarantined instead.
func TestDeleteCategories(t *testing.T) {
	root := t.TempDir()
	in := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	junk := mk(t, root, "Проект A/Thumbs.db", "thumbs")
	emptyFile := mk(t, root, "Проект A/пустой.txt", "")
	emptyDir := in("Проект A/Пустая")
	if err := os.MkdirAll(filepath.Join(emptyDir, "внутри тоже пусто"), 0o755); err != nil {
		t.Fatal(err)
	}
	notEmpty := in("Проект B/Была пустая")
	mk(t, root, "Проект B/Была пустая/новый.docx", "content") // filled after the scan
	archive := mk(t, root, "Проект B/old.zip", "zip")

	p := New("r.json", []string{root})
	p.Actions = []Action{
		{Op: OpQuarantine, Path: junk, Category: "junk", Size: 6},
		{Op: OpQuarantine, Path: emptyFile, Category: "empty-file"},
		{Op: OpQuarantine, Path: emptyDir, Category: "empty-dir", IsDir: true},
		{Op: OpQuarantine, Path: notEmpty, Category: "empty-dir", IsDir: true},
		{Op: OpQuarantine, Path: archive, Category: "archive", Size: 3},
	}
	now := time.Date(2026, 9, 15, 17, 0, 0, 0, time.Local)
	opt := OptionsFromConfig(config.Default(), i18n.RU, "", nil)
	opt.Now = now

	opt.DryRun = true
	dry, err := Apply(p, opt)
	if err != nil {
		t.Fatal(err)
	}
	if dry.Deleted != 3 || dry.Moved != 2 || !exists(junk) || !exists(emptyFile) || !exists(emptyDir) {
		t.Fatalf("dry run: deleted=%d moved=%d", dry.Deleted, dry.Moved)
	}

	opt.DryRun = false
	res, err := Apply(p, opt)
	if err != nil {
		t.Fatal(err)
	}
	if res.Deleted != 3 || res.Moved != 2 || res.Stubs != 1 || len(res.Problems) != 0 {
		t.Fatalf("apply: deleted=%d moved=%d stubs=%d problems=%+v", res.Deleted, res.Moved, res.Stubs, res.Problems)
	}
	if exists(junk) || exists(emptyFile) || exists(emptyDir) {
		t.Error("junk and empty items must be gone")
	}
	for _, p := range []string{junk, emptyFile, emptyDir} {
		if exists(p + ".removed.txt") {
			t.Errorf("%s must not get a stub", p)
		}
	}
	m := res.Manifests[0]
	if exists(filepath.Join(m.Dir, "Проект A", "Thumbs.db")) {
		t.Error("deleted junk must not be copied into the batch folder")
	}
	if !exists(filepath.Join(m.Dir, "Проект B", "Была пустая", "новый.docx")) || exists(notEmpty) {
		t.Error("a no-longer-empty folder must be quarantined, not deleted")
	}
	if !exists(archive + ".removed.txt") {
		t.Error("archive keeps its stub")
	}
	ops := map[string]Op{}
	for _, e := range m.Entries {
		ops[e.From] = e.Op
		if e.Op == OpDelete && e.To != "" {
			t.Errorf("deleted entry must have no quarantine path: %+v", e)
		}
	}
	if ops[junk] != OpDelete || ops[emptyFile] != OpDelete || ops[emptyDir] != OpDelete || ops[notEmpty] != OpQuarantine || ops[archive] != OpQuarantine {
		t.Errorf("manifest ops: %v", ops)
	}

	b := Describe(m)
	if b.Pending != 4 || b.Deleted != 1 || b.Status != "active" {
		t.Errorf("describe: %+v", b)
	}

	// purge must not trip over deleted entries
	pv, err := Purge(m, true)
	if err != nil || pv.Deleted != 2 || len(pv.Problems) != 0 {
		t.Errorf("purge preview: %+v %v", pv, err)
	}

	// restore: empties recreated, junk counted as gone, the rest moved back
	loaded, _ := LoadManifest(m.Path)
	rr, err := Restore(loaded, RestoreOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if rr.Restored != 4 || rr.Gone != 1 || len(rr.Problems) != 0 {
		t.Errorf("restore: %+v", rr)
	}
	if st, err := os.Stat(emptyFile); err != nil || st.Size() != 0 {
		t.Error("empty file must be recreated empty")
	}
	if st, err := os.Stat(emptyDir); err != nil || !st.IsDir() {
		t.Error("empty folder must be recreated")
	}
	if !exists(archive) || !exists(filepath.Join(notEmpty, "новый.docx")) || exists(junk) {
		t.Error("archive and folder back, junk stays gone")
	}
	again, _ := LoadManifest(m.Path)
	b2 := Describe(again)
	if b2.Pending != 0 || b2.Restored != 4 || b2.Deleted != 1 || b2.Status != "restored" {
		t.Errorf("describe after restore: %+v", b2)
	}

	// a junk-only batch reads as "deleted"
	junk2 := mk(t, root, "Проект A/Thumbs.db", "thumbs")
	res2, err := Apply(&Plan{Schema: PlanSchema, Tool: "folder-inspect", Roots: []string{root},
		Actions: []Action{{Op: OpQuarantine, Path: junk2, Category: "junk", Size: 6}}}, opt)
	if err != nil {
		t.Fatal(err)
	}
	if b3 := Describe(res2.Manifests[0]); b3.Status != "deleted" || b3.Pending != 0 {
		t.Errorf("junk-only batch: %+v", b3)
	}
}
