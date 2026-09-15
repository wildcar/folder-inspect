// Package config holds the scan configuration: size rules graded by file
// kind, archive and distributive extensions, junk patterns and exclusions.
// Defaults live here; a YAML file (.folder-inspect.yml) overrides them
// field by field — a list given in the file replaces the default list.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileName is the config file looked up in the scanned root and the user's home.
const FileName = ".folder-inspect.yml"

// GenericExt marks a size rule that applies to every file kind.
const GenericExt = "*"

// SizeRule reports files of the listed kinds above a threshold.
type SizeRule struct {
	Name       string   `yaml:"name" json:"name"`
	Extensions []string `yaml:"extensions" json:"extensions"`
	Threshold  ByteSize `yaml:"threshold" json:"threshold"`
}

// IsGeneric reports whether the rule applies to all files.
func (r SizeRule) IsGeneric() bool {
	for _, e := range r.Extensions {
		if e == GenericExt {
			return true
		}
	}
	return false
}

// Config is the full scan configuration.
type Config struct {
	// Exclude holds glob patterns (see internal/glob) skipped during the walk.
	Exclude []string `yaml:"exclude" json:"exclude"`
	// SizeRules are evaluated most-specific first; see detect.Oversize.
	SizeRules []SizeRule `yaml:"size_rules" json:"size_rules"`
	// Archives and Distributives are extensions without the dot, lower case.
	Archives      []string `yaml:"archives" json:"archives"`
	Distributives []string `yaml:"distributives" json:"distributives"`
	// Junk holds glob patterns matched against file and folder names.
	Junk []string `yaml:"junk" json:"junk"`
	// TopN limits the largest-files / heaviest-folders lists in the report.
	TopN int `yaml:"top_n" json:"top_n"`
	// Duplicates tunes content-based duplicate detection.
	Duplicates DupConfig `yaml:"duplicates" json:"duplicates"`
	// FolderDuplicates tunes identical / overlapping folder detection.
	FolderDuplicates DirDupConfig `yaml:"folder_duplicates" json:"folder_duplicates"`
	// SimilarNames toggles the copy/version-by-name detector.
	SimilarNames SimilarNamesConfig `yaml:"similar_names" json:"similar_names"`
	// Videos are extensions that get the "video" explanation in stubs.
	Videos []string `yaml:"videos" json:"videos"`
	// Quarantine tunes apply: where files go and which stubs are left.
	Quarantine QuarantineConfig `yaml:"quarantine" json:"quarantine"`
}

// SimilarNamesConfig tunes the name-based copy detector.
type SimilarNamesConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// QuarantineConfig tunes the clean-up step.
type QuarantineConfig struct {
	// Dir is the quarantine folder relative to the scanned root.
	Dir string `yaml:"dir" json:"dir"`
	// Stubs enables the "<name>.removed.txt" files left where a file was.
	Stubs bool `yaml:"stubs" json:"stubs"`
	// StubCategories lists the finding categories that get a stub.
	StubCategories []string `yaml:"stub_categories" json:"stub_categories"`
	// StubTexts overrides the reason paragraph per category (plain text).
	StubTexts map[string]string `yaml:"stub_texts" json:"stub_texts,omitempty"`
	// DeleteCategories lists finding categories that apply deletes outright
	// instead of moving to quarantine (owner decision 2026-09-15: junk and
	// empty items are not worth a quarantine copy or a stub). Empty files and
	// folders are re-verified as empty first and can be recreated by restore.
	DeleteCategories []string `yaml:"delete_categories" json:"delete_categories"`
}

// StubFor reports whether a category gets a stub.
func (q QuarantineConfig) StubFor(category string) bool {
	if !q.Stubs {
		return false
	}
	for _, c := range q.StubCategories {
		if c == category {
			return true
		}
	}
	return false
}

