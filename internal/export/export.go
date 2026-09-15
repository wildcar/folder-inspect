// Package export renders a report for people who do not run the tool:
// CSV and XLSX for spreadsheets, a self-contained HTML page for e-mail.
// Everything is derived from report.Report; no file system access beyond
// writing the output.
package export

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/wildcar/folder-inspect/internal/report"
)

// Formats lists the supported export formats.
var Formats = []string{"csv", "xlsx", "html"}

// Valid reports whether format is supported.
func Valid(format string) bool {
	for _, f := range Formats {
		if f == format {
			return true
		}
	}
	return false
}

// ParseList parses "csv,html" into validated formats, de-duplicated and
// in canonical order.
func ParseList(s string) ([]string, error) {
	set := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		if !Valid(part) {
			return nil, fmt.Errorf("unknown export format %q (use %s)", part, strings.Join(Formats, ", "))
		}
		set[part] = true
	}
	var out []string
	for _, f := range Formats {
		if set[f] {
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out, nil
}

// Render writes the report in the given format to w.
func Render(r *report.Report, format string, w io.Writer) error {
	switch format {
	case "csv":
		return CSV(r, w)
	case "xlsx":
		return XLSX(r, w)
	case "html":
		return HTML(r, w)
	}
	return fmt.Errorf("unknown export format %q", format)
}

// WriteFile renders to path, creating parent folders. Overwrite policy is
// the caller's job (report.CheckOverwrite).
func WriteFile(r *report.Report, format, path string) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := Render(r, format, f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
