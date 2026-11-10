package explain

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fixture() analysis.Evidence {
	return analysis.Evidence{Scope: "Working Tree", Files: []analysis.EvidenceFile{{Path: "main.go", Lines: []analysis.EvidenceLine{{Side: "new", Number: 2, Text: "added"}}}}, Limitations: []string{"not runtime tested"}}
}
func answer(path string) string {
	return fmt.Sprintf(`{"summary":"A change","changes":[{"title":"Added","explanation":"Adds a line.","references":[{"path":%q,"side":"new","start":2,"end":2}]}],"impacts":[],"relationships":[],"uncertainties":["Not tested"]}`, path)
}
func providerServer(t *testing.T, code int, content string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Content-Type") != "application/json" {
			t.Error("invalid request")
		}
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}}})
	}))
}
func TestProviderSuccessAndValidation(t *testing.T) {
	s := providerServer(t, 200, answer("main.go"))
	defer s.Close()
	cfg, err := NewConfig(s.URL, "test", "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := NewProvider(cfg).Explain(context.Background(), fixture())
	if err != nil || got.Scope != "Working Tree" || len(got.Changes) != 1 {
		t.Fatalf("response: %+v %v", got, err)
	}
	for _, v := range []string{answer("fake.go"), strings.Replace(answer("main.go"), `"start":2`, `"start":9`, 1), "not json"} {
		s2 := providerServer(t, 200, v)
		cfg, _ = NewConfig(s2.URL, "test", "")
		_, err = NewProvider(cfg).Explain(context.Background(), fixture())
		s2.Close()
		if err == nil {
			t.Fatalf("accepted invalid response %s", v)
		}
	}
	s3 := providerServer(t, 500, "secret server error")
	defer s3.Close()
	cfg, _ = NewConfig(s3.URL, "test", "")
	_, err = NewProvider(cfg).Explain(context.Background(), fixture())
	if err == nil || strings.Contains(err.Error(), "secret") {
		t.Fatalf("provider error: %v", err)
	}
	for code, want := range map[int]string{
		http.StatusBadRequest:          "Check the endpoint, model name",
		http.StatusServiceUnavailable:  "temporarily unavailable",
		http.StatusTooManyRequests:     "rate limit or quota",
		http.StatusUnauthorized:        "rejected the API key",
		http.StatusForbidden:           "denied access",
		http.StatusNotFound:            "endpoint or model",
		http.StatusInternalServerError: "failed internally",
	} {
		s4 := providerServer(t, code, "provider details must not be exposed")
		cfg, _ = NewConfig(s4.URL, "test", "")
		_, err = NewProvider(cfg).Explain(context.Background(), fixture())
		s4.Close()
		if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), "provider details") {
			t.Fatalf("status %d: %v", code, err)
		}
	}
}

func TestProviderAcceptsFencedJSONAndIgnoresProviderMetadata(t *testing.T) {
	content := strings.TrimSuffix(answer("main.go"), "}") + `,"provider_trace":"ignored"}`
	s := providerServer(t, 200, "```json\n"+content+"\n```")
	defer s.Close()
	cfg, _ := NewConfig(s.URL, "test", "")
	got, err := NewProvider(cfg).Explain(context.Background(), fixture())
	if err != nil || got.Summary != "A change" || len(got.Changes) != 1 {
		t.Fatalf("fenced response: %+v %v", got, err)
	}
	for _, reserved := range []string{"scope", "limitations"} {
		content := strings.TrimSuffix(answer("main.go"), "}") + fmt.Sprintf(`,%q:"unexpected"}`, reserved)
		s2 := providerServer(t, 200, content)
		cfg, _ = NewConfig(s2.URL, "test", "")
		_, err = NewProvider(cfg).Explain(context.Background(), fixture())
		s2.Close()
		if err == nil {
			t.Fatalf("accepted reserved field %q", reserved)
		}
	}
}

func TestGroqUsesStrictStructuredOutput(t *testing.T) {
	p := &ChatProvider{Config: Config{Endpoint: "https://api.groq.com/openai/v1/chat/completions"}}
	format := p.responseFormat()
	if format["type"] != "json_schema" {
		t.Fatalf("Groq format = %#v", format)
	}
	schema, ok := format["json_schema"].(map[string]any)
	if !ok || schema["strict"] != true || schema["name"] != "vibecodefolks_explanation" {
		t.Fatalf("invalid Groq schema wrapper: %#v", format)
	}
	encoded, err := json.Marshal(format)
	if err != nil || !json.Valid(encoded) || !strings.Contains(string(encoded), `"additionalProperties":false`) {
		t.Fatalf("invalid Groq schema: %s %v", encoded, err)
	}
	for _, field := range []string{`"graph":`, `"changes":`, `"references":`} {
		if !strings.Contains(string(encoded), field) {
			t.Errorf("strict Groq response schema omits %s", field)
		}
	}
	plain := (&ChatProvider{Config: Config{Endpoint: "https://example.com/v1/chat/completions"}}).responseFormat()
	if plain["type"] != "json_object" {
		t.Fatalf("generic format = %#v", plain)
	}
}

