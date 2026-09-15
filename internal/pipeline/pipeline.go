// Package pipeline runs one full scan: walk → detectors → duplicates →
// folder duplicates → report. Shared by the scan command and the web UI's
// rescan so both produce identical reports.
package pipeline

import (
	"time"

	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/report"
	"github.com/wildcar/folder-inspect/internal/scan"
)

// Run scans roots with cfg and builds the report. It fails only when a root
// is unusable; per-path problems are inside the report.
func Run(roots []string, cfg *config.Config, version string) (*report.Report, error) {
	res, err := scan.Walk(roots, scan.Options{Exclude: cfg.Exclude})
	if err != nil {
		return nil, err
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
	return report.Build(res, findings, dups, dirs, cfg, version), nil
}
