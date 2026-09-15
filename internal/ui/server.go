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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/wildcar/folder-inspect/internal/action"
	"github.com/wildcar/folder-inspect/internal/export"
	"github.com/wildcar/folder-inspect/internal/i18n"
	"github.com/wildcar/folder-inspect/internal/report"
)

//go:embed static
var static embed.FS

// Server holds one report.
type Server struct {
	Report     *report.Report
	ReportPath string // absolute path the report was loaded from
	PlanDir    string // where plans are saved (default: folder of the report)
	Version    string
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
	return mux
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
	if !s.insideRoots(req.Path) {
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

func reveal(path string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer.exe", "/select,"+path).Start()
	case "darwin":
		return exec.Command("open", "-R", path).Start()
	default:
		dir := path
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			dir = filepath.Dir(path)
		}
		return exec.Command("xdg-open", dir).Start()
	}
}

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
