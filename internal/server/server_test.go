package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
)

func TestAPI(t *testing.T) {
	d := t.TempDir()
	c := exec.Command("git", "init", "-b", "main", d)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	h := Handler(d, fstest.MapFS{"index.html": {Data: []byte("ok")}}, "example.com")
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("GET", "/api/changes", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("success: %d %s", r.Code, r.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil || body["scope"] != "Working Tree" {
		t.Fatalf("bad JSON: %s %v", r.Body.String(), err)
	}
	r = httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/changes", nil)
	req.Header.Set("Origin", "http://example.com")
	h.ServeHTTP(r, req)
	if r.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST: %d", r.Code)
	}
	h = Handler(t.TempDir(), fstest.MapFS{}, "example.com")
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("GET", "/api/changes", nil))
	if r.Code != http.StatusInternalServerError {
		t.Fatalf("error: %d", r.Code)
	}
}

func TestConsentAndOrigin(t *testing.T) {
	d := t.TempDir()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	c := exec.Command("git", "init", "-b", "main", d)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	if err := os.WriteFile(filepath.Join(d, "hello.go"), []byte("package hello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"summary":"Added a package","changes":[{"title":"Package","explanation":"Introduces a package.","references":[{"path":"hello.go","side":"new","start":1,"end":1}]}],"impacts":[],"relationships":[],"uncertainties":["Not run"]}`}}}})
	}))
	defer mock.Close()
	t.Setenv("VIBECODEFOLKS_AI_ENDPOINT", mock.URL)
	t.Setenv("VIBECODEFOLKS_AI_MODEL", "test")
	t.Setenv("VIBECODEFOLKS_AI_API_KEY", "")
	h := Handler(d, fstest.MapFS{}, "127.0.0.1:8888")
	request := func(method, path, body, origin, host string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://"+host+path, strings.NewReader(body))
		req.Header.Set("Origin", origin)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	if calls.Load() != 0 {
		t.Fatal("provider called on startup")
	}
	r := request("POST", "/api/explain/prepare", "{}", "http://evil.test", "127.0.0.1:8888")
	if r.Code != 403 {
		t.Fatalf("origin: %d", r.Code)
	}
	r = request("GET", "/api/changes", "", "", "evil.test")
	if r.Code != 403 {
		t.Fatalf("host: %d", r.Code)
	}
	if calls.Load() != 0 {
		t.Fatal("cross-origin caused request")
	}
	r = request("POST", "/api/explain/prepare", "{}", "http://127.0.0.1:8888", "127.0.0.1:8888")
	if r.Code != 200 {
		t.Fatalf("prepare %d %s", r.Code, r.Body.String())
	}
	var prep struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &prep); err != nil || prep.Token == "" {
		t.Fatalf("prepare: %s", r.Body.String())
	}
	if calls.Load() != 0 {
		t.Fatal("preparation contacted provider")
	}
	r = request("POST", "/api/explain", `{"token":"`+prep.Token+`","consent":false}`, "http://127.0.0.1:8888", "127.0.0.1:8888")
	if r.Code != 200 || calls.Load() != 1 {
		t.Fatalf("local explanation: %d %s", r.Code, r.Body.String())
	}
	r = request("POST", "/api/explain", `{"token":"`+prep.Token+`","consent":false}`, "http://127.0.0.1:8888", "127.0.0.1:8888")
	if r.Code != 400 || calls.Load() != 1 {
		t.Fatal("token reused")
	}
	t.Setenv("VIBECODEFOLKS_AI_ENDPOINT", "https://example.com/chat")
	t.Setenv("VIBECODEFOLKS_AI_API_KEY", "test-key")
	r = request("POST", "/api/explain/prepare", "{}", "http://127.0.0.1:8888", "127.0.0.1:8888")
	if r.Code != 200 {
		t.Fatalf("remote prepare: %d %s", r.Code, r.Body.String())
	}
	json.Unmarshal(r.Body.Bytes(), &prep)
	r = request("POST", "/api/explain", `{"token":"`+prep.Token+`","consent":false}`, "http://127.0.0.1:8888", "127.0.0.1:8888")
	if r.Code != 403 || calls.Load() != 1 {
		t.Fatalf("remote without consent: %d", r.Code)
	}
}
