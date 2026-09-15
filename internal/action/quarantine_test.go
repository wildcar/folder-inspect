package action

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListDescribePurge(t *testing.T) {
	root := t.TempDir()
	a := mk(t, root, "a.zip", "aaaa")
	b := mk(t, root, "sub/b.zip", "bbbbbb")
	p := New("r.json", []string{root})
	p.Actions = []Action{
		{Op: OpQuarantine, Path: a, Category: "archive", Size: 4},
		{Op: OpQuarantine, Path: b, Category: "archive", Size: 6},
	}
	res, err := Apply(p, defaultOpts(time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local)))
	if err != nil || res.Moved != 2 {
		t.Fatal(err, res)
	}
	// second batch, later
	c := mk(t, root, "c.zip", "cc")
	p2 := New("r.json", []string{root})
	p2.Actions = []Action{{Op: OpQuarantine, Path: c, Category: "archive", Size: 2}}
	if _, err := Apply(p2, defaultOpts(time.Date(2026, 1, 2, 12, 0, 0, 0, time.Local))); err != nil {
		t.Fatal(err)
	}

	batches, err := ListBatches(root, "")
	if err != nil || len(batches) != 2 {
		t.Fatalf("ListBatches: %v %d", err, len(batches))
	}
	if !batches[0].Manifest.Created.After(batches[1].Manifest.Created) {
		t.Error("newest batch must come first")
	}
	first := batches[1]
	if first.Items != 2 || first.Pending != 2 || first.Size != 10 || first.Status != "active" {
		t.Errorf("describe: %+v", first)
	}

	// restore one entry → partial (entries are stored deeper-first, so take
	// whatever is first and treat the other one as the pending copy)
	m := first.Manifest
	rr, _ := Restore(&Manifest{Schema: m.Schema, Tool: m.Tool, Root: m.Root, Dir: m.Dir, Path: m.Path,
		Entries: []Entry{m.Entries[0]}}, RestoreOptions{})
	if rr.Restored != 1 {
		t.Fatalf("restore one: %+v", rr)
	}
	m.Entries[0].Restored = true
	m.save()
	pending := m.Entries[1]
	d := Describe(m)
	if d.Status != "partial" || d.Pending != 1 || d.Restored != 1 || d.Size != pending.Size {
		t.Errorf("partial: %+v", d)
	}

	// purge: dry run, then real
	pr, err := Purge(m, true)
	if err != nil || pr.Deleted != 1 || pr.Bytes != pending.Size {
		t.Errorf("purge dry: %v %+v", err, pr)
	}
	if !exists(pending.To) {
		t.Error("dry purge must not delete")
	}
	pr, err = Purge(m, false)
	if err != nil || pr.Deleted != 1 || len(pr.Problems) != 0 {
		t.Errorf("purge: %v %+v", err, pr)
	}
	if exists(pending.To) {
		t.Error("purged copy must be gone")
	}
	if !exists(m.Path) {
		t.Error("manifest must stay after purge")
	}
	if !exists(pending.Stub) {
		t.Error("stubs stay after purge")
	}
	again, _ := LoadManifest(m.Path)
	if d := Describe(again); again.Purged == nil || d.Status != "purged" || d.Pending != 0 || d.Size != 0 {
		t.Errorf("manifest must record the purge and show nothing pending: %+v", d)
	}
	if _, err := Purge(again, false); err == nil {
		t.Error("second purge must be refused")
	}
	// restore after purge reports the missing copy, does not crash
	rr, _ = Restore(again, RestoreOptions{})
	if rr.Restored != 0 || len(rr.Problems) != 1 {
		t.Errorf("restore after purge: %+v", rr)
	}
}

func TestPurgeRefusesOutsideBatch(t *testing.T) {
	root := t.TempDir()
	victim := mk(t, root, "important.docx", "keep me")
	m := &Manifest{Schema: ManifestSchema, Tool: "folder-inspect", Root: root,
		Dir: filepath.Join(root, ".folder-inspect", "quarantine", "x"), Path: filepath.Join(root, ".folder-inspect", "quarantine", "x", ManifestName),
		Entries: []Entry{{Op: OpQuarantine, From: victim + ".old", To: victim, Size: 7}}}
	os.MkdirAll(m.Dir, 0o755)
	pr, err := Purge(m, false)
	if err != nil || pr.Deleted != 0 || len(pr.Problems) != 1 || !exists(victim) {
		t.Errorf("a To outside the batch folder must never be deleted: %v %+v", err, pr)
	}
}
