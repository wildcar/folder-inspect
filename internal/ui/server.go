// Package ui serves a saved report as a local web page: findings by
// category, duplicate groups with a pick for the original, exports, and an
// action plan that is saved as plan-<ts>.json for `apply`. It binds to
// 127.0.0.1 only and embeds its static files, so nothing is installed and
// no network access is needed.
package ui

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/wildcar/folder-inspect/internal/action"
	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/export"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/pipeline"
	"github.com/wildcar/folder-inspect/internal/report"
)

//go:embed static
var static embed.FS

// Server holds one report. Rescan and apply replace or act on it, so every
// handler takes the mutex.
type Server struct {
	Report     *report.Report
	ReportPath string // absolute path the report was loaded from
	PlanDir    string // where plans and new reports are saved (default: folder of the report)
	Version    string

	mu sync.Mutex
}

// Handler builds the HTTP routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	sub, err := fs.Sub(static, "static")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", noCache(http.FileServer(http.FS(sub))))
	mux.HandleFunc("/api/report", s.handleReport)
	mux.HandleFunc("/api/export", s.handleExport)
	mux.HandleFunc("/api/plan", s.handlePlan)
	mux.HandleFunc("/api/reveal", s.handleReveal)
	mux.HandleFunc("/api/rescan", s.handleRescan)
	mux.HandleFunc("/api/apply", s.handleApply)
	mux.HandleFunc("/api/quarantine", s.handleQuarantine)
	mux.HandleFunc("/api/restore", s.handleRestore)
	return mux
}

type batchView struct {
	Path     string    `json:"path"`
	Dir      string    `json:"dir"`
	Created  time.Time `json:"created"`
	Items    int       `json:"items"`
	Pending  int       `json:"pending"`
	Restored int       `json:"restored"`
	Size     int64     `json:"size"`
	Status   string    `json:"status"`
}

