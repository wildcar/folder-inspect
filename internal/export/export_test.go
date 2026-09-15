package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/fixture"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
	"github.com/wildcar/folder-inspect/internal/scan"
)

func fixtureReport(t *testing.T) *report.Report {
	t.Helper()
	root := t.TempDir()
	if err := fixture.Generate(root, fixture.Options{Scale: 1024}); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	for i := range cfg.SizeRules {
		cfg.SizeRules[i].Threshold /= 1024
	}
	res, err := scan.Walk([]string{root}, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	findings := detect.Run(res, cfg)
	dups := detect.Duplicates(res.Files, detect.DupOptions{MinSize: int64(cfg.Duplicates.MinSize)})
	findings = append(findings, dups.Findings()...)
	detect.Sort(findings)
	return report.Build(res, findings, dups, cfg, "test")
}

func TestParseList(t *testing.T) {
	got, err := ParseList(" html, CSV ,csv")
	if err != nil || len(got) != 2 || got[0] != "csv" || got[1] != "html" {
		t.Errorf("ParseList: %v %v", got, err)
	}
	if _, err := ParseList("pdf"); err == nil {
		t.Error("unknown format must fail")
	}
	if got, _ := ParseList(""); len(got) != 0 {
		t.Error("empty list")
	}
}

func TestCSV(t *testing.T) {
	r := fixtureReport(t)
	i18n.Set(i18n.RU)
	var buf bytes.Buffer
	if err := CSV(r, &buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(buf.Bytes(), utf8BOM) {
		t.Error("CSV must start with a BOM")
	}
	s := string(bytes.TrimPrefix(buf.Bytes(), utf8BOM))
	lines := strings.Split(strings.TrimRight(s, "\r\n"), "\r\n")
	if len(lines) != len(r.Findings)+1 {
		t.Errorf("want %d lines, got %d", len(r.Findings)+1, len(lines))
	}
	if !strings.Contains(lines[0], "Категория;") {
		t.Errorf("RU header must use ';': %q", lines[0])
	}
	if !strings.Contains(s, "Встреча 2026-03-01.mp4") || !strings.Contains(s, "Дубликаты") {
		t.Error("CSV lacks expected rows")
	}

	i18n.Set(i18n.EN)
	t.Cleanup(func() { i18n.Set(i18n.RU) })
	buf.Reset()
	CSV(r, &buf)
	if !strings.Contains(strings.SplitN(buf.String(), "\r\n", 2)[0], "Category,Rule") {
		t.Errorf("EN header must use ',': %q", buf.String()[:60])
	}
}

func TestHTML(t *testing.T) {
	r := fixtureReport(t)
	var buf bytes.Buffer
	if err := HTML(r, &buf); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	for _, want := range []string{"<!doctype html>", "Отчёт folder-inspect", "Большие файлы", "Дубликаты", "Встреча 2026-03-01.mp4", `class="keep"`, "Самые тяжёлые папки"} {
		if !strings.Contains(s, want) {
			t.Errorf("HTML lacks %q", want)
		}
	}
	if strings.Contains(s, "<script") || strings.Contains(s, "http://") || strings.Contains(s, "https://") {
		t.Error("HTML must be self-contained: no scripts or external links")
	}
	// paths with special characters must be escaped
	r.Findings[0].Path = `C:\x\<b>&.docx`
	buf.Reset()
	HTML(r, &buf)
	if !strings.Contains(buf.String(), "&lt;b&gt;&amp;.docx") {
		t.Error("path not escaped")
	}
}

func TestXLSX(t *testing.T) {
	r := fixtureReport(t)
	var buf bytes.Buffer
	if err := XLSX(r, &buf); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(&buf)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	sheets := f.GetSheetList()
	want := []string{"Сводка", "Находки", "Дубликаты", "Самые большие файлы", "Самые тяжёлые папки"}
	if len(sheets) != len(want) {
		t.Fatalf("sheets: %v", sheets)
	}
	for i, s := range want {
		if sheets[i] != s {
			t.Errorf("sheet %d = %q want %q", i, sheets[i], s)
		}
	}
	rows, err := f.GetRows("Находки")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != len(r.Findings)+1 || rows[0][0] != "Категория" {
		t.Errorf("findings sheet: %d rows, header %v", len(rows), rows[0])
	}
	dupRows, _ := f.GetRows("Дубликаты")
	if len(dupRows) != 4 { // header + 3 copies of the contract
		t.Errorf("duplicates sheet rows: %d", len(dupRows))
	}
	if v, _ := f.GetCellValue("Сводка", "A1"); v != "Показатель" {
		t.Errorf("summary A1 = %q", v)
	}
}

func TestWriteFileAndRender(t *testing.T) {
	r := fixtureReport(t)
	dir := t.TempDir()
	for _, format := range Formats {
		p := dir + "/sub/out." + format
		if err := WriteFile(r, format, p); err != nil {
			t.Errorf("%s: %v", format, err)
		}
	}
	if err := Render(r, "pdf", &bytes.Buffer{}); err == nil {
		t.Error("unknown format must fail")
	}
}
