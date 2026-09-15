package action

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/report"
)

// rulesReport builds a synthetic report: junk + archive + an oversized video
// finding, an empty folder, a duplicate group with three copies (one inside
// an identical-folder copy), and an identical-folder group.
func rulesReport(t *testing.T) (*report.Report, string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "repo")
	at := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	rep := &report.Report{
		Roots: []string{root},
		Findings: []detect.Finding{
			{Category: detect.Junk, Path: at("a/Thumbs.db"), Rel: "a/Thumbs.db", Root: root, Size: 10},
			{Category: detect.Archive, Path: at("a/old.zip"), Rel: "a/old.zip", Root: root, Size: 5000},
			{Category: detect.Oversize, Rule: "huge", Path: at("video/call.mp4"), Rel: "video/call.mp4", Root: root, Size: 900_000},
			{Category: detect.EmptyDir, Path: at("empty"), Rel: "empty", Root: root, IsDir: true},
			{Category: detect.SimilarName, Path: at("a/report (1).docx"), Rel: "a/report (1).docx", Root: root, Size: 100},
			{Category: detect.Junk, Path: at("copy/Thumbs.db"), Rel: "copy/Thumbs.db", Root: root, Size: 10},
		},
		Duplicates: []detect.DupGroup{{
			ID: "g1", Size: 300, Count: 3, Suggested: at("orig/doc.docx"),
			Files: []detect.DupFile{
				{Path: at("orig/doc.docx"), Rel: "orig/doc.docx", Root: root, ModTime: t0},
				{Path: at("copy/doc.docx"), Rel: "copy/doc.docx", Root: root, ModTime: t0.Add(time.Hour)},
				{Path: at("deep/er/doc.docx"), Rel: "deep/er/doc.docx", Root: root, ModTime: t0.Add(2 * time.Hour)},
			},
		}},
		DirDuplicates: []detect.DirGroup{{
			ID: "d1", Size: 310, Files: 2, Count: 2, Suggested: at("orig"),
			Dirs: []detect.DupDir{
				{Path: at("orig"), Rel: "orig", Root: root, ModTime: t0, Size: 310, Files: 2},
				{Path: at("copy"), Rel: "copy", Root: root, ModTime: t0.Add(time.Hour), Size: 310, Files: 2},
			},
		}},
	}
	return rep, root
}

func paths(p *Plan) map[string]Action {
	m := map[string]Action{}
	for _, a := range p.Actions {
		m[a.Path] = a
	}
	return m
}

func TestFromRulesCategories(t *testing.T) {
	rep, root := rulesReport(t)
	p, st, err := FromRules(rep, "r.json", Rules{Categories: []detect.Category{detect.Junk, detect.Archive, detect.EmptyDir}})
	if err != nil {
		t.Fatal(err)
	}
	got := paths(p)
	for _, rel := range []string{"a/Thumbs.db", "a/old.zip", "empty", "copy/Thumbs.db"} {
		a, ok := got[filepath.Join(root, filepath.FromSlash(rel))]
		if !ok || a.Op != OpQuarantine {
			t.Errorf("%s: missing or wrong op (%+v)", rel, a)
		}
	}
	if len(p.Actions) != 4 || st.Findings != 4 {
		t.Errorf("got %d actions, stats %+v", len(p.Actions), st)
	}
	if !got[filepath.Join(root, "empty")].IsDir {
		t.Error("empty dir action must be IsDir")
	}
	if _, ok := got[filepath.Join(root, "video", "call.mp4")]; ok {
		t.Error("oversize was not selected but planned")
	}
}

