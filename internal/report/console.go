package report

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/i18n"
)

// ConsoleOptions tunes the console summary.
type ConsoleOptions struct {
	PerCategory int      // findings (or duplicate groups) listed per category (0 = 10)
	TopN        int      // top files / folders listed (0 = 10)
	ReportPath  string   // printed at the end when not empty
	Exports     []string // export files written, printed at the end
}

// HumanSize formats bytes in binary units with localized unit names.
func HumanSize(n int64) string {
	b := config.ByteSize(n)
	var v float64
	var unit string
	switch {
	case b >= config.TB:
		v, unit = float64(b)/float64(config.TB), "TB"
	case b >= config.GB:
		v, unit = float64(b)/float64(config.GB), "GB"
	case b >= config.MB:
		v, unit = float64(b)/float64(config.MB), "MB"
	case b >= config.KB:
		v, unit = float64(b)/float64(config.KB), "KB"
	default:
		return fmt.Sprintf("%d %s", n, i18n.T("unit.B"))
	}
	return fmt.Sprintf("%.1f %s", v, i18n.T("unit."+unit))
}

// PrintConsole writes the human summary of the report.
func (r *Report) PrintConsole(w io.Writer, opt ConsoleOptions) {
	if opt.PerCategory <= 0 {
		opt.PerCategory = 10
	}
	if opt.TopN <= 0 {
		opt.TopN = 10
	}
	fmt.Fprintln(w, i18n.Tf("scan.header", r.Tool, r.Version, strings.Join(r.Roots, ", ")))
	fmt.Fprintln(w, i18n.Tf("scan.roots", len(r.Roots), r.Stats.Files, r.Stats.Dirs,
		HumanSize(r.Stats.TotalSize), r.Duration().Round(1e6)))
	if r.Stats.Hashed > 0 {
		fmt.Fprintln(w, i18n.Tf("scan.hashed", r.Stats.Hashed, HumanSize(r.Stats.HashedBytes)))
	}
	fmt.Fprintln(w)

	if len(r.Summary) == 0 {
		fmt.Fprintln(w, i18n.T("scan.nothing"))
	} else {
		fmt.Fprintln(w, i18n.T("scan.summary"))
		tw := tabwriter.NewWriter(w, 2, 4, 2, ' ', tabwriter.AlignRight)
		fmt.Fprintf(tw, "  %s\t%s\t%s\t\n", i18n.T("scan.category"), i18n.T("scan.count"), i18n.T("scan.size"))
		for _, s := range r.Summary {
			fmt.Fprintf(tw, "  %s\t%d\t%s\t\n", CategoryName(s.Category), s.Count, HumanSize(s.Size))
		}
		tw.Flush()
		fmt.Fprintln(w)

		for _, c := range detect.Categories {
			if c == detect.Duplicate {
				r.printDuplicates(w, opt.PerCategory)
				continue
			}
			r.printCategory(w, c, opt.PerCategory)
		}
	}

	printTop(w, i18n.T("scan.top_files"), r.TopFiles, opt.TopN)
	printTop(w, i18n.T("scan.top_dirs"), r.TopDirs, opt.TopN)

	if r.Stats.Errors > 0 {
		fmt.Fprintln(w, i18n.Tf("scan.errors", r.Stats.Errors))
		for i, e := range r.Errors {
			if i >= opt.PerCategory {
				fmt.Fprintln(w, "  "+i18n.Tf("scan.more", len(r.Errors)-i))
				break
			}
			fmt.Fprintf(w, "  %s: %s\n", e.Path, e.Err)
		}
	}
	if r.Stats.Skipped > 0 {
		fmt.Fprintln(w, i18n.Tf("scan.skipped", r.Stats.Skipped))
	}
	if opt.ReportPath != "" || len(opt.Exports) > 0 {
		fmt.Fprintln(w)
	}
	if opt.ReportPath != "" {
		fmt.Fprintln(w, i18n.Tf("scan.written", opt.ReportPath))
	}
	for _, e := range opt.Exports {
		fmt.Fprintln(w, i18n.Tf("scan.exported", e))
	}
}

func (r *Report) printCategory(w io.Writer, c detect.Category, limit int) {
	items := r.ByCategory(c)
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(w, "%s (%d)\n", CategoryName(c), len(items))
	for i, f := range items {
		if i >= limit {
			fmt.Fprintln(w, "  "+i18n.Tf("scan.more", len(items)-i))
			break
		}
		tag := Qualifier(f)
		if tag != "" {
			tag = "  [" + tag + "]"
		}
		fmt.Fprintf(w, "  %10s  %s%s\n", HumanSize(f.Size), displayPath(f.Rel, f.Path), tag)
	}
	fmt.Fprintln(w)
}

func (r *Report) printDuplicates(w io.Writer, limit int) {
	if len(r.Duplicates) == 0 {
		return
	}
	var wasted int64
	for _, g := range r.Duplicates {
		wasted += g.Wasted
	}
	fmt.Fprintln(w, i18n.Tf("dup.header", CategoryName(detect.Duplicate), len(r.Duplicates), HumanSize(wasted)))
	fmt.Fprintln(w, "  "+i18n.T("dup.legend"))
	for i, g := range r.Duplicates {
		if i >= limit {
			fmt.Fprintln(w, "  "+i18n.Tf("scan.more", len(r.Duplicates)-i))
			break
		}
		fmt.Fprintf(w, "  %10s  %s  [%s]\n", HumanSize(g.Size), i18n.Tf("dup.copies", g.Count), g.ID)
		for _, f := range g.Files {
			mark := " "
			if f.Path == g.Suggested {
				mark = "*"
			}
			fmt.Fprintf(w, "            %s %s  (%s)\n", mark, displayPath(f.Rel, f.Path), FormatTime(f.ModTime))
		}
	}
	fmt.Fprintln(w)
}

func printTop(w io.Writer, title string, items []Item, limit int) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintln(w, title)
	for i, it := range items {
		if i >= limit {
			break
		}
		fmt.Fprintf(w, "  %10s  %s\n", HumanSize(it.Size), displayPath(it.Rel, it.Path))
	}
	fmt.Fprintln(w)
}

// displayPath shows the path relative to its root; the roots are printed
// in the header. A root itself has no relative path and is shown in full.
func displayPath(rel, path string) string {
	if rel == "" {
		return path
	}
	return rel
}
