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
func HumanSize(n int64) string { return HumanSizeLang(n, i18n.Lang()) }

// HumanSizeLang is HumanSize in an explicit language.
func HumanSizeLang(n int64, lang string) string {
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
		return fmt.Sprintf("%d %s", n, i18n.TL(lang, "unit.B"))
	}
	return fmt.Sprintf("%.1f %s", v, i18n.TL(lang, "unit."+unit))
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
			switch c {
			case detect.Duplicate:
				r.printDuplicates(w, opt.PerCategory)
			case detect.DirDuplicate:
				r.printDirDuplicates(w, opt.PerCategory)
			case detect.DirOverlap:
				r.printOverlaps(w, opt.PerCategory)
			case detect.SimilarName:
				r.printSimilarNames(w, opt.PerCategory)
			default:
				r.printCategory(w, c, opt.PerCategory)
			}
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

func (r *Report) printDirDuplicates(w io.Writer, limit int) {
	if len(r.DirDuplicates) == 0 {
		return
	}
	var wasted int64
	for _, g := range r.DirDuplicates {
		wasted += g.Wasted
	}
	fmt.Fprintln(w, i18n.Tf("dup.header", CategoryName(detect.DirDuplicate), len(r.DirDuplicates), HumanSize(wasted)))
	fmt.Fprintln(w, "  "+i18n.T("dup.legend"))
	for i, g := range r.DirDuplicates {
		if i >= limit {
			fmt.Fprintln(w, "  "+i18n.Tf("scan.more", len(r.DirDuplicates)-i))
			break
		}
		fmt.Fprintf(w, "  %10s  %s, %s  [%s]\n", HumanSize(g.Size), i18n.Tf("dirdup.folders", g.Count), i18n.Tf("dirdup.files", g.Files), g.ID)
		for _, d := range g.Dirs {
			mark := " "
			if d.Path == g.Suggested {
				mark = "*"
			}
			fmt.Fprintf(w, "            %s %s  (%s)\n", mark, displayPath(d.Rel, d.Path), FormatTime(d.ModTime))
		}
	}
	fmt.Fprintln(w)
}

func (r *Report) printOverlaps(w io.Writer, limit int) {
	if len(r.DirOverlaps) == 0 {
		return
	}
	var shared int64
	for _, o := range r.DirOverlaps {
		shared += o.SharedBytes
	}
	fmt.Fprintln(w, i18n.Tf("overlap.header", CategoryName(detect.DirOverlap), len(r.DirOverlaps), HumanSize(shared)))
	for i, o := range r.DirOverlaps {
		if i >= limit {
			fmt.Fprintln(w, "  "+i18n.Tf("scan.more", len(r.DirOverlaps)-i))
			break
		}
		fmt.Fprintf(w, "  %10s  %s  <->  %s\n", HumanSize(o.SharedBytes), displayPath(o.A.Rel, o.A.Path), displayPath(o.B.Rel, o.B.Path))
		fmt.Fprintf(w, "              %s (%s / %s)\n", OverlapLine(o), Percent(o.RatioA), Percent(o.RatioB))
	}
	fmt.Fprintln(w)
}

func (r *Report) printSimilarNames(w io.Writer, limit int) {
	if len(r.SimilarNames) == 0 {
		return
	}
	var variants int
	var size int64
	for _, g := range r.SimilarNames {
		variants += g.Variants
		size += g.Size
	}
	fmt.Fprintln(w, i18n.Tf("names.header", CategoryName(detect.SimilarName), len(r.SimilarNames), variants, HumanSize(size)))
	fmt.Fprintln(w, "  "+i18n.T("names.legend"))
	for i, g := range r.SimilarNames {
		if i >= limit {
			fmt.Fprintln(w, "  "+i18n.Tf("scan.more", len(r.SimilarNames)-i))
			break
		}
		fmt.Fprintf(w, "  %s  (%d)  [%s]\n", g.Name, g.Count, g.ID)
		for _, f := range g.Files {
			mark := " "
			tag := f.Marker
			if f.IsBase {
				mark = "*"
				tag = i18n.T("names.base")
			}
			if f.DupGroup != "" {
				tag += ", " + i18n.Tf("names.dup", f.DupGroup)
			}
			fmt.Fprintf(w, "    %s %10s  %s  [%s]  (%s)\n", mark, HumanSize(f.Size), displayPath(f.Rel, f.Path), tag, FormatTime(f.ModTime))
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
