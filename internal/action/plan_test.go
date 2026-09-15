package action

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanValidate(t *testing.T) {
	root := filepath.Join(t.TempDir(), "root")
	in := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	p := New("r.json", []string{root})

	p.Actions = []Action{{Op: OpQuarantine, Path: in("a/Thumbs.db"), Size: 5}}
	if err := p.Validate(); err != nil {
		t.Errorf("simple quarantine must validate: %v", err)
	}
	p.Actions = []Action{{Op: "delete", Path: in("x")}}
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "unknown op") {
		t.Errorf("unknown op: %v", err)
	}
	p.Actions = []Action{{Op: OpQuarantine, Path: filepath.Join(t.TempDir(), "outside.txt")}}
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Errorf("outside root: %v", err)
	}
	p.Actions = []Action{{Op: OpQuarantine, Path: root}}
	if err := p.Validate(); err == nil {
		t.Error("the root itself must never be an action target")
	}
	p.Actions = []Action{{Op: OpQuarantineDuplicate, Path: in("b/copy.docx")}}
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "original") {
		t.Errorf("duplicate without original: %v", err)
	}
	p.Actions = []Action{{Op: OpQuarantineDuplicate, Path: in("b/copy.docx"), Original: in("b/copy.docx")}}
	if err := p.Validate(); err == nil {
		t.Error("original equal to the copy must fail")
	}
	p.Actions = []Action{{Op: OpQuarantineDir, Path: in("b"), Original: in("b/sub")}}
	if err := p.Validate(); err == nil {
		t.Error("original inside the quarantined folder must fail")
	}
	p.Actions = []Action{
		{Op: OpQuarantineDuplicate, Path: in("b/copy.docx"), Original: in("a/orig.docx")},
		{Op: OpQuarantine, Path: in("a/orig.docx")},
	}
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "both an original") {
		t.Errorf("original planned for quarantine: %v", err)
	}
	p.Actions = []Action{{Op: OpQuarantine, Path: in("x")}, {Op: OpQuarantine, Path: in("x")}}
	if err := p.Validate(); err == nil || !strings.Contains(err.Error(), "twice") {
		t.Errorf("duplicate path: %v", err)
	}
	p.Actions = []Action{
		{Op: OpQuarantineDuplicate, Path: in("b/copy.docx"), Original: in("a/orig.docx"), Size: 10},
		{Op: OpQuarantineDir, Path: in("old/Приложения (копия)"), Original: in("Приложения"), Size: 20, IsDir: true},
	}
	if err := p.Validate(); err != nil {
		t.Errorf("valid plan rejected: %v", err)
	}
	if p.TotalSize() != 30 {
		t.Errorf("TotalSize %d", p.TotalSize())
	}
}

func TestPlanSaveLoadNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".folder-inspect", "reports")
	p := New("r.json", []string{root})
	p.Actions = []Action{{Op: OpQuarantine, Path: filepath.Join(root, "junk.tmp"), Size: 1}}
	p1, err := p.Save(dir)
	if err != nil {
		t.Fatal(err)
	}
	p2, err := p.Save(dir) // same second → suffixed name
	if err != nil {
		t.Fatal(err)
	}
	if p1 == p2 || !strings.HasSuffix(p2, "-2.json") {
		t.Errorf("second save must not overwrite: %q %q", p1, p2)
	}
	back, err := Load(p1)
	if err != nil || len(back.Actions) != 1 || back.Roots[0] != root {
		t.Errorf("Load: %v %+v", err, back)
	}
	bad := New("r.json", nil)
	if _, err := bad.Save(dir); err == nil {
		t.Error("plan without roots must not save")
	}
}
