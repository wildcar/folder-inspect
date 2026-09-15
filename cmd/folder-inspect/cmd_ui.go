package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
	"github.com/wildcar/folder-inspect/internal/ui"
)

// runUI serves a report in the browser. The argument is a report.json or a
// scanned folder (then the newest report in <folder>/.folder-inspect/reports).
func runUI(args []string) int {
	fs := flag.NewFlagSet("ui", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		port      = fs.Int("port", 0, "port on 127.0.0.1 (default: a free one)")
		noBrowser = fs.Bool("no-browser", false, "do not open the browser, only print the address")
		lang      = fs.String("lang", "", "interface language: ru or en (default: from the OS locale)")
	)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: folder-inspect ui [options] <report.json | scanned folder>")
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
	path, err := resolveReport(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitError
	}
	rep, err := report.ReadJSON(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "report:", err)
		return exitError
	}
	srv := &ui.Server{Report: rep, ReportPath: path, PlanDir: filepath.Dir(path), Version: version}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := ui.Serve(ctx, srv, fmt.Sprintf("127.0.0.1:%d", *port), !*noBrowser, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "ui:", err)
		return exitError
	}
	return exitOK
}

// resolveReport turns the argument into an absolute report path.
func resolveReport(arg string) (string, error) {
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return abs, nil
	}
	return report.NewestReport(report.DefaultDir(abs))
}