func TestFromRulesFilters(t *testing.T) {
	rep, root := rulesReport(t)
	p, _, err := FromRules(rep, "r.json", Rules{Categories: []detect.Category{detect.Junk, detect.Archive, detect.Oversize}, Include: []string{"a/*"}, Exclude: []string{"*.zip"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Actions) != 1 || p.Actions[0].Path != filepath.Join(root, "a", "Thumbs.db") {
		t.Errorf("include/exclude: got %+v", p.Actions)
	}
	p, _, err = FromRules(rep, "r.json", Rules{Categories: RuleCategories, MinSize: 100_000})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Actions) != 1 || p.Actions[0].Category != string(detect.Oversize) {
		t.Errorf("min-size: got %+v", p.Actions)
	}
}

func TestFromRulesDuplicatesPolicies(t *testing.T) {
	rep, root := rulesReport(t)
	cases := map[KeepPolicy]string{
		KeepOldest:     "orig/doc.docx",
		KeepNewest:     "deep/er/doc.docx",
		KeepShallowest: "copy/doc.docx", // same depth and length as orig/, so alphabetical order decides
	}
	for policy, keepRel := range cases {
		p, st, err := FromRules(rep, "r.json", Rules{Duplicates: policy})
		if err != nil {
			t.Fatalf("%s: %v", policy, err)
		}
		keep := filepath.Join(root, filepath.FromSlash(keepRel))
		if len(p.Actions) != 2 || st.Duplicates != 2 {
			t.Fatalf("%s: got %d actions", policy, len(p.Actions))
		}
		for _, a := range p.Actions {
			if a.Op != OpQuarantineDuplicate || a.Original != keep || a.Path == keep || a.Group != "g1" || a.Size != 300 {
				t.Errorf("%s: bad action %+v (keep %s)", policy, a, keep)
			}
		}
	}
}

func TestFromRulesDirDuplicatesWinOverItemsInside(t *testing.T) {
	rep, root := rulesReport(t)
	p, st, err := FromRules(rep, "r.json", Rules{Categories: []detect.Category{detect.Junk}, Duplicates: KeepNewest, DirDuplicates: KeepOldest})
	if err != nil {
		t.Fatal(err)
	}
	got := paths(p)
	dir, ok := got[filepath.Join(root, "copy")]
	if !ok || dir.Op != OpQuarantineDir || dir.Original != filepath.Join(root, "orig") || !dir.IsDir {
		t.Fatalf("folder copy not planned: %+v", dir)
	}
	if _, ok := got[filepath.Join(root, "copy", "Thumbs.db")]; ok {
		t.Error("junk inside a quarantined folder must be skipped")
	}
	if _, ok := got[filepath.Join(root, "copy", "doc.docx")]; ok {
		t.Error("duplicate inside a quarantined folder must be skipped")
	}
	// KeepNewest would keep deep/er/doc.docx, but orig/ is the kept folder of
	// the identical-folder group, so its file is protected and stays.
	a := got[filepath.Join(root, "deep", "er", "doc.docx")]
	if a.Op != OpQuarantineDuplicate || a.Original != filepath.Join(root, "orig", "doc.docx") {
		t.Errorf("duplicate handling with a kept folder: %+v", a)
	}
	if _, ok := got[filepath.Join(root, "orig", "doc.docx")]; ok {
		t.Error("file inside the kept folder must not be quarantined")
	}
	if st.DirDuplicates != 1 || st.Duplicates != 1 || st.Findings != 1 || st.Skipped != 2 {
		t.Errorf("stats %+v", st)
	}
}

func TestFromRulesRejects(t *testing.T) {
	rep, _ := rulesReport(t)
	if _, _, err := FromRules(rep, "r.json", Rules{}); err == nil {
		t.Error("empty rules must fail")
	}
	if _, _, err := FromRules(rep, "r.json", Rules{Categories: []detect.Category{detect.SimilarName}}); err == nil {
		t.Error("similar-name by rule must fail")
	}
	if _, err := ParseCategories("junk,similar-name"); err == nil {
		t.Error("ParseCategories must reject similar-name")
	}
	if _, err := ParseCategories("duplicate"); err == nil {
		t.Error("ParseCategories must reject duplicate (policy flag instead)")
	}
	cs, err := ParseCategories("all,junk")
	if err != nil || len(cs) != len(RuleCategories) {
		t.Errorf("all: %v %v", cs, err)
	}
	if _, err := ParseKeepPolicy("random"); err == nil {
		t.Error("bad policy accepted")
	}
	if pol, err := ParseKeepPolicy(" Newest "); err != nil || pol != KeepNewest {
		t.Errorf("policy parse: %q %v", pol, err)
	}
}

func TestFromRulesEmptyGroupsSkipped(t *testing.T) {
	rep, _ := rulesReport(t)
	// Every copy excluded → the group yields nothing and is counted as skipped.
	p, st, err := FromRules(rep, "r.json", Rules{Duplicates: KeepOldest, Exclude: []string{"*.docx"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Actions) != 0 || st.GroupsSkipped != 1 {
		t.Errorf("got %d actions, stats %+v", len(p.Actions), st)
	}
}
