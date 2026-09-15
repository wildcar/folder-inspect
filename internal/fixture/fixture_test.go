package fixture

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
	"github.com/wildcar/folder-inspect/internal/scan"
)

const scale = 1024

// scaledConfig divides every size threshold by the fixture scale.
func scaledConfig() *config.Config {
	cfg := config.Default()
	for i := range cfg.SizeRules {
		cfg.SizeRules[i].Threshold /= scale
	}
	return cfg
}

func sha(t *testing.T, p string) [32]byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(b)
}

func TestGenerateIsDeterministicAndRefusesNonEmpty(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	if err := Generate(a, Options{Scale: scale}); err != nil {
		t.Fatal(err)
	}
	if err := Generate(b, Options{Scale: scale}); err != nil {
		t.Fatal(err)
	}
	c1 := filepath.Join(a, "Проект A", "Договоры", "Договор.docx")
	c2 := filepath.Join(b, "Проект B", "Копия Договор.docx")
	if sha(t, c1) != sha(t, c2) {
		t.Error("duplicates must be byte-identical across runs")
	}
	if sha(t, c1) == sha(t, filepath.Join(a, "Проект A", "Отчёты", "Отчёт за март (1).docx")) {
		t.Error("near-duplicate must differ in content")
	}
	if err := Generate(a, Options{}); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Errorf("must refuse non-empty dir, got %v", err)
	}
}

func TestEndToEndScan(t *testing.T) {
	root := t.TempDir()
	if err := Generate(root, Options{Scale: scale}); err != nil {
		t.Fatal(err)
	}
	cfg := scaledConfig()
	res, err := scan.Walk([]string{root}, scan.Options{Exclude: cfg.Exclude})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Files) != len(Files) {
		t.Fatalf("indexed %d files, fixture has %d", len(res.Files), len(Files))
	}
	findings := detect.Run(res, cfg)
	rep := report.Build(res, findings, cfg, "test")

	want := map[detect.Category]int{
		detect.Oversize: 5, detect.Archive: 1, detect.Distributive: 1,
		detect.Junk: 4, detect.EmptyDir: 2, detect.EmptyFile: 1,
	}
	got := map[detect.Category]int{}
	for _, s := range rep.Summary {
		got[s.Category] = s.Count
	}
	for c, n := range want {
		if got[c] != n {
			t.Errorf("%s: %d findings, want %d", c, got[c], n)
		}
	}
	rules := map[string]string{}
	for _, f := range findings {
		if f.Category == detect.Oversize {
			rules[filepath.Base(f.Rel)] = f.Rule
		}
	}
	wantRules := map[string]string{
		"Встреча 2026-03-01.mp4": "huge", "Презентация для заказчика.pptx": "presentations",
		"Итоговый отчёт.docx": "documents", "IMG_0001.jpg": "images", "Сканы протоколов.pdf": "pdf",
	}
	for name, rule := range wantRules {
		if rules[name] != rule {
			t.Errorf("%s: rule %q want %q", name, rules[name], rule)
		}
	}
	if len(rep.TopFiles) == 0 || !strings.HasSuffix(rep.TopFiles[0].Rel, ".mp4") {
		t.Errorf("largest file should be the video: %+v", rep.TopFiles)
	}
	if len(rep.TopDirs) == 0 || rep.TopDirs[0].Rel != "Проект A" {
		t.Errorf("heaviest folder should be Проект A: %+v", rep.TopDirs)
	}

	// JSON round trip
	out := filepath.Join(t.TempDir(), "sub", "report.json")
	if err := rep.WriteJSON(out); err != nil {
		t.Fatal(err)
	}
	back, err := report.ReadJSON(out)
	if err != nil {
		t.Fatal(err)
	}
	if back.Schema != report.SchemaVersion || len(back.Findings) != len(findings) || back.Stats.Files != len(Files) {
		t.Errorf("round trip lost data: %+v", back.Stats)
	}
	if back.Config.SizeRules[0].Threshold != cfg.SizeRules[0].Threshold {
		t.Error("config thresholds must survive the JSON round trip")
	}

	// console output in both languages mentions every category and the file
	for _, lang := range []string{i18n.RU, i18n.EN} {
		i18n.Set(lang)
		var buf bytes.Buffer
		rep.PrintConsole(&buf, report.ConsoleOptions{ReportPath: out})
		s := buf.String()
		for _, c := range detect.Categories {
			if !strings.Contains(s, i18n.T("cat."+string(c))) {
				t.Errorf("[%s] console lacks category %s:\n%s", lang, c, s)
			}
		}
		if !strings.Contains(s, "Встреча 2026-03-01.mp4") || !strings.Contains(s, out) {
			t.Errorf("[%s] console lacks the video or report path:\n%s", lang, s)
		}
	}
	i18n.Set(i18n.RU)
}
