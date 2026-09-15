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
	names := detect.SimilarNames(res.Files, dups)
	findings = append(findings, dups.Findings()...)
	rep := report.Build(res, findings, dups, dirs, names, cfg, "test")
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

func TestRescanEndpoint(t *testing.T) {
	s, _ := newServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	before := s.ReportPath
	if resp, _ := http.Get(ts.URL + "/api/rescan"); resp.StatusCode != 405 {
		t.Errorf("GET rescan: %d", resp.StatusCode)
	}
	resp, err := http.Post(ts.URL+"/api/rescan", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := readAll(resp)
	if resp.StatusCode != 200 {
		t.Fatalf("rescan: %d %s", resp.StatusCode, body)
	}
	var rr rescanResponse
	json.Unmarshal([]byte(body), &rr)
	if rr.ReportPath == before || !strings.HasPrefix(rr.ReportPath, s.PlanDir) || rr.Files != len(fixture.Files) {
		t.Errorf("rescan response: %+v (before %s)", rr, before)
	}
	if s.ReportPath != rr.ReportPath || !exists(rr.ReportPath) {
		t.Error("server must switch to the new report on disk")
	}
	// the served report is the new one
	r2, _ := http.Get(ts.URL + "/api/report")
	var served reportResponse
	json.NewDecoder(r2.Body).Decode(&served)
	if served.ReportPath != rr.ReportPath {
		t.Error("/api/report must serve the rescanned report")
	}
}

func TestApplyEndpoint(t *testing.T) {
	s, root := newServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	post := func(body any) (int, applyResponse, string) {
		b, _ := json.Marshal(body)
		resp, err := http.Post(ts.URL+"/api/apply", "application/json", bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		txt, _ := readAll(resp)
		var ar applyResponse
		json.Unmarshal([]byte(txt), &ar)
		return resp.StatusCode, ar, txt
	}
	junk := filepath.Join(root, "Проект A", "Thumbs.db")
	archive := filepath.Join(root, "Проект B", "Старое", "Архив проекта 2024.zip")
	actions := []action.Action{
		{Op: action.OpQuarantine, Path: junk, Category: "junk", Size: 2048},
		{Op: action.OpQuarantine, Path: archive, Category: "archive", Size: 16384},
	}
	// dry run: reported, nothing moved, no plan saved
	code, dry, txt := post(applyRequest{Actions: actions, DryRun: true, Lang: "ru"})
	if code != 200 || !dry.DryRun || dry.Moved != 2 || dry.Plan != "" || len(dry.Entries) != 2 {
		t.Fatalf("dry run: %d %s", code, txt)
	}
	if !exists(junk) || !exists(archive) {
		t.Fatal("dry run must not move anything")
	}
	// real run
	code, real, txt := post(applyRequest{Actions: actions, DryRun: false, Lang: "ru"})
	if code != 200 || real.Moved != 2 || real.Stubs != 1 || len(real.Manifests) != 1 || real.Plan == "" {
		t.Fatalf("apply: %d %s", code, txt)
	}
	if exists(junk) || exists(archive) || !exists(archive+".removed.txt") || exists(junk+".removed.txt") {
		t.Error("files must be quarantined; only the archive gets a stub")
	}
	if !strings.HasPrefix(real.Manifests[0], filepath.Join(root, ".folder-inspect", "quarantine")) || !exists(real.Manifests[0]) {
		t.Errorf("manifest: %v", real.Manifests)
	}
	if !exists(real.Plan) {
		t.Error("plan must be saved for a real apply")
	}
	// outside path refused
	code, _, _ = post(applyRequest{Actions: []action.Action{{Op: action.OpQuarantine, Path: filepath.Join(t.TempDir(), "x")}}})
	if code != 403 {
		t.Errorf("outside path: %d", code)
	}
}

func TestQuarantineAndRestoreEndpoints(t *testing.T) {
	s, root := newServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	// nothing yet
	resp, _ := http.Get(ts.URL + "/api/quarantine")
	var list []batchView
	json.NewDecoder(resp.Body).Decode(&list)
	if len(list) != 0 {
		t.Fatalf("empty quarantine expected: %+v", list)
	}
	// apply one archive
	archive := filepath.Join(root, "Проект B", "Старое", "Архив проекта 2024.zip")
	b, _ := json.Marshal(applyRequest{Actions: []action.Action{{Op: action.OpQuarantine, Path: archive, Category: "archive", Size: 16384}}})
	resp, _ = http.Post(ts.URL+"/api/apply", "application/json", bytes.NewReader(b))
	if resp.StatusCode != 200 {
		t.Fatal("apply failed")
	}
	resp, _ = http.Get(ts.URL + "/api/quarantine")
	json.NewDecoder(resp.Body).Decode(&list)
	if len(list) != 1 || list[0].Pending != 1 || list[0].Status != "active" || list[0].Size != 16384 {
		t.Fatalf("quarantine list: %+v", list)
	}
	// restore via the API
	rb, _ := json.Marshal(map[string]string{"manifest": list[0].Path})
	resp, _ = http.Post(ts.URL+"/api/restore", "application/json", bytes.NewReader(rb))
	body, _ := readAll(resp)
	var rr restoreResponse
	json.Unmarshal([]byte(body), &rr)
	if resp.StatusCode != 200 || rr.Restored != 1 || len(rr.Problems) != 0 || !exists(archive) || exists(archive+".removed.txt") {
		t.Errorf("restore: %d %s", resp.StatusCode, body)
	}
	resp, _ = http.Get(ts.URL + "/api/quarantine")
	json.NewDecoder(resp.Body).Decode(&list)
	if list[0].Status != "restored" || list[0].Pending != 0 {
		t.Errorf("after restore: %+v", list)
	}
	// outside / wrong file refused
	for _, bad := range []string{filepath.Join(t.TempDir(), "manifest.json"), filepath.Join(root, "Проект A", "Thumbs.db")} {
		rb, _ := json.Marshal(map[string]string{"manifest": bad})
		resp, _ = http.Post(ts.URL+"/api/restore", "application/json", bytes.NewReader(rb))
		if resp.StatusCode != 403 {
			t.Errorf("%s: %d", bad, resp.StatusCode)
		}
	}
}

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

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
