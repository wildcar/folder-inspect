package export

import (
	"html/template"
	"io"
	"strings"

	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
	"github.com/wildcar/folder-inspect/internal/scan"
)

// HTML writes a self-contained page (inline CSS, no scripts, no external
// resources) suitable for sending by e-mail or opening from a share.
func HTML(r *report.Report, w io.Writer) error {
	return htmlTmpl.Execute(w, newHTMLView(r))
}

type kv struct{ K, V string }

type row struct {
	Size, Path, Tag, MTime string
}

type section struct {
	Name  string
	Count int
	Rows  []row
}

type dupFile struct {
	Path, MTime string
	Suggested   bool
}

type dupGroup struct {
	ID, Size, Copies, Wasted string
	Files                    []dupFile
}

type htmlView struct {
	L        map[string]string
	Title    string
	ToolLine string
	Roots    []string
	Stats    []kv
	Summary  []struct{ Name, Count, Size string }
	Sections []section
	Dups     []dupGroup
	DupLine  string
	TopFiles []row
	TopDirs  []row
	Errors   []scan.Error
	Skipped  []string
}

func newHTMLView(r *report.Report) htmlView {
	v := htmlView{
		L: map[string]string{
			"summary":   i18n.T("scan.summary"),
			"category":  i18n.T("scan.category"),
			"count":     i18n.T("scan.count"),
			"size":      i18n.T("scan.size"),
			"nothing":   i18n.T("scan.nothing"),
			"topFiles":  i18n.T("scan.top_files"),
			"topDirs":   i18n.T("scan.top_dirs"),
			"errors":    i18n.T("html.errors"),
			"skipped":   i18n.T("html.skipped"),
			"path":      i18n.T("col.path"),
			"mtime":     i18n.T("col.mtime"),
			"error":     i18n.T("col.error"),
			"legend":    i18n.T("dup.legend"),
			"generated": i18n.Tf("html.generated", report.FormatTime(r.Finished)),
		},
		Title:    i18n.T("html.title"),
		ToolLine: r.Tool + " " + r.Version,
		Roots:    r.Roots,
		Errors:   r.Errors,
		Skipped:  r.Skipped,
	}
	v.Stats = []kv{
		{i18n.T("sum.files"), itoa(r.Stats.Files)},
		{i18n.T("sum.dirs"), itoa(r.Stats.Dirs)},
		{i18n.T("sum.total"), report.HumanSize(r.Stats.TotalSize)},
		{i18n.T("sum.duration"), r.Duration().Round(1e6).String()},
	}
	if r.Stats.Hashed > 0 {
		v.Stats = append(v.Stats, kv{i18n.T("sum.hashed"), itoa(r.Stats.Hashed) + " / " + report.HumanSize(r.Stats.HashedBytes)})
	}
	for _, s := range r.Summary {
		v.Summary = append(v.Summary, struct{ Name, Count, Size string }{
			report.CategoryName(s.Category), itoa(s.Count), report.HumanSize(s.Size),
		})
	}
	for _, c := range detect.Categories {
		if c == detect.Duplicate {
			continue
		}
		items := r.ByCategory(c)
		if len(items) == 0 {
			continue
		}
		sec := section{Name: report.CategoryName(c), Count: len(items)}
		for _, f := range items {
			sec.Rows = append(sec.Rows, row{report.HumanSize(f.Size), f.Path, report.Qualifier(f), report.FormatTime(f.ModTime)})
		}
		v.Sections = append(v.Sections, sec)
	}
	var wasted int64
	for _, g := range r.Duplicates {
		wasted += g.Wasted
		dg := dupGroup{ID: g.ID, Size: report.HumanSize(g.Size), Copies: i18n.Tf("dup.copies", g.Count), Wasted: report.HumanSize(g.Wasted)}
		for _, f := range g.Files {
			dg.Files = append(dg.Files, dupFile{f.Path, report.FormatTime(f.ModTime), f.Path == g.Suggested})
		}
		v.Dups = append(v.Dups, dg)
	}
	if len(r.Duplicates) > 0 {
		v.DupLine = i18n.Tf("dup.header", report.CategoryName(detect.Duplicate), len(r.Duplicates), report.HumanSize(wasted))
	}
	for _, it := range r.TopFiles {
		v.TopFiles = append(v.TopFiles, row{Size: report.HumanSize(it.Size), Path: it.Path})
	}
	for _, it := range r.TopDirs {
		v.TopDirs = append(v.TopDirs, row{Size: report.HumanSize(it.Size), Path: it.Path, Tag: itoa(it.Files)})
	}
	return v
}

func itoa(n int) string {
	return strings.TrimSpace(strings.Replace(template.HTMLEscapeString(intString(n)), " ", " ", -1))
}

func intString(n int) string {
	// group thousands with a thin space for readability: 12 345
	s := []byte{}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := []byte{}
	if n == 0 {
		digits = append(digits, '0')
	}
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		s = append(s, digits[i])
		if i > 0 && i%3 == 0 {
			s = append(s, ' ')
		}
	}
	if neg {
		return "-" + string(s)
	}
	return string(s)
}

