package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
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
		out     = fs.String("out", "report.json", "where to write the JSON report")
		lang    = fs.String("lang", "", "console language: ru or en (default: from the OS locale)")
		topN    = fs.Int("top", 0, "override top-N for the largest files / heaviest folders")
		quiet   = fs.Bool("quiet", false, "do not print the summary, only write the report")
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
	*out = filepath.Clean(*out)

	cfg := config.Default()
	if p, ok := config.Discover(*cfgPath, roots); ok {
		var err error
		if cfg, err = config.Load(p); err != nil {
			fmt.Fprintln(os.Stderr, "config:", err)
			return exitError
		}
	}
	cfg.Exclude = append(cfg.Exclude, exclude...)
	if *topN > 0 {
		cfg.TopN = *topN
	}

	res, err := scan.Walk(roots, scan.Options{Exclude: cfg.Exclude})
	if err != nil {
		fmt.Fprintln(os.Stderr, "scan:", err)
		return exitError
	}
	findings := detect.Run(res, cfg)
	rep := report.Build(res, findings, cfg, version)

	if err := rep.WriteJSON(*out); err != nil {
		fmt.Fprintln(os.Stderr, "report:", err)
		return exitError
	}
	if !*quiet {
		rep.PrintConsole(os.Stdout, report.ConsoleOptions{ReportPath: *out})
	}
	if rep.Stats.Errors > 0 {
		return exitError
	}
	return exitOK
}
