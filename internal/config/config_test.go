package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseByteSize(t *testing.T) {
	cases := map[string]ByteSize{
		"100MB": 100 * MB, "15 mb": 15 * MB, "1.5G": ByteSize(1.5 * float64(GB)),
		"512": 512, "2K": 2 * KB, "1kb": KB, "0": 0,
	}
	for in, want := range cases {
		got, err := ParseByteSize(in)
		if err != nil || got != want {
			t.Errorf("ParseByteSize(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "MB", "-1", "ten MB", "1PB"} {
		if _, err := ParseByteSize(bad); err == nil {
			t.Errorf("ParseByteSize(%q) should fail", bad)
		}
	}
}

func TestByteSizeString(t *testing.T) {
	if s := (15 * MB).String(); s != "15.0 MB" {
		t.Errorf("got %q", s)
	}
	if s := ByteSize(512).String(); s != "512 B" {
		t.Errorf("got %q", s)
	}
}

func TestDefaultIsValid(t *testing.T) {
	cfg := Default()
	if err := cfg.normalize(); err != nil {
		t.Fatal(err)
	}
	generic := 0
	for _, r := range cfg.SizeRules {
		if r.IsGeneric() {
			generic++
		}
	}
	if generic != 1 {
		t.Errorf("want exactly one generic rule, got %d", generic)
	}
}

func TestLoadOverridesLists(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, FileName)
	yml := `
exclude: ["Старое/", "*.log"]
size_rules:
  - name: video
    extensions: [".MP4", "mkv"]
    threshold: 200MB
  - name: any
    extensions: ["*"]
    threshold: 1GB
top_n: 5
`
	if err := os.WriteFile(p, []byte(yml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.SizeRules) != 2 || cfg.SizeRules[0].Extensions[0] != "mp4" || cfg.SizeRules[0].Threshold != 200*MB {
		t.Errorf("size_rules not replaced/normalized: %+v", cfg.SizeRules)
	}
	if cfg.TopN != 5 || len(cfg.Exclude) != 2 {
		t.Errorf("scalars not loaded: %+v", cfg)
	}
	if len(cfg.Archives) != len(Default().Archives) {
		t.Errorf("archives should keep defaults when absent")
	}
	if !cfg.Duplicates.Enabled || cfg.Duplicates.MinSize != KB {
		t.Errorf("duplicates defaults must survive when the key is absent: %+v", cfg.Duplicates)
	}
}

func TestLoadFolderDuplicatesSection(t *testing.T) {
	p := filepath.Join(t.TempDir(), FileName)
	os.WriteFile(p, []byte("folder_duplicates:\n  min_overlap: 0.8\n  min_files: 5\n"), 0o644)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.FolderDuplicates.Enabled || cfg.FolderDuplicates.MinOverlap != 0.8 || cfg.FolderDuplicates.MinFiles != 5 {
		t.Errorf("folder_duplicates not applied: %+v", cfg.FolderDuplicates)
	}
	os.WriteFile(p, []byte("folder_duplicates:\n  min_overlap: 1.5\n"), 0o644)
	if _, err := Load(p); err == nil {
		t.Error("min_overlap > 1 must fail")
	}
}

func TestLoadDuplicatesSection(t *testing.T) {
	p := filepath.Join(t.TempDir(), FileName)
	os.WriteFile(p, []byte("duplicates:\n  enabled: false\n  min_size: 10MB\n"), 0o644)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Duplicates.Enabled || cfg.Duplicates.MinSize != 10*MB {
		t.Errorf("duplicates section not applied: %+v", cfg.Duplicates)
	}
}

func TestLoadRejectsUnknownKeyAndBadRule(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, FileName)
	os.WriteFile(p, []byte("sizerules: []\n"), 0o644)
	if _, err := Load(p); err == nil || !strings.Contains(err.Error(), "sizerules") {
		t.Errorf("unknown key should fail, got %v", err)
	}
	os.WriteFile(p, []byte("size_rules:\n  - name: x\n    extensions: [pdf]\n"), 0o644)
	if _, err := Load(p); err == nil {
		t.Error("rule without threshold should fail")
	}
}

func TestLoadEmptyFileGivesDefaults(t *testing.T) {
	p := filepath.Join(t.TempDir(), FileName)
	os.WriteFile(p, nil, 0o644)
	cfg, err := Load(p)
	if err != nil || len(cfg.SizeRules) != len(Default().SizeRules) {
		t.Fatalf("empty file: %v %+v", err, cfg)
	}
}

func TestDiscover(t *testing.T) {
	root := t.TempDir()
	if p, ok := Discover("explicit.yml", []string{root}); !ok || p != "explicit.yml" {
		t.Error("explicit path must win")
	}
	os.WriteFile(filepath.Join(root, FileName), []byte(""), 0o644)
	if p, ok := Discover("", []string{root}); !ok || p != filepath.Join(root, FileName) {
		t.Errorf("root config not found: %q %v", p, ok)
	}
}
