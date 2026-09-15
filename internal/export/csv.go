package export

import (
	"encoding/csv"
	"io"
	"strconv"

	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
)

// utf8BOM is the byte order mark Excel needs to read UTF-8 CSV correctly.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// CSV writes one row per finding. The file starts with a UTF-8 BOM so
// Excel shows Cyrillic correctly, and uses ";" for Russian (Excel's list
// separator in that locale) and "," for English.
func CSV(r *report.Report, w io.Writer) error {
	if _, err := w.Write(utf8BOM); err != nil {
		return err
	}
	cw := csv.NewWriter(w)
	cw.UseCRLF = true
	if i18n.Lang() == i18n.RU {
		cw.Comma = ';'
	}
	header := []string{
		i18n.T("col.category"), i18n.T("col.rule"), i18n.T("col.group"), i18n.T("col.path"),
		i18n.T("col.size_bytes"), i18n.T("col.size"), i18n.T("col.mtime"), i18n.T("col.threshold"),
		i18n.T("col.detail"), i18n.T("col.root"),
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, f := range r.Findings {
		threshold := ""
		if f.Threshold > 0 {
			threshold = report.HumanSize(f.Threshold)
		}
		detail := ""
		if f.Detail != "" {
			detail = i18n.T("detail." + f.Detail)
		}
		row := []string{
			report.CategoryName(f.Category), f.Rule, f.Group, f.Path,
			strconv.FormatInt(f.Size, 10), report.HumanSize(f.Size), report.FormatTime(f.ModTime),
			threshold, detail, f.Root,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	// Folder pairs are not findings; add them as rows with the second
	// folder and the overlap figures in the note column.
	for _, o := range r.DirOverlaps {
		row := []string{
			report.CategoryName(detect.DirOverlap), "", "", o.A.Path,
			strconv.FormatInt(o.SharedBytes, 10), report.HumanSize(o.SharedBytes), "",
			"", o.B.Path + " | " + report.OverlapLine(o), o.A.Root,
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
