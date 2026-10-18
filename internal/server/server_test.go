package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"testing/fstest"
)

func TestAPI(t *testing.T) {
	d := t.TempDir()
	c := exec.Command("git", "init", "-b", "main", d)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	h := Handler(d, fstest.MapFS{"index.html": {Data: []byte("ok")}})
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
	h.ServeHTTP(r, httptest.NewRequest("POST", "/api/changes", nil))
	if r.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST: %d", r.Code)
	}
	h = Handler(t.TempDir(), fstest.MapFS{})
	r = httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest("GET", "/api/changes", nil))
	if r.Code != http.StatusInternalServerError {
		t.Fatalf("error: %d", r.Code)
	}
}
