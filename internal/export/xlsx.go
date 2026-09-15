package export

import (
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
)

// XLSX writes a workbook: summary, all findings (filterable), duplicate
// groups, top files, top folders and read errors.
func XLSX(r *report.Report, w io.Writer) error {
	f := excelize.NewFile()
	defer f.Close()
	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return err
	}

	// Summary
	summary := i18n.T("sheet.summary")
	if err := f.SetSheetName("Sheet1", summary); err != nil {
		return err
	}
	rows := [][]any{
		{i18n.T("col.metric"), i18n.T("col.value")},
		{i18n.T("sum.tool"), r.Tool + " " + r.Version},
		{i18n.T("sum.started"), report.FormatTime(r.Started)},
		{i18n.T("sum.duration"), r.Duration().Round(1e6).String()},
		{i18n.T("sum.roots"), strings.Join(r.Roots, "\n")},
		{i18n.T("sum.files"), r.Stats.Files},
		{i18n.T("sum.dirs"), r.Stats.Dirs},
		{i18n.T("sum.total"), report.HumanSize(r.Stats.TotalSize)},
		{i18n.T("sum.errors"), r.Stats.Errors},
		{i18n.T("sum.skipped"), r.Stats.Skipped},
		{},
		{i18n.T("col.category"), i18n.T("col.count"), i18n.T("col.size"), i18n.T("col.size_bytes")},
	}
	catHeader := len(rows)
	for _, s := range r.Summary {
		rows = append(rows, []any{report.CategoryName(s.Category), s.Count, report.HumanSize(s.Size), s.Size})
	}
	if err := writeRows(f, summary, rows); err != nil {
		return err
	}
	f.SetCellStyle(summary, "A1", "B1", bold)
	f.SetCellStyle(summary, fmt.Sprintf("A%d", catHeader), fmt.Sprintf("D%d", catHeader), bold)
	f.SetColWidth(summary, "A", "A", 28)
	f.SetColWidth(summary, "B", "B", 60)

	// Findings
	findings := i18n.T("sheet.findings")
	rows = [][]any{{
		i18n.T("col.category"), i18n.T("col.rule"), i18n.T("col.group"), i18n.T("col.path"),
		i18n.T("col.size_bytes"), i18n.T("col.size"), i18n.T("col.mtime"), i18n.T("col.threshold"),
		i18n.T("col.detail"), i18n.T("col.root"),
	}}
	for _, x := range r.Findings {
		threshold, detail := "", ""
		if x.Threshold > 0 {
			threshold = report.HumanSize(x.Threshold)
		}
		if x.Detail != "" {
			detail = i18n.T("detail." + x.Detail)
		}
		rows = append(rows, []any{
			report.CategoryName(x.Category), x.Rule, x.Group, x.Path, x.Size, report.HumanSize(x.Size),
			report.FormatTime(x.ModTime), threshold, detail, x.Root,
		})
	}
	if err := newTable(f, findings, rows, bold, []float64{16, 14, 14, 80, 14, 12, 17, 12, 22, 40}); err != nil {
		return err
	}

	// Duplicates
	dups := i18n.T("sheet.duplicates")
	rows = [][]any{{
		i18n.T("col.group"), i18n.T("col.size_bytes"), i18n.T("col.size"), i18n.T("col.count"),
		i18n.T("col.wasted"), i18n.T("col.suggested"), i18n.T("col.path"), i18n.T("col.mtime"),
	}}
	for _, g := range r.Duplicates {
		for _, x := range g.Files {
			suggested := ""
			if x.Path == g.Suggested {
				suggested = "*"
			}
			rows = append(rows, []any{g.ID, g.Size, report.HumanSize(g.Size), g.Count, report.HumanSize(g.Wasted), suggested, x.Path, report.FormatTime(x.ModTime)})
		}
	}
	if err := newTable(f, dups, rows, bold, []float64{14, 14, 12, 8, 12, 10, 80, 17}); err != nil {
		return err
	}

	// Duplicate folders
	dirDups := i18n.T("sheet.dir_duplicates")
	rows = [][]any{{
		i18n.T("col.group"), i18n.T("col.size_bytes"), i18n.T("col.size"), i18n.T("col.files"),
		i18n.T("col.count"), i18n.T("col.wasted"), i18n.T("col.suggested"), i18n.T("col.path"), i18n.T("col.mtime"),
	}}
	for _, g := range r.DirDuplicates {
		for _, d := range g.Dirs {
			suggested := ""
			if d.Path == g.Suggested {
				suggested = "*"
			}
			rows = append(rows, []any{g.ID, g.Size, report.HumanSize(g.Size), g.Files, g.Count, report.HumanSize(g.Wasted), suggested, d.Path, report.FormatTime(d.ModTime)})
		}
	}
	if err := newTable(f, dirDups, rows, bold, []float64{14, 14, 12, 8, 8, 12, 10, 80, 17}); err != nil {
		return err
	}

	// Folders with shared content
	overlaps := i18n.T("sheet.overlaps")
	rows = [][]any{{
		i18n.T("col.path"), i18n.T("col.related"), i18n.T("col.shared_files"), i18n.T("col.shared_bytes"),
		i18n.T("col.size"), i18n.T("col.ratio"), i18n.T("col.ratio_a"), i18n.T("col.ratio_b"),
	}}
	for _, o := range r.DirOverlaps {
		rows = append(rows, []any{o.A.Path, o.B.Path, o.SharedFiles, o.SharedBytes, report.HumanSize(o.SharedBytes),
			report.Percent(o.Ratio), report.Percent(o.RatioA), report.Percent(o.RatioB)})
	}
	if err := newTable(f, overlaps, rows, bold, []float64{70, 70, 10, 14, 12, 12, 12, 12}); err != nil {
		return err
	}

	// Top files / folders
	for _, top := range []struct {
		sheet string
		items []report.Item
		dirs  bool
	}{
		{i18n.T("sheet.top_files"), r.TopFiles, false},
		{i18n.T("sheet.top_dirs"), r.TopDirs, true},
	} {
		rows = [][]any{{i18n.T("col.size_bytes"), i18n.T("col.size"), i18n.T("col.path")}}
		if top.dirs {
			rows[0] = append(rows[0], i18n.T("col.files"))
		}
		for _, it := range top.items {
			row := []any{it.Size, report.HumanSize(it.Size), it.Path}
			if top.dirs {
				row = append(row, it.Files)
			}
			rows = append(rows, row)
		}
		if err := newTable(f, top.sheet, rows, bold, []float64{14, 12, 80, 8}); err != nil {
			return err
		}
	}

	// Errors
	if len(r.Errors) > 0 {
		errs := i18n.T("sheet.errors")
		rows = [][]any{{i18n.T("col.path"), i18n.T("col.error")}}
		for _, e := range r.Errors {
			rows = append(rows, []any{e.Path, e.Err})
		}
		if err := newTable(f, errs, rows, bold, []float64{80, 60}); err != nil {
			return err
		}
	}

	f.SetActiveSheet(0)
	return f.Write(w)
}

// newTable creates a sheet with a bold, frozen, filterable header row.
func newTable(f *excelize.File, sheet string, rows [][]any, headerStyle int, widths []float64) error {
	if _, err := f.NewSheet(sheet); err != nil {
		return err
	}
	if err := writeRows(f, sheet, rows); err != nil {
		return err
	}
	cols := len(rows[0])
	last, _ := excelize.ColumnNumberToName(cols)
	f.SetCellStyle(sheet, "A1", last+"1", headerStyle)
	f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"})
	if len(rows) > 1 {
		f.AutoFilter(sheet, fmt.Sprintf("A1:%s%d", last, len(rows)), nil)
	}
	for i, wdt := range widths {
		if i >= cols {
			break
		}
		col, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, col, col, wdt)
	}
	return nil
}

func writeRows(f *excelize.File, sheet string, rows [][]any) error {
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return err
		}
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			return err
		}
	}
	return nil
}