// handleQuarantine lists the quarantine batches of every root.
func (s *Server) handleQuarantine(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir := ""
	if s.Report.Config != nil {
		dir = s.Report.Config.Quarantine.Dir
	}
	out := []batchView{}
	for _, root := range s.Report.Roots {
		batches, err := action.ListBatches(root, dir)
		if err != nil {
			continue
		}
		for _, b := range batches {
			out = append(out, batchView{Path: b.Path, Dir: b.Manifest.Dir, Created: b.Manifest.Created,
				Items: b.Items, Pending: b.Pending, Restored: b.Restored, Size: b.Size, Status: b.Status})
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(out)
}

type restoreResponse struct {
	Restored int              `json:"restored"`
	Problems []action.Problem `json:"problems"`
}

// handleRestore brings one batch back. The manifest must lie inside a root.
func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Manifest string `json:"manifest"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.insideRoots(req.Manifest) || filepath.Base(req.Manifest) != action.ManifestName {
		http.Error(w, "manifest is outside the scanned folders", http.StatusForbidden)
		return
	}
	m, err := action.LoadManifest(req.Manifest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := action.Restore(m, action.RestoreOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if res.Problems == nil {
		res.Problems = []action.Problem{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(restoreResponse{Restored: res.Restored, Problems: res.Problems})
}

type rescanResponse struct {
	ReportPath string `json:"report_path"`
	Files      int    `json:"files"`
	Findings   int    `json:"findings"`
}

// handleRescan re-runs the scan with the report's own effective config
// (same roots, thresholds and exclusions), saves a new report next to the
// current one and switches the UI to it.
func (s *Server) handleRescan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cfg := s.Report.Config
	if cfg == nil {
		cfg = config.Default()
	}
	rep, err := pipeline.Run(s.Report.Roots, cfg, s.Version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	path := report.UniquePath(filepath.Join(s.PlanDir, report.DefaultName(rep.Started)))
	if err := rep.WriteJSON(path); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.Report, s.ReportPath = rep, path
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(rescanResponse{ReportPath: path, Files: rep.Stats.Files, Findings: len(rep.Findings)})
}

type applyRequest struct {
	Actions []action.Action `json:"actions"`
	DryRun  bool            `json:"dry_run"`
	Lang    string          `json:"lang"`
}

type applyResponse struct {
	DryRun    bool             `json:"dry_run"`
	Plan      string           `json:"plan,omitempty"`
	Moved     int              `json:"moved"`
	Stubs     int              `json:"stubs"`
	Manifests []string         `json:"manifests"`
	Entries   []action.Entry   `json:"entries"`
	Problems  []action.Problem `json:"problems"`
}

// handleApply saves the plan (unless dry-run) and carries it out with the
// quarantine settings of the report's config. The browser asked for
// confirmation before calling this; the server only checks validity.
func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req applyRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 50<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := action.New(s.ReportPath, s.Report.Roots)
	p.Actions = req.Actions
	if len(p.Actions) == 0 {
		http.Error(w, "plan is empty", http.StatusBadRequest)
		return
	}
	if err := p.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	resp := applyResponse{DryRun: req.DryRun, Manifests: []string{}, Entries: []action.Entry{}, Problems: []action.Problem{}}
	if !req.DryRun {
		path, err := p.Save(s.PlanDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resp.Plan = path
	}
	lang := req.Lang
	if lang != i18n.EN && lang != i18n.RU {
		lang = i18n.Lang()
	}
	opt := action.OptionsFromConfig(s.Report.Config, lang, resp.Plan, action.FindingsByPath(s.Report.Findings))
	opt.DryRun = req.DryRun
	res, err := action.Apply(p, opt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Moved, resp.Stubs, resp.Problems = res.Moved, res.Stubs, res.Problems
	if resp.Problems == nil {
		resp.Problems = []action.Problem{}
	}
	for _, m := range res.Manifests {
		resp.Entries = append(resp.Entries, m.Entries...)
		if !req.DryRun && len(m.Entries) > 0 {
			resp.Manifests = append(resp.Manifests, m.Path)
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(resp)
}

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		h.ServeHTTP(w, r)
	})
}

type reportResponse struct {
	Lang       string            `json:"lang"`
	Strings    map[string]string `json:"strings"`
	ReportPath string            `json:"report_path"`
	PlanDir    string            `json:"plan_dir"`
	Version    string            `json:"version"`
	OS         string            `json:"os"`
	Report     *report.Report    `json:"report"`
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lang := pickLang(r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(reportResponse{
		Lang: lang, Strings: i18n.All(lang), ReportPath: s.ReportPath, PlanDir: s.PlanDir,
		Version: s.Version, OS: runtime.GOOS, Report: s.Report,
	})
}

func pickLang(r *http.Request) string {
	if l := r.URL.Query().Get("lang"); l == i18n.EN || l == i18n.RU {
		return l
	}
	return i18n.Lang()
}

var contentTypes = map[string]string{
	"csv":  "text/csv; charset=utf-8",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"html": "text/html; charset=utf-8",
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if !export.Valid(format) {
		http.Error(w, "unknown format", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// Exports read the process-wide language; the tool is single-user.
	prev := i18n.Lang()
	i18n.Set(pickLang(r))
	defer i18n.Set(prev)

	name := report.SiblingPath(filepath.Base(s.ReportPath), format)
	w.Header().Set("Content-Type", contentTypes[format])
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	if err := export.Render(s.Report, format, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type planRequest struct {
	Actions []action.Action `json:"actions"`
}

type planResponse struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
	Size  int64  `json:"size"`
}

func (s *Server) handlePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req planRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 50<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	p := action.New(s.ReportPath, s.Report.Roots)
	p.Actions = req.Actions
	if len(p.Actions) == 0 {
		http.Error(w, "plan is empty", http.StatusBadRequest)
		return
	}
	if err := p.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	path, err := p.Save(s.PlanDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(planResponse{Path: path, Count: len(p.Actions), Size: p.TotalSize()})
}

// handleReveal opens the file manager at a path inside the scanned roots.
func (s *Server) handleReveal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	inside := s.insideRoots(req.Path)
	s.mu.Unlock()
	if !inside {
		http.Error(w, "path is outside the scanned folders", http.StatusForbidden)
		return
	}
	if err := reveal(req.Path); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) insideRoots(p string) bool {
	if p == "" || !filepath.IsAbs(p) {
		return false
	}
	lp := strings.ToLower(filepath.Clean(p))
	for _, root := range s.Report.Roots {
		lr := strings.ToLower(filepath.Clean(root))
		if lp == lr || strings.HasPrefix(lp, lr+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// reveal is implemented per OS: reveal_windows.go and reveal_other.go.

// OpenBrowser opens url in the default browser; failures are not fatal.
func OpenBrowser(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

// Serve listens on addr (e.g. "127.0.0.1:0"), prints the URL to out,
// optionally opens the browser, and blocks until ctx is cancelled.
func Serve(ctx context.Context, s *Server, addr string, openBrowser bool, out io.Writer) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	url := "http://" + ln.Addr().String()
	fmt.Fprintln(out, i18n.Tf("ui.listening", url))
	if openBrowser {
		if err := OpenBrowser(url); err != nil {
			fmt.Fprintln(out, i18n.Tf("ui.open_failed", err))
		}
	}
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
