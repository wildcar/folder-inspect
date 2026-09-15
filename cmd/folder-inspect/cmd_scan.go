package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/export"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
	"github.com/wildcar/folder-inspect/internal/scan"
)

// multiFlag collects a repeatable string flag.
type multiFlag []string

func (m *multiFlag) String() string     { return fmt.Sprint([]string(*m)) }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func runScan(args []string) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		cfgPath = fs.String("config", "", "config file (default: .folder-inspect.yml in the first folder, then in the home folder)")
		out     = fs.String("out", "", "where to write the JSON report (default: <first folder>/.folder-inspect/reports/report-<date>_<time>.json)")
		exports = fs.String("export", "", "also write these formats next to the report: csv, xlsx, html (comma-separated)")
		force   = fs.Bool("force", false, "overwrite existing report/export files")
		lang    = fs.String("lang", "", "console language: ru or en (default: from the OS locale)")
		topN    = fs.Int("top", 0, "override top-N for the largest files / heaviest folders")
		noDups  = fs.Bool("no-dups", false, "skip duplicate detection (no file contents are read)")
		quiet   = fs.Bool("quiet", false, "do not print the summary, only write the files")
		exclude multiFlag
	)
	fs.Var(&exclude, "exclude", "glob pattern to skip (repeatable); a pattern with '/' matches the relative path")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: folder-inspect scan [options] <folder> [<folder>...]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	roots := fs.Args()
	if len(roots) == 0 {
		fs.Usage()
		return exitUsage
	}
	if *lang != "" {
		i18n.Set(*lang)
	}

	// Decide every output path up front so a long scan never ends in a
	// refused write.
	outPath := *out
	if outPath == "" {
		outPath = defaultReportPath(roots[0])
	}
	outPath = filepath.Clean(outPath)
	if err := report.CheckOverwrite(outPath, *force); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitError
	}
	formats, err := export.ParseList(*exports)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitUsage
	}
	exportPaths := map[string]string{}
	for _, f := range formats {
		p := report.SiblingPath(outPath, f)
		if err := report.CheckOverwrite(p, *force); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return exitError
		}
		exportPaths[f] = p
	}

	cfg := config.Default()
	if p, ok := config.Discover(*cfgPath, roots); ok {
		if cfg, err = config.Load(p); err != nil {
			fmt.Fprintln(os.Stderr, "config:", err)
			return exitError
		}
	}
	cfg.Exclude = append(cfg.Exclude, exclude...)
	if *topN > 0 {
		cfg.TopN = *topN
	}
	if *noDups {
		cfg.Duplicates.Enabled = false
	}

	res, err := scan.Walk(roots, scan.Options{Exclude: cfg.Exclude})
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
		return exitError
	}
	findings := detect.Run(res, cfg)
	var dups detect.DupResult
	var dirs detect.DirDupResult
	if cfg.Duplicates.Enabled {
		dups = detect.Duplicates(res.Files, detect.DupOptions{MinSize: int64(cfg.Duplicates.MinSize)})
		findings = append(findings, dups.Findings()...)
		if cfg.FolderDuplicates.Enabled {
			dirs = detect.DuplicateDirs(res, dups, detect.DirDupOptions{
				MinOverlap: cfg.FolderDuplicates.MinOverlap, MinFiles: cfg.FolderDuplicates.MinFiles,
			})
			findings = append(findings, dirs.Findings()...)
		}
		detect.Sort(findings)
		res.Finished = time.Now() // the scan includes hashing
	}
	rep := report.Build(res, findings, dups, dirs, cfg, version)

	if err := rep.WriteJSON(outPath); err != nil {
		fmt.Fprintln(os.Stderr, "report:", err)
		return exitError
	}
	var written []string
	for _, f := range formats {
		if err := export.WriteFile(rep, f, exportPaths[f]); err != nil {
			fmt.Fprintf(os.Stderr, "export %s: %v\n", f, err)
			return exitError
		}
		written = append(written, exportPaths[f])
	}
	if !*quiet {
		rep.PrintConsole(os.Stdout, report.ConsoleOptions{ReportPath: outPath, Exports: written})
		fmt.Println(i18n.Tf("scan.ui_hint", outPath))
	}
	if rep.Stats.Errors > 0 {
		return exitError
	}
	return exitOK
}

// defaultReportPath is <root>/.folder-inspect/reports/report-<ts>.json;
// when that folder cannot be created (read-only share) the report goes to
// the current folder instead, with a notice.
func defaultReportPath(root string) string {
	dir := report.DefaultDir(root)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, i18n.Tf("scan.fallback_dir", dir, err))
		return report.UniquePath(report.DefaultName(time.Now()))
	}
	return report.UniquePath(filepath.Join(dir, report.DefaultName(time.Now())))
}
