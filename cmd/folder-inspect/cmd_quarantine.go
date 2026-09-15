package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wildcar/folder-inspect/internal/action"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
)

func quarantineUsage() {
	fmt.Fprintln(os.Stderr, `Usage:
  folder-inspect quarantine list  <scanned folder>                       batches, newest first
  folder-inspect quarantine show  <manifest | batch folder | folder>      entries of one batch
  folder-inspect quarantine purge [-yes] [-dry-run] <manifest | batch folder | folder>
                                  delete the quarantined copies for good (needs -yes)`)
}

// runQuarantine manages quarantine batches: list, show, purge.
func runQuarantine(args []string) int {
	if len(args) == 0 {
		quarantineUsage()
		return exitUsage
	}
	switch args[0] {
	case "list":
		return quarantineList(args[1:])
	case "show":
		return quarantineShow(args[1:])
	case "purge":
		return quarantinePurge(args[1:])
	}
	fmt.Fprintf(os.Stderr, "unknown quarantine command %q\n\n", args[0])
	quarantineUsage()
	return exitUsage
}

func quarantineList(args []string) int {
	fs := flag.NewFlagSet("quarantine list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	lang := fs.String("lang", "", "ru or en")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		quarantineUsage()
		return exitUsage
	}
	if *lang != "" {
		i18n.Set(*lang)
	}
	root, _ := filepath.Abs(fs.Arg(0))
	batches, err := action.ListBatches(root, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitError
	}
	fmt.Println(i18n.Tf("q.list_header", root, len(batches)))
	if len(batches) == 0 {
		fmt.Println(i18n.T("q.none"))
		return exitOK
	}
	for _, b := range batches {
		fmt.Printf("  %s  %-22s %3d / %-3d %10s  %s\n", b.Manifest.Created.Format("2006-01-02 15:04"),
			i18n.T("q.status."+b.Status), b.Pending, b.Items, report.HumanSize(b.Size), b.Manifest.Dir)
	}
	return exitOK
}

func quarantineShow(args []string) int {
	fs := flag.NewFlagSet("quarantine show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	lang := fs.String("lang", "", "ru or en")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		quarantineUsage()
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
	b := action.Describe(m)
	fmt.Println(i18n.Tf("restore.header", path, len(m.Entries), m.Dir))
	fmt.Printf("  %s: %s, %s: %d, %s: %s\n", i18n.T("ui.q_status"), i18n.T("q.status."+b.Status), i18n.T("ui.q_pending"), b.Pending, i18n.T("col.size"), report.HumanSize(b.Size))
	for _, e := range m.Entries {
		mark := " "
		if e.Restored {
			mark = "+"
		}
		fmt.Printf("  %s %-28s %s\n", mark, i18n.T("op."+string(e.Op)), e.From)
	}
	return exitOK
}

func quarantinePurge(args []string) int {
	fs := flag.NewFlagSet("quarantine purge", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		yes    = fs.Bool("yes", false, "really delete; without it only a preview is printed")
		dryRun = fs.Bool("dry-run", false, "preview only")
		lang   = fs.String("lang", "", "ru or en")
	)
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		quarantineUsage()
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
	if m.Purged != nil {
		fmt.Fprintln(os.Stderr, i18n.Tf("q.already_purged", m.Dir))
		return exitError
	}
	preview, err := action.Purge(m, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitError
	}
	if preview.Deleted == 0 {
		fmt.Println(i18n.Tf("q.nothing_to_purge", m.Dir))
		return exitOK
	}
	fmt.Println(i18n.Tf("q.purge_preview", preview.Deleted, report.HumanSize(preview.Bytes), m.Dir))
	for _, e := range m.Entries {
		if !e.Restored {
			fmt.Printf("  - %s\n", e.To)
		}
	}
	if !*yes || *dryRun {
		fmt.Println(i18n.Tf("q.purge_hint", m.Dir))
		return exitOK
	}
	res, err := action.Purge(m, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitError
	}
	for _, pr := range res.Problems {
		fmt.Printf("  ! %s: %s\n", pr.Path, pr.Err)
	}
	fmt.Println(i18n.Tf("q.purged", res.Deleted, report.HumanSize(res.Bytes)))
	if len(res.Problems) > 0 {
		return exitError
	}
	return exitOK
}
