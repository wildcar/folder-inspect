// Package action holds the clean-up side of the tool: the action plan a
// person assembles in the UI (this file) and, later, apply / restore with
// quarantine and pointer stubs. Nothing here touches the user's files yet.
package action

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Op is what to do with a path.
type Op string

const (
	// OpQuarantine moves a file or folder into quarantine (junk, archive, oversized…).
	OpQuarantine Op = "quarantine"
	// OpQuarantineDuplicate moves a duplicate file into quarantine and leaves a
	// pointer stub naming Original.
	OpQuarantineDuplicate Op = "quarantine-duplicate"
	// OpQuarantineDir moves a duplicate folder into quarantine and leaves a
	// pointer stub naming the Original folder.
	OpQuarantineDir Op = "quarantine-dir"
)

var ops = map[Op]bool{OpQuarantine: true, OpQuarantineDuplicate: true, OpQuarantineDir: true}

// Action is one planned step.
type Action struct {
	Op       Op     `json:"op"`
	Path     string `json:"path"`
	Original string `json:"original,omitempty"` // duplicates: the copy that stays
	Group    string `json:"group,omitempty"`    // duplicate group id
	Category string `json:"category,omitempty"` // finding category the item came from
	Size     int64  `json:"size"`
	IsDir    bool   `json:"is_dir,omitempty"`
}

// PlanSchema changes when the plan layout changes incompatibly.
const PlanSchema = 1

// Plan is a reviewed list of actions, saved as plan-<ts>.json next to the
// report and consumed by `apply`.
type Plan struct {
	Schema  int       `json:"schema"`
	Tool    string    `json:"tool"`
	Created time.Time `json:"created"`
	Report  string    `json:"report"` // report the plan was built from
	Roots   []string  `json:"roots"`  // every path must lie under one of these
	Actions []Action  `json:"actions"`
}

// New starts an empty plan for a report.
func New(reportPath string, roots []string) *Plan {
	return &Plan{Schema: PlanSchema, Tool: "folder-inspect", Created: time.Now(), Report: reportPath, Roots: roots, Actions: []Action{}}
}

// Validate checks every action: known op, path inside a root, duplicate ops
// naming an original that is a different path inside a root.
func (p *Plan) Validate() error {
	if len(p.Roots) == 0 {
		return errors.New("plan has no roots")
	}
	seen := map[string]bool{}
	for i, a := range p.Actions {
		if !ops[a.Op] {
			return fmt.Errorf("action %d: unknown op %q", i+1, a.Op)
		}
		if !p.underRoot(a.Path) {
			return fmt.Errorf("action %d: %s is outside the scanned folders", i+1, a.Path)
		}
		if seen[a.Path] {
			return fmt.Errorf("action %d: %s is listed twice", i+1, a.Path)
		}
		seen[a.Path] = true
		if a.Op == OpQuarantineDuplicate || a.Op == OpQuarantineDir {
			if a.Original == "" || !p.underRoot(a.Original) {
				return fmt.Errorf("action %d: %s needs an original inside the scanned folders", i+1, a.Path)
			}
			if sameOrUnder(a.Original, a.Path) || sameOrUnder(a.Path, a.Original) {
				return fmt.Errorf("action %d: original %s and copy %s overlap", i+1, a.Original, a.Path)
			}
		}
	}
	for _, a := range p.Actions {
		if a.Op == OpQuarantineDuplicate || a.Op == OpQuarantineDir {
			if seen[a.Original] {
				return fmt.Errorf("%s is both an original and planned for quarantine", a.Original)
			}
		}
	}
	return nil
}

func (p *Plan) underRoot(path string) bool {
	if path == "" || !filepath.IsAbs(path) {
		return false
	}
	clean := filepath.Clean(path)
	for _, r := range p.Roots {
		if sameOrUnder(clean, filepath.Clean(r)) && clean != filepath.Clean(r) {
			return true
		}
	}
	return false
}

func sameOrUnder(p, parent string) bool {
	p, parent = strings.ToLower(p), strings.ToLower(parent)
	return p == parent || strings.HasPrefix(p, parent+string(filepath.Separator))
}

// TotalSize sums the planned sizes.
func (p *Plan) TotalSize() int64 {
	var n int64
	for _, a := range p.Actions {
		n += a.Size
	}
	return n
}

// Save validates and writes the plan as plan-<ts>.json in dir, never
// overwriting an existing file. Returns the path written.
func (p *Plan) Save(dir string) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	base := "plan-" + p.Created.Format("2006-01-02_150405")
	path := filepath.Join(dir, base+".json")
	for i := 2; ; i++ {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			break
		}
		path = filepath.Join(dir, fmt.Sprintf("%s-%d.json", base, i))
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

// Load reads a plan written by Save.
func Load(path string) (*Plan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	if p.Tool != "folder-inspect" || p.Schema == 0 {
		return nil, errors.New(path + ": not a folder-inspect plan")
	}
	return &p, nil
}
