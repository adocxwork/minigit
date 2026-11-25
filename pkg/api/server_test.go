package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"mgit/pkg/core"
)

func testAPI(t *testing.T) (string, http.Handler) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := core.InitRepo(); err != nil {
		t.Fatal(err)
	}
	root, err := core.GetRepoPath()
	if err != nil {
		t.Fatal(err)
	}
	return root, (&API{RepoRoot: root}).handler(nil)
}

func request(t *testing.T, handler http.Handler, method, path, body string, code int, setup func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, strings.NewReader(body))
	if method == http.MethodPost {
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://127.0.0.1:8080")
	}
	if setup != nil {
		setup(r)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != code {
		t.Fatalf("%s %s returned %d, expected %d: %s", method, path, w.Code, code, w.Body)
	}
	return w
}

func TestDashboardAPIStagesFromRepositoryRoot(t *testing.T) {
	root, handler := testAPI(t)
	dir := filepath.Join(root, "nested")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "root-file"), []byte("demo"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	request(t, handler, "POST", "/api/branch", `{"name":"feature"}`, 409, nil)
	request(t, handler, "POST", "/api/add", `{"paths":["root-file"]}`, 200, nil)
	request(t, handler, "POST", "/api/commit", `{"message":"first"}`, 200, nil)
	request(t, handler, "POST", "/api/branch", `{"name":"feature"}`, 200, nil)
	w := request(t, handler, "GET", "/api/overview", "", 200, nil)
	var overview struct {
		Name   string
		Status struct{ HeadCommit string }
		Log    []logEntry
	}
	if err := json.Unmarshal(w.Body.Bytes(), &overview); err != nil {
		t.Fatal(err)
	}
	if len(overview.Log) != 1 || overview.Log[0].Message != "first" ||
		overview.Log[0].ID != overview.Status.HeadCommit || overview.Name != filepath.Base(root) {
		t.Fatalf("unexpected overview: %+v", overview)
	}
	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected headers: %v", w.Header())
	}
}

func TestAPIRejectsUnexpectedRequestMetadata(t *testing.T) {
	_, handler := testAPI(t)
	for _, tc := range []struct {
		name, method, path, body string
		code                     int
		setup                    func(*http.Request)
	}{
		{"unexpected host", "GET", "/api/overview", "", 403, func(r *http.Request) { r.Host = "invalid.example" }},
		{"unexpected origin", "POST", "/api/add", "{}", 403, func(r *http.Request) { r.Header.Set("Origin", "http://localhost:3000") }},
		{"unexpected fetch site", "GET", "/api/overview", "", 403, func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", "cross-site") }},
		{"wrong media type", "POST", "/api/add", "{}", 415, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }},
		{"unknown field", "POST", "/api/add", `{"unexpected":true}`, 400, nil},
		{"trailing JSON", "POST", "/api/add", "{} {}", 400, nil},
		{"oversized JSON", "POST", "/api/add", `{"paths":["` + strings.Repeat(" ", 1<<20) + `"]}`, 413, nil},
		{"unsupported method", "DELETE", "/api/branch", "", 405, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request(t, handler, tc.method, tc.path, tc.body, tc.code, tc.setup)
		})
	}
}

func TestDashboardCanServeStaticFilesAndAPI(t *testing.T) {
	root, _ := testAPI(t)
	handler := (&API{RepoRoot: root}).handler(fstest.MapFS{
		"index.html": {Data: []byte("<title>MiniGit</title>")},
	})
	w := request(t, handler, "GET", "/", "", 200, nil)
	if !strings.Contains(w.Body.String(), "MiniGit") {
		t.Fatalf("unexpected dashboard response: %s", w.Body)
	}
	request(t, handler, "POST", "/", "", 405, nil)
	request(t, handler, "GET", "/api/overview", "", 200, nil)
	request(t, handler, "GET", "/api/unknown", "", 404, nil)
}