var htmlTmpl = template.Must(template.New("report").Parse(`<!doctype html>
<html lang="{{/* language is implied by the text */}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
<style>
  body { font: 14px/1.45 system-ui, -apple-system, "Segoe UI", Roboto, sans-serif; color: #1f2328; background: #fff; margin: 0; padding: 24px; }
  h1 { font-size: 22px; margin: 0 0 4px; }
  h2 { font-size: 17px; margin: 28px 0 8px; border-bottom: 1px solid #d0d7de; padding-bottom: 4px; }
  .muted { color: #59636e; }
  table { border-collapse: collapse; width: 100%; margin: 8px 0 16px; }
  th, td { text-align: left; padding: 5px 8px; border-bottom: 1px solid #eaeef2; vertical-align: top; }
  th { background: #f6f8fa; font-weight: 600; white-space: nowrap; }
  td.num, th.num { text-align: right; white-space: nowrap; font-variant-numeric: tabular-nums; }
  td.path { font-family: ui-monospace, Consolas, monospace; font-size: 12.5px; word-break: break-all; }
  td.tag { color: #59636e; white-space: nowrap; }
  .stats { display: flex; flex-wrap: wrap; gap: 12px 28px; margin: 12px 0 4px; }
  .stats div b { display: block; font-size: 18px; }
  .group { margin: 10px 0 14px; padding: 8px 12px; border: 1px solid #d0d7de; border-radius: 6px; }
  .group .head { font-weight: 600; }
  .group ul { list-style: none; margin: 6px 0 0; padding: 0; }
  .group li { font-family: ui-monospace, Consolas, monospace; font-size: 12.5px; padding: 2px 0; word-break: break-all; }
  .group li.keep::before { content: "★ "; color: #9a6700; }
  .group li:not(.keep)::before { content: "· "; color: #8c959f; }
  ul.roots { margin: 4px 0; padding-left: 18px; }
  ul.roots li { font-family: ui-monospace, Consolas, monospace; font-size: 12.5px; }
</style>
</head>
<body>
<h1>{{.Title}}</h1>
<div class="muted">{{.ToolLine}} · {{.L.generated}}</div>
<ul class="roots">{{range .Roots}}<li>{{.}}</li>{{end}}</ul>
<div class="stats">{{range .Stats}}<div><b>{{.V}}</b><span class="muted">{{.K}}</span></div>{{end}}</div>

<h2>{{.L.summary}}</h2>
{{if .Summary}}
<table><tr><th>{{.L.category}}</th><th class="num">{{.L.count}}</th><th class="num">{{.L.size}}</th></tr>
{{range .Summary}}<tr><td>{{.Name}}</td><td class="num">{{.Count}}</td><td class="num">{{.Size}}</td></tr>{{end}}
</table>
{{else}}<p>{{.L.nothing}}</p>{{end}}

{{range .Sections}}
<h2>{{.Name}} <span class="muted">({{.Count}})</span></h2>
<table><tr><th class="num">{{$.L.size}}</th><th>{{$.L.path}}</th><th></th><th>{{$.L.mtime}}</th></tr>
{{range .Rows}}<tr><td class="num">{{.Size}}</td><td class="path">{{.Path}}</td><td class="tag">{{.Tag}}</td><td class="tag">{{.MTime}}</td></tr>{{end}}
</table>
{{end}}

{{if .Dups}}
<h2>{{.DupLine}}</h2>
<div class="muted">{{.L.legend}}</div>
{{range .Dups}}
<div class="group">
  <div class="head">{{.Size}} · {{.Copies}} · <span class="muted">{{.ID}}</span></div>
  <ul>{{range .Files}}<li{{if .Suggested}} class="keep"{{end}}>{{.Path}} <span class="muted">({{.MTime}})</span></li>{{end}}</ul>
</div>
{{end}}
{{end}}

{{if .TopFiles}}
<h2>{{.L.topFiles}}</h2>
<table><tr><th class="num">{{.L.size}}</th><th>{{.L.path}}</th></tr>
{{range .TopFiles}}<tr><td class="num">{{.Size}}</td><td class="path">{{.Path}}</td></tr>{{end}}
</table>
{{end}}

{{if .TopDirs}}
<h2>{{.L.topDirs}}</h2>
<table><tr><th class="num">{{.L.size}}</th><th>{{.L.path}}</th><th class="num">{{.L.count}}</th></tr>
{{range .TopDirs}}<tr><td class="num">{{.Size}}</td><td class="path">{{.Path}}</td><td class="num">{{.Tag}}</td></tr>{{end}}
</table>
{{end}}

{{if .Errors}}
<h2>{{.L.errors}} <span class="muted">({{len .Errors}})</span></h2>
<table><tr><th>{{.L.path}}</th><th>{{.L.error}}</th></tr>
{{range .Errors}}<tr><td class="path">{{.Path}}</td><td>{{.Err}}</td></tr>{{end}}
</table>
{{end}}

{{if .Skipped}}
<h2>{{.L.skipped}} <span class="muted">({{len .Skipped}})</span></h2>
<ul class="roots">{{range .Skipped}}<li>{{.}}</li>{{end}}</ul>
{{end}}
</body>
</html>
`))
