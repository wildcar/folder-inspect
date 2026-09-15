package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wildcar/folder-inspect/internal/export"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
)

// runReport re-renders a saved report.json: console summary by default, or
// an export file. No scanning happens here.
func runReport(args []string) int {
	fs := flag.NewFlagSet("report", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		format = fs.String("format", "console", "console, csv, xlsx or html")
		out    = fs.String("out", "", "output file for csv/xlsx/html (default: next to the report, same name)")
		force  = fs.Bool("force", false, "overwrite an existing output file")
		lang   = fs.String("lang", "", "language: ru or en (default: from the OS locale)")
		all    = fs.Bool("all", false, "console: list every finding instead of the first 10 per category")
	)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: folder-inspect report [options] <report.json>")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return exitUsage
	}
	if *lang != "" {
		i18n.Set(*lang)
	}
	rep, err := report.ReadJSON(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "report:", err)
		return exitError
	}

	if *format == "console" {
		opt := report.ConsoleOptions{}
		if *all {
			opt.PerCategory = len(rep.Findings) + len(rep.Duplicates) + 1
			opt.TopN = len(rep.TopFiles) + len(rep.TopDirs) + 1
		}
		rep.PrintConsole(os.Stdout, opt)
		return exitOK
	}
	if !export.Valid(*format) {
		fmt.Fprintf(os.Stderr, "unknown format %q\n", *format)
		return exitUsage
	}
	outPath := *out
	if outPath == "" {
		outPath = report.SiblingPath(fs.Arg(0), *format)
	}
	outPath = filepath.Clean(outPath)
	if err := report.CheckOverwrite(outPath, *force); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitError
	}
	if err := export.WriteFile(rep, *format, outPath); err != nil {
		fmt.Fprintln(os.Stderr, "export:", err)
		return exitError
	}
	fmt.Println(i18n.Tf("scan.exported", outPath))
	return exitOK
}
