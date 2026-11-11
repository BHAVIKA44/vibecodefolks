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

// Live Groq qwen/qwen3.8-27b returned one single-citation relationship among
// otherwise valid findings; the whole explanation used to be discarded.
func TestDecodeDropsSingleReferenceRelationshipButKeepsExplanation(t *testing.T) {
	ev := fixture()
	ev.Files[0].Lines = append(ev.Files[0].Lines, analysis.EvidenceLine{Side: "new", Number: 3, Text: "call"})
	content := strings.Replace(answer("main.go"), `"relationships":[]`, `"relationships":[`+
		`{"title":"One end","explanation":"Only one citation.","references":[{"path":"main.go","side":"new","start":2,"end":2}]},`+
		`{"title":"Two ends","explanation":"Connects lines.","references":[{"path":"main.go","side":"new","start":2,"end":2},{"path":"main.go","side":"new","start":3,"end":3}]}]`, 1)
	got, err := decodeExplanation(content, ev)
	if err != nil || len(got.Relationships) != 1 || got.Relationships[0].Title != "Two ends" || len(got.Changes) != 1 {
		t.Fatalf("relationships=%+v err=%v", got.Relationships, err)
	}
	// Citations outside the evidence are still rejected, never rewritten.
	bad := strings.Replace(answer("main.go"), `"start":2,"end":2`, `"start":7,"end":7`, 1)
	if _, err := decodeExplanation(bad, ev); err == nil {
		t.Fatal("accepted out-of-evidence change citation")
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
	for _, a := range os.Args {
		if a == "--help" {
			os.Stdout.WriteString("--sandbox --output-schema --output-last-message --ephemeral --skip-git-repo-check --model\n")
			return
		}
		if a == "--version" {
			os.Stdout.WriteString("codex-cli 0.166.0\n")
			return
		}
	}
	if len(os.Args) > 2 && os.Args[1] == "login" && os.Args[2] == "status" {
		os.Stdout.WriteString("Logged in using ChatGPT\n")
		return
	}
	if os.Getenv("VIBE_TEST_FAIL") == "auth" { os.Stderr.WriteString("not logged in secret-should-not-escape"); os.Exit(1) }
	if os.Getenv("VIBE_TEST_FAIL") == "quota" { os.Stderr.Write(bytes.Repeat([]byte("echoed-prompt\n"), 2000)); os.Stderr.WriteString("ERROR: You’ve hit your usage limit. secret-should-not-escape"); os.Exit(1) }
	if os.Getenv("VIBE_TEST_FAIL") == "model" { os.Stderr.WriteString("ERROR: The model is not supported secret-should-not-escape"); os.Exit(1) }
	if os.Getenv("VIBE_TEST_FAIL") == "process" { os.Stderr.WriteString("secret-should-not-escape"); os.Exit(1) }
	var out, schema string
	for i := 1; i+1 < len(os.Args); i++ {
		switch os.Args[i] {
		case "-o", "--output-last-message": out = os.Args[i+1]
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
	for _, tc := range []struct{ mode, expected string }{{"auth", "sign-in"}, {"quota", "usage or rate limit"}, {"model", "selected model is not supported"}, {"process", "exited"}} {
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

func TestModelAgnosticCapabilitiesAndSchemaOrdering(t *testing.T) {
	gptOss := CapabilitiesFor(Config{Endpoint: "https://api.groq.com/openai/v1/chat/completions", Model: "openai/gpt-oss-120b"})
	if gptOss.StructuredMode != "strict_json_schema" || gptOss.ReasoningFields["reasoning_effort"] != "low" || gptOss.MaxCompletionTokens != 2000 {
		t.Fatalf("unexpected gpt-oss capabilities: %+v", gptOss)
	}
	qwen := CapabilitiesFor(Config{Endpoint: "https://api.groq.com/openai/v1/chat/completions", Model: "qwen/qwen3.8-27b"})
	if _, hasEffort := qwen.ReasoningFields["reasoning_effort"]; qwen.StructuredMode != "strict_json_schema" || qwen.ReasoningFields["reasoning_format"] != "hidden" || hasEffort || qwen.MaxCompletionTokens != 0 {
		t.Fatalf("unexpected qwen capabilities: %+v", qwen)
	}
	gptOss20b := CapabilitiesFor(Config{Endpoint: "https://api.groq.com/openai/v1/chat/completions", Model: "openai/gpt-oss-20b"})
	if gptOss20b.StructuredMode != "strict_json_schema" || gptOss20b.ReasoningFields["reasoning_effort"] != "low" {
		t.Fatalf("unexpected gpt-oss-20b capabilities: %+v", gptOss20b)
	}
	llama := CapabilitiesFor(Config{Endpoint: "https://api.groq.com/openai/v1/chat/completions", Model: "llama-3.3-70b-versatile"})
	if llama.StructuredMode != "json_object" || len(llama.ReasoningFields) != 0 {
		t.Fatalf("Groq llama should use json_object without reasoning flags: %+v", llama)
	}
	o3 := CapabilitiesFor(Config{Endpoint: "https://api.openai.com/v1/chat/completions", Model: "o3-mini"})
	if o3.SupportsTemperature || o3.MaxCompletionTokens != 0 || len(o3.ReasoningFields) != 0 {
		t.Fatalf("OpenAI o3-mini must omit temperature and Groq fields: %+v", o3)
	}
	gemini := CapabilitiesFor(Config{Endpoint: "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions", Model: "gemini-2.5-flash"})
	if gemini.StructuredMode != "json_object" || gemini.MaxCompletionTokens != 0 || len(gemini.ReasoningFields) != 0 {
		t.Fatalf("Gemini must use json_object without Groq fields: %+v", gemini)
	}
	whisper := CapabilitiesFor(Config{Endpoint: "https://api.groq.com/openai/v1/chat/completions", Model: "whisper-large-v3"})
	if whisper.UnsupportedReason == "" {
		t.Fatal("expected whisper-large-v3 to be rejected as non-chat model")
	}

	quizSchemaBytes, err := json.Marshal(dedicatedQuizSchema())
	if err != nil {
		t.Fatal(err)
	}
	quizSchemaStr := string(quizSchemaBytes)
	expIdx := strings.Index(quizSchemaStr, `"explanation"`)
	optExpIdx := strings.Index(quizSchemaStr, `"option_explanations"`)
	correctIdx := strings.Index(quizSchemaStr, `"correct_index"`)
	if expIdx < 0 || optExpIdx < 0 || correctIdx < 0 || !(expIdx < optExpIdx && optExpIdx < correctIdx) {
		t.Fatalf("quiz schema properties not in logical order: exp=%d optExp=%d correct=%d", expIdx, optExpIdx, correctIdx)
	}
}

func TestDiscoverModelsFiltersNonChatModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "whisper-large-v3"},
				{"id": "meta-llama/llama-prompt-guard-2-86m"},
				{"id": "qwen/qwen3.8-27b"},
				{"id": "openai/gpt-oss-120b"},
			},
		})
	}))
	defer srv.Close()
	cfg, err := NewConfig(srv.URL+"/v1/chat/completions", "discovery", "")
	if err != nil {
		t.Fatal(err)
	}
	models, err := DiscoverModels(context.Background(), cfg)
	if err != nil {
		t.Fatalf("DiscoverModels failed: %v", err)
	}
	if len(models) != 2 || models[0] != "openai/gpt-oss-120b" || models[1] != "qwen/qwen3.8-27b" {
		t.Fatalf("unexpected discovered models: %v", models)
	}
}
