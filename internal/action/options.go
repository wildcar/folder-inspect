package action

import (
	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
)

// OptionsFromConfig maps the quarantine section of a config onto
// ApplyOptions. findings (by path) may be nil; it only enriches stubs.
func OptionsFromConfig(cfg *config.Config, lang, planPath string, findings map[string]detect.Finding) ApplyOptions {
	if cfg == nil {
		cfg = config.Default()
	}
	cats := make(map[string]bool, len(cfg.Quarantine.StubCategories))
	for _, c := range cfg.Quarantine.StubCategories {
		cats[c] = true
	}
	videos := make(map[string]bool, len(cfg.Videos))
	for _, v := range cfg.Videos {
		videos[v] = true
	}
	return ApplyOptions{
		QuarantineDir:  cfg.Quarantine.Dir,
		Stubs:          cfg.Quarantine.Stubs,
		StubCategories: cats,
		StubTexts:      cfg.Quarantine.StubTexts,
		Videos:         videos,
		Lang:           lang,
		PlanPath:       planPath,
		Findings:       findings,
	}
}

// FindingsByPath indexes a report's findings for OptionsFromConfig.
func FindingsByPath(findings []detect.Finding) map[string]detect.Finding {
	m := make(map[string]detect.Finding, len(findings))
	for _, f := range findings {
		m[f.Path] = f
	}
	return m
}
