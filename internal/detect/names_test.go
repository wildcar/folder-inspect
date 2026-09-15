package detect

import (
	"strings"
	"testing"
	"time"

	"github.com/wildcar/folder-inspect/internal/scan"
)

func TestSplitName(t *testing.T) {
	cases := []struct {
		in, base, markers string
	}{
		{"Отчёт", "Отчёт", ""},
		{"Копия Отчёт", "Отчёт", "Копия"},
		{"Copy of Report", "Report", "Copy of"},
		{"Отчёт - копия", "Отчёт", "копия"},
		{"Отчёт - копия (2)", "Отчёт", "копия (2)"},
		{"Report - Copy (3)", "Report", "Copy (3)"},
		{"Отчёт (1)", "Отчёт", "(1)"},
		{"Отчёт (Восстановлен)", "Отчёт", "(Восстановлен)"},
		{"Договор_v2", "Договор", "v2"},
		{"Договор v3.1", "Договор", "v3.1"},
		{"Смета_final", "Смета", "final"},
		{"Смета_старый", "Смета", "старый"},
		{"План-new", "План", "new"},
		{"Копия Отчёт (2)", "Отчёт", "Копия, (2)"},
		{"IMG_0002", "IMG_0002", ""},           // numbers alone are not markers
		{"Приложение 1", "Приложение 1", ""},   // neither are spaced numbers
		{"Отчёт 2026-03", "Отчёт 2026-03", ""}, // dates stay
		{"v2", "v2", ""}, // a marker alone is a name
		{"final", "final", ""},
	}
	for _, c := range cases {
		base, markers := SplitName(c.in)
		if base != c.base || strings.Join(markers, ", ") != c.markers {
			t.Errorf("SplitName(%q) = %q %q; want %q %q", c.in, base, strings.Join(markers, ", "), c.base, c.markers)
		}
	}
}

func nf(rel string, size int64, mt time.Time) scan.Entry {
	e := file(rel, size)
	e.ModTime = mt
	return e
}

func TestSimilarNames(t *testing.T) {
	t0 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.Local)
	files := []scan.Entry{
		nf("A/Отчёт.docx", 100, t0),
		nf("A/Копия Отчёт.docx", 100, t0.Add(time.Hour)),
		nf("B/Отчёт (2).docx", 120, t0.Add(2*time.Hour)),
		nf("B/отчёт - копия.DOCX", 90, t0), // case-insensitive name and extension
		nf("A/Смета_v2.xlsx", 50, t0),
		nf("A/Смета_v3.xlsx", 55, t0.Add(time.Hour)), // two variants, no base
		nf("A/Договор.docx", 10, t0),
		nf("C/Договор.docx", 10, t0), // same plain name twice: not reported
		nf("A/Отчёт.pdf", 10, t0),    // different extension: separate key, alone
	}
	dups := DupResult{Groups: []DupGroup{{ID: "dupid", Files: []DupFile{{Path: files[0].Path}, {Path: files[1].Path}}}}}
	r := SimilarNames(files, dups)
	if len(r.Groups) != 2 {
		t.Fatalf("want 2 groups, got %d: %+v", len(r.Groups), r.Groups)
	}
	g := r.Groups[0] // biggest variant size: Отчёт (100+120+90)
	if g.Name != "Отчёт.docx" || g.Count != 4 || g.Variants != 3 || g.Size != 310 {
		t.Errorf("report group: %+v", g)
	}
	if !g.Files[0].IsBase || g.Files[0].Path != files[0].Path || g.Base != files[0].Path {
		t.Errorf("base must come first: %+v", g.Files[0])
	}
	if g.Files[1].Path != files[2].Path { // newest variant next
		t.Errorf("variants must be newest first: %+v", g.Files[1])
	}
	if g.Files[0].DupGroup != "dupid" || g.Files[1].DupGroup != "" {
		t.Errorf("dup group marks: %+v", g.Files)
	}
	s := r.Groups[1]
	if s.Name != "Смета.xlsx" || s.Base != "" || s.Variants != 2 || s.Files[0].Marker != "v3" {
		t.Errorf("variants-only group: %+v", s)
	}
	fs := r.Findings()
	if len(fs) != 6 || fs[0].Category != SimilarName || fs[0].Rule != "" || fs[1].Rule == "" {
		t.Errorf("findings: %+v", fs)
	}
}
