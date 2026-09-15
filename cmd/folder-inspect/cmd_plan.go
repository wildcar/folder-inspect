package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wildcar/folder-inspect/internal/action"
	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
)

// runPlan builds a plan from rules instead of ticks in the UI: whole
// categories ("all junk"), duplicate groups with an explicit keep policy,
// path and size filters. The plan is saved next to the report and applied
// with `apply` as usual; -dry-run only prints it.
func runPlan(args []string) int {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		sel     = fs.String("select", "", "categories to quarantine, comma-separated: junk, archive, distributive, oversize, empty-dir, empty-file, or all")
		dups    = fs.String("duplicates", "", "quarantine duplicate files, keeping one copy per group: oldest, newest or shallowest")
		dirDups = fs.String("dir-duplicates", "", "quarantine identical folders, keeping one per group: oldest, newest or shallowest")
		include = fs.String("include", "", "only paths matching these globs (comma-separated, case-insensitive, e.g. \"*.mp4,Архив/*\")")
		exclude = fs.String("exclude", "", "skip paths matching these globs")
		minSize = fs.String("min-size", "", "skip items smaller than this (e.g. 10MB)")
		out     = fs.String("out", "", "plan file (default: plan-<date>.json next to the report)")
		force   = fs.Bool("force", false, "overwrite an existing -out file")
		dryRun  = fs.Bool("dry-run", false, "print the plan, do not save it")
		quiet   = fs.Bool("quiet", false, "print only the saved plan path (for scripts)")
		lang    = fs.String("lang", "", "language of messages: ru or en (default: from the OS locale)")
	)
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: folder-inspect plan [options] <report.json | scanned folder>")
		fmt.Fprintln(os.Stderr, "Example: folder-inspect plan -select junk,archive -duplicates oldest D:\\Projects")
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

	rules := action.Rules{Include: splitList(*include), Exclude: splitList(*exclude)}
	var err error
	if rules.Categories, err = action.ParseCategories(*sel); err != nil {
		fmt.Fprintln(os.Stderr, "-select:", err)
		return exitUsage
	}
	if rules.Duplicates, err = action.ParseKeepPolicy(*dups); err != nil {
		fmt.Fprintln(os.Stderr, "-duplicates:", err)
		return exitUsage
	}
	if rules.DirDuplicates, err = action.ParseKeepPolicy(*dirDups); err != nil {
		fmt.Fprintln(os.Stderr, "-dir-duplicates:", err)
		return exitUsage
	}
	if *minSize != "" {
		n, err := config.ParseByteSize(*minSize)
		if err != nil {
			fmt.Fprintln(os.Stderr, "-min-size:", err)
			return exitUsage
		}
		rules.MinSize = int64(n)
	}
	if len(rules.Categories) == 0 && rules.Duplicates == "" && rules.DirDuplicates == "" {
		fs.Usage()
		return exitUsage
	}

	reportPath, err := resolveReport(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return exitError
	}
	rep, err := report.ReadJSON(reportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "report:", err)
		return exitError
	}
	var outPath string
	if *out != "" {
		if outPath, err = filepath.Abs(*out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return exitError
		}
		if err := report.CheckOverwrite(outPath, *force); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return exitError
		}
	}

	p, st, err := action.FromRules(rep, reportPath, rules)
	if err != nil {
		fmt.Fprintln(os.Stderr, "plan:", err)
		return exitError
	}

	if !*quiet {
		fmt.Println(i18n.Tf("plan.header", reportPath, len(rep.Roots)))
		fmt.Println(i18n.Tf("plan.rules", categoryList(rules.Categories), policyName(rules.Duplicates), policyName(rules.DirDuplicates)))
		if len(rules.Include) > 0 || len(rules.Exclude) > 0 || rules.MinSize > 0 {
			fmt.Println(i18n.Tf("plan.filters", strings.Join(rules.Include, ", "), strings.Join(rules.Exclude, ", "), report.HumanSize(rules.MinSize)))
		}
		fmt.Println()
		for _, a := range p.Actions {
			fmt.Printf("  %-28s %s\n", i18n.T("op."+string(a.Op)), a.Path)
			if a.Original != "" {
				fmt.Printf("  %-28s %s\n", "", i18n.Tf("plan.original", a.Original))
			}
		}
		if len(p.Actions) > 0 {
			fmt.Println()
		}
		fmt.Println(i18n.Tf("plan.summary", len(p.Actions), report.HumanSize(p.TotalSize()), st.Findings, st.Duplicates, st.DirDuplicates, st.Skipped, st.GroupsSkipped))
	}
	if len(p.Actions) == 0 {
		if !*quiet {
			fmt.Println(i18n.T("plan.empty"))
		}
		return exitOK
	}
	if *dryRun {
		if !*quiet {
			fmt.Println(i18n.T("plan.dry"))
		}
		return exitOK
	}

	var saved string
	if outPath != "" {
		saved, err = p.SaveAs(outPath)
	} else {
		saved, err = p.Save(filepath.Dir(reportPath))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "plan:", err)
		return exitError
	}
	if *quiet {
		fmt.Println(saved)
		return exitOK
	}
	fmt.Println(i18n.Tf("plan.saved", saved))
	fmt.Println(i18n.Tf("plan.apply_hint", saved, saved))
	return exitOK
}

func splitList(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func categoryList(cs []detect.Category) string {
	if len(cs) == 0 {
		return i18n.T("plan.off")
	}
	names := make([]string, len(cs))
	for i, c := range cs {
		names[i] = report.CategoryName(c)
	}
	return strings.Join(names, ", ")
}

func policyName(p action.KeepPolicy) string {
	if p == "" {
		return i18n.T("plan.off")
	}
	return i18n.T("plan.keep." + string(p))
}
