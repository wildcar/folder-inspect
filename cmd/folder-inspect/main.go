// Command folder-inspect inspects a project document repository for
// oversized files, archives, distributives, junk, empty folders and (later)
// duplicates. See AGENTS/SPEC.md for the contract.
package main

import (
	"fmt"
	"os"

	"github.com/wildcar/folder-inspect/internal/i18n"
)

// version is overridden at build time: -ldflags "-X main.version=1.2.3".
var version = "0.1.0-dev"

const (
	exitOK    = 0
	exitError = 1 // scan finished but some paths were unreadable, or a command failed
	exitUsage = 2
)

func usage() {
	fmt.Fprintf(os.Stderr, `folder-inspect %s

Usage:
  folder-inspect scan [options] <folder> [<folder>...]   scan folders, write <folder>/.folder-inspect/reports/report-<date>.json, print a summary
  folder-inspect ui [options] <report.json | folder>     open the report in the browser (newest report of a folder)
  folder-inspect report [options] <report.json>          re-print a saved report or export it (csv, xlsx, html)
  folder-inspect plan [options] <report.json | folder>   build a plan from rules (-select junk,archive -duplicates oldest) without the UI
  folder-inspect apply [options] <plan.json>             move planned items to quarantine, leave stubs (-dry-run to preview)
  folder-inspect restore [options] <manifest | folder>   bring a quarantine batch back
  folder-inspect quarantine list|show|purge ...          inspect quarantine batches; purge deletes for good (-yes)
  folder-inspect fixture [options] <folder>              generate a demo "dirty repository" for trying the tool
  folder-inspect version                                 print the version

Run "folder-inspect <command> -h" for the options of a command.
`, version)
}

func main() {
	i18n.Set(i18n.Detect())
	if len(os.Args) < 2 {
		usage()
		os.Exit(exitUsage)
	}
	var code int
	switch os.Args[1] {
	case "scan":
		code = runScan(os.Args[2:])
	case "report":
		code = runReport(os.Args[2:])
	case "ui":
		code = runUI(os.Args[2:])
	case "plan":
		code = runPlan(os.Args[2:])
	case "apply":
		code = runApply(os.Args[2:])
	case "restore":
		code = runRestore(os.Args[2:])
	case "quarantine":
		code = runQuarantine(os.Args[2:])
	case "fixture":
		code = runFixture(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Println("folder-inspect", version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		code = exitUsage
	}
	os.Exit(code)
}
