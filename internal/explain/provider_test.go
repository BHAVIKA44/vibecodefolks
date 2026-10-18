package explain

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
	"net/http"
	"net/http/httptest"
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