func TestStructuredJSONObjectExtractsWrappedObject(t *testing.T) {
	wrapped := "Here is the result:\n" + answer("main.go") + "\nDone."
	got, err := structuredJSONObject(wrapped)
	if err != nil || !json.Valid(got) {
		t.Fatalf("wrapped JSON: %q %v", got, err)
	}
}
func TestResponsesEndpoint(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if r.URL.Path != "/v1/responses" || json.NewDecoder(r.Body).Decode(&payload) != nil {
			t.Fatal("invalid responses request")
		}
		if _, ok := payload["messages"]; ok {
			t.Fatal("sent chat payload to responses endpoint")
		}
		if payload["store"] != false {
			t.Fatal("responses must not store evidence")
		}
		json.NewEncoder(w).Encode(map[string]any{"output": []any{map[string]any{"content": []any{map[string]string{"type": "output_text", "text": answer("main.go")}}}}})
	}))
	defer s.Close()
	cfg, _ := NewConfig(s.URL+"/v1/responses", "test", "")
	got, err := NewProvider(cfg).Explain(context.Background(), fixture())
	if err != nil || got.Summary != "A change" {
		t.Fatalf("responses result: %+v %v", got, err)
	}
}
func TestConfig(t *testing.T) {
	for _, v := range []struct{ url, model, key string }{{"", "m", ""}, {"http://example.com", "m", "key"}, {"https://example.com", "m", ""}, {"https://u:p@example.com", "m", "key"}, {"https://example.com?q=secret", "m", "key"}} {
		if _, err := NewConfig(v.url, v.model, v.key); err == nil {
			t.Fatalf("accepted %+v", v)
		}
	}
	if _, err := NewConfig("http://localhost:1234/v1/chat/completions", "model", ""); err != nil {
		t.Fatal(err)
	}
}
func TestProviderCancellationAndTimeout(t *testing.T) {
	var hits atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); time.Sleep(100 * time.Millisecond) }))
	defer s.Close()
	cfg, _ := NewConfig(s.URL, "m", "")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := NewProvider(cfg).Explain(ctx, fixture())
	if err == nil {
		t.Fatal("expected timeout")
	}
	if hits.Load() != 1 {
		t.Fatal("mock not reached")
	}
}

func TestRelationshipNeedsBothReferences(t *testing.T) {
	var e Explanation
	if err := json.Unmarshal([]byte(answer("main.go")), &e); err != nil {
		t.Fatal(err)
	}
	e.Relationships = []Finding{{Title: "Connection", Explanation: "Related", References: []Reference{{Path: "main.go", Side: "new", Start: 2, End: 2}}}}
	if err := Validate(e, fixture()); err == nil {
		t.Fatal("accepted relationship with only one reference")
	}
}

func TestCodexHarnessUsesLocalCLIAndValidatesResult(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "codex")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	response := strings.TrimSuffix(answer("main.go"), "}") + `,"quiz":{"questions":[{"id":"branch","text":"Which branch can reach the added line?","options":["First","Second","Third","Fourth"],"correct_index":1,"explanation":"The second branch reaches the line.","references":[{"path":"main.go","side":"new","start":2,"end":2}],"category":"change"}]}}`
	source := fmt.Sprintf(`package main
import ("os"; "bytes")
func main() {
	if os.Getenv("VIBE_TEST_FAIL") == "auth" { os.Stderr.WriteString("not logged in secret-should-not-escape"); os.Exit(1) }
	if os.Getenv("VIBE_TEST_FAIL") == "process" { os.Stderr.WriteString("secret-should-not-escape"); os.Exit(1) }
	var out, schema string
	for i := 1; i+1 < len(os.Args); i++ {
		switch os.Args[i] {
		case "-o": out = os.Args[i+1]
		case "--output-schema": schema = os.Args[i+1]
		}
	}
	body, err := os.ReadFile(schema)
	if err != nil || !bytes.Contains(body, []byte("summary")) { os.Exit(1) }
 found := false
 for i := 1; i+1 < len(os.Args); i++ { if os.Args[i] == "--model" && os.Args[i+1] == "fictional-model" { found = true } }
 if !found { os.Exit(1) }
	if os.WriteFile(out, []byte(%q), 0600) != nil { os.Exit(1) }
}`, response)
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(sourcePath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(goTool, "build", "-o", binary, sourcePath).CombinedOutput(); err != nil {
		t.Fatalf("build CLI fixture: %v: %s", err, output)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	cfg, err := NewHarnessConfig("codex")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Model = "fictional-model"
	got, err := ProviderFor(cfg).Explain(context.Background(), fixture())
	if err != nil || got.Summary != "A change" || got.Graph == nil || got.Quiz == nil || *got.Quiz.Questions[0].CorrectIndex != 1 {
		t.Fatalf("harness response: %+v %v", got, err)
	}
	for _, tc := range []struct{ mode, expected string }{{"auth", "sign-in"}, {"process", "exited"}} {
		t.Run(tc.mode, func(t *testing.T) {
			t.Setenv("VIBE_TEST_FAIL", tc.mode)
			_, err := ProviderFor(cfg).Explain(context.Background(), fixture())
			if err == nil || !strings.Contains(err.Error(), tc.expected) || strings.Contains(err.Error(), "secret-should-not-escape") {
				t.Fatalf("unsafe or missing diagnostic: %v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ProviderFor(cfg).Explain(ctx, fixture()); err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("canceled request: %v", err)
	}
	if _, err := NewHarnessConfig("unknown"); err == nil {
		t.Fatal("accepted unknown harness")
	}
}
