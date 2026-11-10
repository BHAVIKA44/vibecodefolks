package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/BHAVIKA44/vibecodefolks/internal/explain"
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
		Token    string `json:"token"`
		Snapshot string `json:"snapshot"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &prep); err != nil || prep.Token == "" {
		t.Fatalf("prepare: %s", r.Body.String())
	}
	if calls.Load() != 0 {
		t.Fatal("preparation contacted provider")
	}
	r = request("POST", "/api/quiz/fresh", `{"snapshot":"`+prep.Snapshot+`"}`, "http://127.0.0.1:8888", "127.0.0.1:8888")
	if r.Code != 200 || !strings.Contains(r.Body.String(), `"fresh":true`) {
		t.Fatalf("snapshot fresh: %d %s", r.Code, r.Body.String())
	}
	if err := os.WriteFile(filepath.Join(d, "hello.go"), []byte("package changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r = request("POST", "/api/quiz/fresh", `{"snapshot":"`+prep.Snapshot+`"}`, "http://127.0.0.1:8888", "127.0.0.1:8888")
	if r.Code != 200 || !strings.Contains(r.Body.String(), `"fresh":false`) {
		t.Fatalf("snapshot stale: %d %s", r.Code, r.Body.String())
	}
	if calls.Load() != 0 {
		t.Fatal("snapshot check contacted provider")
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

func TestPreparedEvidenceCannotBeUsedConcurrently(t *testing.T) {
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
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"summary":"change","changes":[{"title":"package","explanation":"Package line","references":[{"path":"hello.go","side":"new","start":1,"end":1}]}],"impacts":[],"relationships":[],"uncertainties":[]}`}}}})
	}))
	defer mock.Close()
	t.Setenv("VIBECODEFOLKS_AI_ENDPOINT", mock.URL)
	t.Setenv("VIBECODEFOLKS_AI_MODEL", "test")
	t.Setenv("VIBECODEFOLKS_AI_API_KEY", "")
	h := Handler(d, fstest.MapFS{}, "example.test")
	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "http://example.test"+path, strings.NewReader(body))
		req.Header.Set("Origin", "http://example.test")
		req.Header.Set("Content-Type", "application/json")
		out := httptest.NewRecorder()
		h.ServeHTTP(out, req)
		return out
	}
	prep := post("/api/explain/prepare", "{}")
	if prep.Code != 200 {
		t.Fatal(prep.Body.String())
	}
	var token struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(prep.Body.Bytes(), &token); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan int, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- post("/api/explain", `{"token":"`+token.Token+`","consent":false}`).Code
		}()
	}
	wg.Wait()
	close(results)
	success, invalid := 0, 0
	for status := range results {
		if status == 200 {
			success++
		} else if status == 400 {
			invalid++
		}
	}
	if success != 1 || invalid != 1 || calls.Load() != 1 {
		t.Fatalf("success=%d invalid=%d calls=%d", success, invalid, calls.Load())
	}
}