// DupConfig tunes duplicate detection.
type DupConfig struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`   // hash candidates and report groups
	MinSize ByteSize `yaml:"min_size" json:"min_size"` // ignore files smaller than this
}

// DirDupConfig tunes folder comparison (needs Duplicates enabled).
type DirDupConfig struct {
	Enabled    bool    `yaml:"enabled" json:"enabled"`
	MinOverlap float64 `yaml:"min_overlap" json:"min_overlap"` // 0..1, share of the smaller folder
	MinFiles   int     `yaml:"min_files" json:"min_files"`     // shared files needed for a pair
}

// Default returns the built-in configuration agreed in AGENTS/SPEC.md.
func Default() *Config {
	return &Config{
		SizeRules: []SizeRule{
			{Name: "huge", Extensions: []string{GenericExt}, Threshold: 100 * MB},
			{Name: "documents", Extensions: []string{"doc", "docx", "rtf", "odt"}, Threshold: 15 * MB},
			{Name: "presentations", Extensions: []string{"ppt", "pptx", "odp"}, Threshold: 15 * MB},
			{Name: "spreadsheets", Extensions: []string{"xls", "xlsx", "ods"}, Threshold: 15 * MB},
			{Name: "pdf", Extensions: []string{"pdf"}, Threshold: 30 * MB},
			{Name: "images", Extensions: []string{"jpg", "jpeg", "png", "gif", "bmp", "tif", "tiff", "heic", "webp"}, Threshold: 5 * MB},
		},
		Archives:      []string{"zip", "rar", "7z", "tar", "gz", "tgz", "bz2", "xz", "z", "cab", "arj", "lzh", "iso", "img"},
		Distributives: []string{"exe", "msi", "msix", "appx", "deb", "rpm", "dmg", "pkg"},
		Junk: []string{
			"*.bak", "*.tmp", "*.temp", "*.old", "*.orig", "*.swp",
			"~$*", "~*.tmp",
			"Thumbs.db", "desktop.ini", ".DS_Store", "._*", ".Spotlight-V100", ".Trashes",
			"*.crdownload", "*.part",
		},
		TopN:             20,
		Duplicates:       DupConfig{Enabled: true, MinSize: 1 * KB},
		FolderDuplicates: DirDupConfig{Enabled: true, MinOverlap: 0.5, MinFiles: 2},
		SimilarNames:     SimilarNamesConfig{Enabled: true},
		Videos:           []string{"mp4", "mkv", "avi", "mov", "wmv", "m4v", "mpg", "mpeg", "webm", "3gp", "insv", "lrv", "m4a", "mp3", "wav", "flac"},
		Quarantine: QuarantineConfig{
			Dir:              ".folder-inspect/quarantine",
			Stubs:            true,
			StubCategories:   []string{"oversize", "archive", "distributive", "duplicate", "dir-duplicate", "similar-name"},
			DeleteCategories: []string{"junk", "empty-file", "empty-dir"},
		},
	}
}

// Load reads a YAML file on top of the defaults. Unknown keys are errors so
// a typo in the file does not silently fall back to defaults.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := Default()
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil && !errors.Is(err, os.ErrNotExist) {
		if err.Error() == "EOF" { // empty file: defaults
			return cfg, cfg.normalize()
		}
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if err := cfg.normalize(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// Discover returns the config file to use: an explicit path, else
// .folder-inspect.yml in the first root, else in the user's home. The
// second value is false when no file exists (defaults apply).
func Discover(explicit string, roots []string) (string, bool) {
	if explicit != "" {
		return explicit, true
	}
	var candidates []string
	if len(roots) > 0 {
		candidates = append(candidates, filepath.Join(roots[0], FileName))
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, FileName))
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, true
		}
	}
	return "", false
}

// normalize lower-cases extensions, strips dots and validates the rules.
func (c *Config) normalize() error {
	if c.TopN <= 0 {
		c.TopN = Default().TopN
	}
	if fd := &c.FolderDuplicates; fd.Enabled {
		if fd.MinOverlap <= 0 || fd.MinOverlap > 1 {
			return fmt.Errorf("folder_duplicates.min_overlap must be within (0, 1], got %v", fd.MinOverlap)
		}
		if fd.MinFiles < 1 {
			return fmt.Errorf("folder_duplicates.min_files must be at least 1")
		}
	}
	for i := range c.SizeRules {
		r := &c.SizeRules[i]
		if r.Name == "" {
			return fmt.Errorf("size rule #%d has no name", i+1)
		}
		if len(r.Extensions) == 0 {
			return fmt.Errorf("size rule %q has no extensions (use \"*\" for all files)", r.Name)
		}
		if r.Threshold <= 0 {
			return fmt.Errorf("size rule %q has no threshold", r.Name)
		}
		r.Extensions = normalizeExts(r.Extensions)
	}
	c.Archives = normalizeExts(c.Archives)
	c.Distributives = normalizeExts(c.Distributives)
	c.Videos = normalizeExts(c.Videos)
	q := &c.Quarantine
	raw := strings.TrimSpace(strings.ReplaceAll(q.Dir, "\\", "/"))
	if raw == "" {
		raw = Default().Quarantine.Dir
	}
	if strings.HasPrefix(raw, "/") || strings.Contains(raw, ":") || strings.Contains("/"+raw+"/", "/../") {
		return fmt.Errorf("quarantine.dir must be a relative path inside the scanned folder, got %q", q.Dir)
	}
	q.Dir = strings.Trim(raw, "/")
	return nil
}

func normalizeExts(exts []string) []string {
	out := make([]string, 0, len(exts))
	for _, e := range exts {
		e = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(e), "."))
		if e != "" {
			out = append(out, e)
		}
	}
	return out
}
