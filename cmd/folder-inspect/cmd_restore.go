package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/wildcar/folder-inspect/internal/action"
	"github.com/wildcar/folder-inspect/internal/i18n"
)

// runRestore brings a quarantine batch back: every entry of the manifest
// returns to where it was and its stub is removed.
func runRestore(args []string) int {
	fs := flag.NewFlagSet("restore", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dryRun = fs.Bool("dry-run", false, "only show what would be restored; change nothing")
		lang   = fs.String("lang", "", "language of messages: ru or en (default: from the OS locale)")
	)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: folder-inspect restore [options] <manifest.json | quarantine batch folder | scanned folder>")
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
	path, err := action.FindManifest(fs.Arg(0), "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitError
	}
	m, err := action.LoadManifest(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "manifest:", err)
		return exitError
	}
	fmt.Println(i18n.Tf("restore.header", path, len(m.Entries), m.Dir))
	res, err := action.Restore(m, action.RestoreOptions{DryRun: *dryRun})
	if err != nil {
		fmt.Fprintln(os.Stderr, "restore:", err)
		return exitError
	}
	for _, e := range m.Entries {
		mark := " "
		if e.Restored {
			mark = "+"
		}
		fmt.Printf("  %s %s\n", mark, e.From)
	}
	if len(res.Problems) > 0 {
		fmt.Println(i18n.T("apply.problems"))
		for _, pr := range res.Problems {
			fmt.Printf("  ! %s: %s\n", pr.Path, pr.Err)
		}
	}
	fmt.Println(i18n.Tf("restore.summary", res.Restored, len(res.Problems)))
	if *dryRun {
		fmt.Println(i18n.T("restore.dry"))
	}
	if len(res.Problems) > 0 {
		return exitError
	}
	return exitOK
}
