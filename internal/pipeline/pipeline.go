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
	var names detect.NameResult
	if cfg.Duplicates.Enabled {
		dups = detect.Duplicates(res.Files, detect.DupOptions{MinSize: int64(cfg.Duplicates.MinSize)})
		findings = append(findings, dups.Findings()...)
		if cfg.FolderDuplicates.Enabled {
			dirs = detect.DuplicateDirs(res, dups, detect.DirDupOptions{
				MinOverlap: cfg.FolderDuplicates.MinOverlap, MinFiles: cfg.FolderDuplicates.MinFiles,
			})
			findings = append(findings, dirs.Findings()...)
		}
		res.Finished = time.Now() // the scan includes hashing
	}
	if cfg.SimilarNames.Enabled {
		names = detect.SimilarNames(res.Files, dups)
		findings = append(findings, names.Findings()...)
	}
	detect.Sort(findings)
	return report.Build(res, findings, dups, dirs, names, cfg, version), nil
}
