package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wildcar/folder-inspect/internal/action"
	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
)

// runApply carries out a saved plan: moves the planned items into the
// quarantine folder of their root and leaves stubs. Dry-run by flag.
func runApply(args []string) int {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dryRun  = fs.Bool("dry-run", false, "only show what would be moved; change nothing")
		noStubs = fs.Bool("no-stubs", false, "do not leave <name>.removed.txt files")
		lang    = fs.String("lang", "", "language of messages and stubs: ru or en (default: from the OS locale)")
		cfgPath = fs.String("config", "", "config file for quarantine settings (default: the config recorded in the plan's report)")
	)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: folder-inspect apply [options] <plan.json>")
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
	planPath, _ := filepath.Abs(fs.Arg(0))
	p, err := action.Load(planPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "plan:", err)
		return exitError
	}

	// Config: explicit file, else the effective config stored in the report
	// the plan came from, else defaults.
	cfg := config.Default()
	var findings map[string]detect.Finding
	if rep, err := report.ReadJSON(p.Report); err == nil {
		if rep.Config != nil {
			cfg = rep.Config
		}
		findings = action.FindingsByPath(rep.Findings)
	}
	if *cfgPath != "" {
		if cfg, err = config.Load(*cfgPath); err != nil {
			fmt.Fprintln(os.Stderr, "config:", err)
			return exitError
		}
	}
	opt := action.OptionsFromConfig(cfg, i18n.Lang(), planPath, findings)
	opt.DryRun = *dryRun
	if *noStubs {
		opt.Stubs = false
	}

	fmt.Println(i18n.Tf("apply.header", planPath, len(p.Actions), report.HumanSize(p.TotalSize())))
	res, err := action.Apply(p, opt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "apply:", err)
		return exitError
	}
	for _, m := range res.Manifests {
		for _, e := range m.Entries {
			fmt.Printf("  %-28s %s\n", i18n.T("op."+string(e.Op)), e.From)
			fmt.Printf("  %-28s -> %s\n", "", e.To)
			if e.Stub != "" {
				fmt.Printf("  %-28s +  %s\n", "", filepath.Base(e.Stub))
			}
		}
	}
	fmt.Println()
	if len(res.Problems) > 0 {
		fmt.Println(i18n.T("apply.problems"))
		for _, pr := range res.Problems {
			fmt.Printf("  ! %s: %s\n", pr.Path, pr.Err)
		}
		fmt.Println()
	}
	fmt.Println(i18n.Tf("apply.summary", res.Moved, res.Stubs, len(res.Problems)))
	if *dryRun {
		fmt.Println(i18n.T("apply.dry"))
	} else {
		for _, m := range res.Manifests {
			if len(m.Entries) > 0 {
				fmt.Println(i18n.Tf("apply.manifest", m.Path))
				fmt.Println(i18n.Tf("apply.restore_hint", m.Path))
			}
		}
	}
	if len(res.Problems) > 0 {
		return exitError
	}
	return exitOK
}
