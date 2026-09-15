package ui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wildcar/folder-inspect/internal/action"
	"github.com/wildcar/folder-inspect/internal/config"
	"github.com/wildcar/folder-inspect/internal/detect"
	"github.com/wildcar/folder-inspect/internal/fixture"
	"github.com/wildcar/folder-inspect/internal/report"
	"github.com/wildcar/folder-inspect/internal/scan"
)

func newServer(t *testing.T) (*Server, string) {
	t.Helper()
	root := t.TempDir()
	if err := fixture.Generate(root, fixture.Options{Scale: 1024}); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	res, err := scan.Walk([]string{root}, scan.Options{})
	if err != nil {
		t.Fatal(err)
	}
	findings := detect.Run(res, cfg)
	dups := detect.Duplicates(res.Files, detect.DupOptions{MinSize: 1024})
	dirs := detect.DuplicateDirs(res, dups, detect.DirDupOptions{})
	findings = append(findings, dups.Findings()...)
	rep := report.Build(res, findings, dups, dirs, cfg, "test")
	dir := report.DefaultDir(res.Roots[0])
	os.MkdirAll(dir, 0o755)
	rp := filepath.Join(dir, "report-test.json")
	if err := rep.WriteJSON(rp); err != nil {
		t.Fatal(err)
	}
	return &Server{Report: rep, ReportPath: rp, PlanDir: dir, Version: "test"}, res.Roots[0]
}

func TestIndexAndReport(t *testing.T) {
	s, _ := newServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := readAll(resp)
	if resp.StatusCode != 200 || !strings.Contains(body, "<title>folder-inspect</title>") || !strings.Contains(body, "app.js") {
		t.Errorf("index: %d %q", resp.StatusCode, body[:min(len(body), 200)])
	}
	for _, f := range []string{"/app.js", "/style.css"} {
		r, _ := http.Get(ts.URL + f)
		if r.StatusCode != 200 {
			t.Errorf("%s: %d", f, r.StatusCode)
		}
	}

	resp, _ = http.Get(ts.URL + "/api/report?lang=en")
	var rr reportResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		t.Fatal(err)
	}
	if rr.Lang != "en" || rr.Strings["cat.junk"] != "Junk" || rr.Report.Stats.Files != len(fixture.Files) || rr.PlanDir == "" {
		t.Errorf("report response: lang=%s junk=%q files=%d", rr.Lang, rr.Strings["cat.junk"], rr.Report.Stats.Files)
	}
	if len(rr.Report.Duplicates) == 0 || len(rr.Report.DirDuplicates) == 0 {
		t.Error("duplicates missing from the served report")
	}
}

func TestExport(t *testing.T) {
	s, _ := newServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	for format, ct := range contentTypes {
		resp, err := http.Get(ts.URL + "/api/export?format=" + format + "&lang=ru")
		if err != nil {
			t.Fatal(err)
		}
		body, _ := readAll(resp)
		if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != ct || len(body) == 0 {
			t.Errorf("%s: %d %q", format, resp.StatusCode, resp.Header.Get("Content-Type"))
		}
		if cd := resp.Header.Get("Content-Disposition"); !strings.Contains(cd, "report-test."+format) {
			t.Errorf("%s: disposition %q", format, cd)
		}
	}
	if resp, _ := http.Get(ts.URL + "/api/export?format=pdf"); resp.StatusCode != 400 {
		t.Errorf("unknown format: %d", resp.StatusCode)
	}
}

func TestPlanEndpoint(t *testing.T) {
	s, root := newServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	post := func(body any) (*http.Response, string) {
		b, _ := json.Marshal(body)
		resp, err := http.Post(ts.URL+"/api/plan", "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		s, _ := readAll(resp)
		return resp, s
	}
	g := s.Report.Duplicates[0]
	orig, copy := g.Files[0].Path, g.Files[1].Path
	resp, body := post(map[string]any{"actions": []action.Action{
		{Op: action.OpQuarantine, Path: filepath.Join(root, "Проект A", "Thumbs.db"), Size: 2048, Category: "junk"},
		{Op: action.OpQuarantineDuplicate, Path: copy, Original: orig, Group: g.ID, Size: g.Size},
	}})
	if resp.StatusCode != 200 {
		t.Fatalf("valid plan: %d %s", resp.StatusCode, body)
	}
	var pr planResponse
	json.Unmarshal([]byte(body), &pr)
	if pr.Count != 2 || !strings.HasPrefix(pr.Path, s.PlanDir) {
		t.Errorf("plan response: %+v", pr)
	}
	loaded, err := action.Load(pr.Path)
	if err != nil || len(loaded.Actions) != 2 || loaded.Report != s.ReportPath {
		t.Errorf("saved plan: %v %+v", err, loaded)
	}

	resp, _ = post(map[string]any{"actions": []action.Action{{Op: action.OpQuarantine, Path: filepath.Join(t.TempDir(), "x")}}})
	if resp.StatusCode != 403 {
		t.Errorf("outside path must be refused: %d", resp.StatusCode)
	}
	resp, _ = post(map[string]any{"actions": []action.Action{}})
	if resp.StatusCode != 400 {
		t.Errorf("empty plan: %d", resp.StatusCode)
	}
	if resp, _ := http.Get(ts.URL + "/api/plan"); resp.StatusCode != 405 {
		t.Errorf("GET plan: %d", resp.StatusCode)
	}
}

func TestRevealRefusesOutside(t *testing.T) {
	s, _ := newServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	b, _ := json.Marshal(map[string]string{"path": filepath.Join(t.TempDir(), "secret")})
	resp, _ := http.Post(ts.URL+"/api/reveal", "application/json", bytes.NewReader(b))
	if resp.StatusCode != 403 {
		t.Errorf("outside path: %d", resp.StatusCode)
	}
	if !s.insideRoots(filepath.Join(s.Report.Roots[0], "x")) || s.insideRoots("relative/x") {
		t.Error("insideRoots")
	}
}

func readAll(r *http.Response) (string, error) {
	defer r.Body.Close()
	var b bytes.Buffer
	_, err := b.ReadFrom(r.Body)
	return b.String(), err
}