func TestHarnessDiscoveryReportsAvailabilityWithoutClaimingSupport(t *testing.T) {
	dir := t.TempDir()
	name := "claude"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	handler := Handler(t.TempDir(), fstest.MapFS{}, "example.test")
	r := httptest.NewRequest("GET", "http://example.test/api/harnesses", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	var entries []struct {
		Name      string `json:"name"`
		Installed bool   `json:"installed"`
		Supported bool   `json:"supported"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &entries) != nil || len(entries) != 8 {
		t.Fatalf("discovery response: %d %s", w.Code, w.Body.String())
	}
	for _, entry := range entries {
		if entry.Name == "claude" && (!entry.Installed || entry.Supported) {
			t.Fatalf("bad Claude status: %+v", entry)
		}
		if entry.Name == "codex" && (entry.Installed || !entry.Supported) {
			t.Fatalf("bad Codex status: %+v", entry)
		}
	}
}

func TestDedicatedQuizUsesCapturedProviderAndFreshSnapshot(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	if output, err := exec.Command("git", "init", "-b", "main", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, output)
	}
	file := filepath.Join(dir, "hello.go")
	if err := os.WriteFile(file, []byte("package hello\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var content string
		if calls.Add(1) == 1 {
			content = `{"summary":"Added","changes":[{"title":"Added","explanation":"Added a package","references":[{"path":"hello.go","side":"new","start":1,"end":1}]}],"impacts":[],"relationships":[],"uncertainties":[]}`
		} else {
			content = `{"questions":[{"id":"q1","text":"Which outcome follows?","options":["one","two","three","four"],"correct_index":0,"explanation":"One follows.","option_explanations":["One follows.","Two does not.","Three does not.","Four does not."],"category":"change","references":[{"path":"hello.go","side":"new","start":1,"end":1}]}]}`
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}}})
	}))
	defer first.Close()
	var secondCalls atomic.Int32
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondCalls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer second.Close()
	t.Setenv("VIBECODEFOLKS_AI_ENDPOINT", first.URL)
	t.Setenv("VIBECODEFOLKS_AI_MODEL", "test")
	t.Setenv("VIBECODEFOLKS_AI_API_KEY", "")
	handler := Handler(dir, fstest.MapFS{}, "example.test")
	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "http://example.test"+path, strings.NewReader(body))
		req.Header.Set("Origin", "http://example.test")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	prepared := post("/api/explain/prepare", "{}")
	var prep struct {
		Token string `json:"token"`
	}
	if prepared.Code != 200 || json.Unmarshal(prepared.Body.Bytes(), &prep) != nil {
		t.Fatalf("prepare: %d %s", prepared.Code, prepared.Body.String())
	}
	explanation := post("/api/explain", `{"token":"`+prep.Token+`"}`)
	var review struct {
		Session  string `json:"quiz_session"`
		Snapshot string `json:"snapshot"`
	}
	if explanation.Code != 200 || json.Unmarshal(explanation.Body.Bytes(), &review) != nil || review.Session == "" {
		t.Fatalf("explanation: %d %s", explanation.Code, explanation.Body.String())
	}
	t.Setenv("VIBECODEFOLKS_AI_ENDPOINT", second.URL)
	body := `{"session":"` + review.Session + `","snapshot":"` + review.Snapshot + `"}`
	quiz := post("/api/quiz/generate", body)
	if quiz.Code != 200 || calls.Load() != 2 || secondCalls.Load() != 0 {
		t.Fatalf("quiz: %d %s, calls=%d second=%d", quiz.Code, quiz.Body.String(), calls.Load(), secondCalls.Load())
	}
	var output struct {
		Diagnostic string `json:"quiz_diagnostic"`
	}
	if json.Unmarshal(quiz.Body.Bytes(), &output) != nil || output.Diagnostic != "available" {
		t.Fatalf("quiz output: %s", quiz.Body.String())
	}
	if err := os.WriteFile(file, []byte("package changed\n"), 0600); err != nil {
		t.Fatal(err)
	}
	stale := post("/api/quiz/generate", body)
	if stale.Code != http.StatusConflict || calls.Load() != 2 {
		t.Fatalf("stale: %d %s, calls=%d", stale.Code, stale.Body.String(), calls.Load())
	}
}

func TestQuizSessionExpiresWithoutFurtherRequests(t *testing.T) {
	s := &state{quizzes: make(map[string]quizSession)}
	s.mu.Lock()
	s.addQuizSessionLocked("sample", prepared{config: explain.Config{APIKey: "fixture-secret"}}, time.Now(), 20*time.Millisecond)
	s.mu.Unlock()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		_, exists := s.quizzes["sample"]
		s.mu.Unlock()
		if !exists {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("expired quiz session was retained")
}

func TestProviderModelDiscoveryAndSessionSwitching(t *testing.T) {
	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/models" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]string{
					{"id": "whisper-large-v3"},
					{"id": "qwen/qwen3.8-27b"},
					{"id": "openai/gpt-oss-20b"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer mock.Close()

	h := Handler(t.TempDir(), fstest.MapFS{}, "example.test")
	post := func(path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "http://example.test"+path, strings.NewReader(body))
		req.Header.Set("Origin", "http://example.test")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	disc := post("/api/provider/models", `{"endpoint":"`+mock.URL+`/v1/chat/completions","api_key":"session-secret"}`)
	if disc.Code != http.StatusOK {
		t.Fatalf("discover: %d %s", disc.Code, disc.Body.String())
	}
	var discRes struct {
		Verified bool     `json:"verified"`
		KeySaved bool     `json:"key_saved"`
		Models   []string `json:"models"`
	}
	if err := json.Unmarshal(disc.Body.Bytes(), &discRes); err != nil || !discRes.Verified || !discRes.KeySaved || len(discRes.Models) != 2 {
		t.Fatalf("unexpected discovery payload: %+v %v", discRes, err)
	}

	// Non-chat audio model is rejected with a clear error.
	badModel := post("/api/provider", `{"endpoint":"`+mock.URL+`/v1/chat/completions","model":"whisper-large-v3","api_key":""}`)
	if badModel.Code != http.StatusBadRequest || !strings.Contains(badModel.Body.String(), "audio model") {
		t.Fatalf("expected non-chat model rejection: %d %s", badModel.Code, badModel.Body.String())
	}

	// Valid chat model reuses the saved session key without requiring it to be re-sent.
	goodModel := post("/api/provider", `{"endpoint":"`+mock.URL+`/v1/chat/completions","model":"qwen/qwen3.8-27b","api_key":""}`)
	if goodModel.Code != http.StatusOK || !strings.Contains(goodModel.Body.String(), `"verified":true`) {
		t.Fatalf("expected verified provider save: %d %s", goodModel.Code, goodModel.Body.String())
	}
}
