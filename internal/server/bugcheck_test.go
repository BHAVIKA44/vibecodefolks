package server

import (
	"encoding/json"
	"fmt"
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

func TestBugChecks(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	cmd := exec.Command("git", "init", "-b", "main", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}

	t.Setenv("VIBECODEFOLKS_AI_ENDPOINT", "")
	t.Setenv("VIBECODEFOLKS_AI_MODEL", "")

	h := Handler(dir, fstest.MapFS{"index.html": {Data: []byte("ok")}}, "example.test")

	request := func(method, path, body, contentType string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://example.test"+path, strings.NewReader(body))
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		req.Header.Set("Origin", "http://example.test")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	t.Run("GET_api_provider_returns_405", func(t *testing.T) {
		r := request("GET", "/api/provider", "", "")
		fmt.Printf("GET /api/provider: code=%d body=%q\n", r.Code, r.Body.String())
		if r.Code != http.StatusMethodNotAllowed {
			t.Errorf("GET /api/provider: expected 405, got %d", r.Code)
		}
	})

	t.Run("DELETE_api_provider_returns_405", func(t *testing.T) {
		r := request("DELETE", "/api/provider", "", "")
		fmt.Printf("DELETE /api/provider: code=%d body=%q\n", r.Code, r.Body.String())
		if r.Code != http.StatusMethodNotAllowed {
			t.Errorf("DELETE /api/provider: expected 405, got %d", r.Code)
		}
	})

	t.Run("non_hex_64char_snapshot_quiz_fresh", func(t *testing.T) {
		// 64 chars that are valid length but not valid hex (z is not hex)
		snapshot := strings.Repeat("z", 64)
		r := request("POST", "/api/quiz/fresh", `{"snapshot":"`+snapshot+`"}`, "application/json")
		fmt.Printf("non-hex 64char snapshot: code=%d body=%q\n", r.Code, r.Body.String())
		// Server only checks len(in.Snapshot) != 64, not if it's valid hex
		// So this should pass server validation but return fresh:false (since it won't match any digest)
		// BUT wait — in this test dir there's no evidence at all, so Collect() will error
		if r.Code == http.StatusOK {
			t.Logf("non-hex snapshot accepted by server (expected) — returns: %s", r.Body.String())
		} else {
			t.Logf("non-hex snapshot: %d %s (Collect() may fail on empty repo)", r.Code, r.Body.String())
		}
	})

	t.Run("POST_api_provider_without_content_type", func(t *testing.T) {
		r := request("POST", "/api/provider", "{}", "") // no Content-Type
		fmt.Printf("POST /api/provider no CT: code=%d body=%q\n", r.Code, r.Body.String())
		if r.Code != http.StatusUnsupportedMediaType {
			t.Logf("POST /api/provider without Content-Type returned %d (expected 415)", r.Code)
		}
	})

	t.Run("extra_fields_in_explain_body", func(t *testing.T) {
		// /api/explain uses DisallowUnknownFields — extra field should give 400
		r := request("POST", "/api/explain", `{"token":"abc","consent":false,"extra":"field"}`, "application/json")
		fmt.Printf("extra field in /api/explain: code=%d body=%q\n", r.Code, r.Body.String())
		if r.Code != http.StatusBadRequest {
			t.Errorf("extra field in /api/explain: expected 400 got %d", r.Code)
		}
	})

	t.Run("extra_fields_in_quiz_fresh_body", func(t *testing.T) {
		snapshot := strings.Repeat("a", 64)
		r := request("POST", "/api/quiz/fresh", `{"snapshot":"`+snapshot+`","extra":"field"}`, "application/json")
		fmt.Printf("extra field in /api/quiz/fresh: code=%d body=%q\n", r.Code, r.Body.String())
		if r.Code != http.StatusBadRequest {
			t.Errorf("extra field in /api/quiz/fresh: expected 400 got %d", r.Code)
		}
	})

	t.Run("empty_body_to_explain_prepare", func(t *testing.T) {
		r := request("POST", "/api/explain/prepare", "", "application/json")
		fmt.Printf("empty body to /api/explain/prepare: code=%d body=%q\n", r.Code, r.Body.String())
		// json.NewDecoder on empty body -> EOF error -> but the handler doesn't use decoder for body
		// prepare handler has: var in struct{}; json.NewDecoder(r.Body).Decode(&in) is NOT there!
		// Actually looking at the code: prepare just calls analysis.Collect — no body decode needed
	})

	t.Run("cors_sec_fetch_site_cross_site", func(t *testing.T) {
		req := httptest.NewRequest("GET", "http://example.test/api/changes", nil)
		req.Header.Set("Sec-Fetch-Site", "cross-site")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		fmt.Printf("Sec-Fetch-Site cross-site: code=%d body=%q\n", w.Code, w.Body.String())
		if w.Code != http.StatusForbidden {
			t.Errorf("cross-site Sec-Fetch-Site: expected 403 got %d", w.Code)
		}
	})

	t.Run("very_long_token_in_explain", func(t *testing.T) {
		// Token field has no length limit in the struct, only the 4096 byte body limit
		longToken := strings.Repeat("a", 4000)
		r := request("POST", "/api/explain", `{"token":"`+longToken+`","consent":false}`, "application/json")
		fmt.Printf("very long token: code=%d body=%q\n", r.Code, r.Body.String())
		// Should 400 (not found) not panic
		if r.Code == http.StatusInternalServerError {
			t.Errorf("long token caused 500!")
		}
	})

	t.Run("snapshot_exactly_63_chars", func(t *testing.T) {
		snapshot := strings.Repeat("a", 63)
		r := request("POST", "/api/quiz/fresh", `{"snapshot":"`+snapshot+`"}`, "application/json")
		fmt.Printf("63-char snapshot: code=%d body=%q\n", r.Code, r.Body.String())
		if r.Code != http.StatusBadRequest {
			t.Errorf("63-char snapshot: expected 400 got %d", r.Code)
		}
	})

	t.Run("token_retry_on_provider_failure", func(t *testing.T) {
		// Create a file to have changes
		if err := os.WriteFile(filepath.Join(dir, "feature.go"), []byte("package feature\nfunc Run() {}\n"), 0600); err != nil {
			t.Fatal(err)
		}
		// First mock that fails with 500
		var failCall atomic.Int32
		failingMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			failCall.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer failingMock.Close()

		t.Setenv("VIBECODEFOLKS_AI_ENDPOINT", failingMock.URL)
		t.Setenv("VIBECODEFOLKS_AI_MODEL", "test")
		t.Setenv("VIBECODEFOLKS_AI_API_KEY", "")

		hWithMock := Handler(dir, fstest.MapFS{"index.html": {Data: []byte("ok")}}, "example.test")
		postReq := func(path, body string) *httptest.ResponseRecorder {
			req := httptest.NewRequest("POST", "http://example.test"+path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "http://example.test")
			w := httptest.NewRecorder()
			hWithMock.ServeHTTP(w, req)
			return w
		}

		// Prepare a request
		prep := postReq("/api/explain/prepare", "{}")
		if prep.Code != 200 {
			t.Fatalf("prepare failed: %d %s", prep.Code, prep.Body.String())
		}
		var tokenResp struct {
			Token string `json:"token"`
		}
		json.Unmarshal(prep.Body.Bytes(), &tokenResp)

		// First call should fail with 502 (bad gateway)
		res1 := postReq("/api/explain", fmt.Sprintf(`{"token":%q,"consent":false}`, tokenResp.Token))
		if res1.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 Bad Gateway, got %d %s", res1.Code, res1.Body.String())
		}

		// Retry with SAME token should NOT return 400 "expired or already used"
		// It should attempt to call the provider again and fail with 502 again
		res2 := postReq("/api/explain", fmt.Sprintf(`{"token":%q,"consent":false}`, tokenResp.Token))
		if res2.Code == http.StatusBadRequest {
			t.Errorf("FAIL: token was deleted on provider failure! Cannot retry.")
		} else if res2.Code == http.StatusBadGateway {
			t.Logf("PASS: token preserved after provider failure, allowed retry")
		} else {
			t.Logf("unexpected status: %d", res2.Code)
		}
	})
}
