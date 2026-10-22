package explain

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
)

func graphFixture() analysis.Evidence {
	return analysis.Evidence{Scope: "Working Tree", Files: []analysis.EvidenceFile{{Path: "login.tsx", Type: "modified", Lines: []analysis.EvidenceLine{{Side: "new", Number: 3, Text: "login()"}}}, {Path: "api.py", Type: "added", Lines: []analysis.EvidenceLine{{Side: "new", Number: 8, Text: "def login():"}}}, {Path: "old.java", Type: "deleted", Lines: []analysis.EvidenceLine{{Side: "old", Number: 4, Text: "class Old"}}}}}
}
func validGraph() Graph {
	a := Reference{Path: "login.tsx", Side: "new", Start: 3, End: 3}
	b := Reference{Path: "api.py", Side: "new", Start: 8, End: 8}
	return Graph{Nodes: []GraphNode{{ID: "form", Label: "Login form", Description: "Accepts input", Change: "Updated", Impact: "May start login", Uncertainty: "Not tested", Status: "modified", References: []Reference{a}}, {ID: "api", Label: "Login API", Description: "Handles login", Change: "Added", Impact: "May accept login", Uncertainty: "Not tested", Status: "added", References: []Reference{b}}}, Edges: []GraphEdge{{From: "form", To: "api", Label: "may call", Description: "The evidence suggests a connection", Kind: "inferred", References: []Reference{a, b}}}}
}
func TestGraphValidation(t *testing.T) {
	ev := graphFixture()
	if err := validateGraph(validGraph(), ev); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*Graph){"duplicate": func(g *Graph) { g.Nodes[1].ID = g.Nodes[0].ID }, "missing endpoint": func(g *Graph) { g.Edges[0].To = "missing" }, "fabricated file": func(g *Graph) { g.Nodes[0].References[0].Path = "fake.go" }, "wrong side": func(g *Graph) { g.Nodes[0].References[0].Side = "old" }, "bad line": func(g *Graph) { g.Nodes[0].References[0].Start = 300; g.Nodes[0].References[0].End = 300 }, "unsupported edge": func(g *Graph) { g.Edges[0].References = g.Edges[0].References[:1] }, "claimed deterministic": func(g *Graph) { g.Edges[0].Kind = "observed" }, "wrong status": func(g *Graph) { g.Nodes[0].Status = "added" }, "oversized label": func(g *Graph) { g.Nodes[0].Label = strings.Repeat("x", 161) }, "invalid id": func(g *Graph) { g.Nodes[0].ID = "<script>" }, "too many nodes": func(g *Graph) { g.Nodes = append(g.Nodes, make([]GraphNode, MaxGraphNodes)...) }, "too many edges": func(g *Graph) { g.Edges = append(g.Edges, make([]GraphEdge, MaxGraphEdges)...) }}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			g := validGraph()
			mutate(&g)
			if err := validateGraph(g, ev); err == nil {
				t.Fatal("invalid graph accepted")
			}
			raw, _ := json.Marshal(g)
			fallback := graphOrFallback(raw, ev)
			if fallback.Source != "unavailable" || fallback.Diagnostic != "invalid" || len(fallback.Edges) != 0 || len(fallback.Nodes) != 0 {
				t.Fatalf("fallback: %+v", fallback)
			}
		})
	}
}
func TestGraphSingleNodeAndMissingGraph(t *testing.T) {
	ev := graphFixture()
	g := validGraph()
	g.Nodes = g.Nodes[:1]
	g.Edges = nil
	raw, _ := json.Marshal(g)
	got := graphOrFallback(raw, ev)
	if got.Source != "unavailable" || got.Diagnostic != "unsupported" || len(got.Nodes) != 0 {
		t.Fatalf("single node: %+v", got)
	}
	for _, raw := range []json.RawMessage{nil, []byte("null"), []byte(`{"nodes":"bad"}`)} {
		got = graphOrFallback(raw, ev)
		if got.Source != "unavailable" || len(got.Edges) != 0 {
			t.Fatalf("fallback: %+v", got)
		}

	}
}
func TestGraphEscapedContent(t *testing.T) {
	g := validGraph()
	g.Nodes[0].Label = "<img src=x onerror=alert(1)>"
	raw, _ := json.Marshal(g)
	got := graphOrFallback(raw, graphFixture())
	if got.Source != "ai" || got.Nodes[0].Label != g.Nodes[0].Label {
		t.Fatal("unexpected graph")
	}
	encoded, _ := json.Marshal(got)
	if strings.Contains(string(encoded), "<img") {
		t.Fatalf("JSON must escape HTML: %s", encoded)
	}
}
func TestGraphOmittedAndInvalidPreserveText(t *testing.T) {
	for _, raw := range []string{"", `,"graph":{"nodes":"wrong"}`, `,"graph":{"nodes":[],"edges":[]}`} {
		content := strings.TrimSuffix(answer("main.go"), "}") + raw + "}"
		s := providerServer(t, 200, content)
		cfg, _ := NewConfig(s.URL, "test", "")
		got, err := NewProvider(cfg).Explain(context.Background(), fixture())
		s.Close()
		if err != nil || got.Summary != "A change" || got.Graph == nil || got.Graph.Source != "unavailable" {
			t.Fatalf("response: %+v %v", got, err)
		}
	}
}

func TestGraphSemanticAddedInModifiedFile(t *testing.T) {
	ev := graphFixture()
	ev.Files[0].Excerpt = "@@ -1,2 +1,3 @@\n existing\n+login()\n existing\n"
	ev.Files[0].Lines[0].Number = 2
	g := validGraph()
	g.Nodes[0].Status = "added"
	g.Nodes[0].References[0].Start = 2
	g.Nodes[0].References[0].End = 2
	g.Edges[0].References[0] = g.Nodes[0].References[0]
	if err := validateGraph(g, ev); err != nil {
		t.Fatalf("added function in modified file: %v", err)
	}
	g.Nodes[0].References[0].Start = 1
	g.Nodes[0].References[0].End = 1
	if err := validateGraph(g, ev); err == nil {
		t.Fatal("unsupported reference accepted")
	}
}
func TestGraphDiagnostics(t *testing.T) {
	ev := graphFixture()
	cases := []struct{ raw, want string }{{"", "missing"}, {"null", "missing"}, {`{"nodes":"bad"}`, "malformed"}, {`{"nodes":[],"edges":[]}`, "invalid"}}
	for _, tc := range cases {
		got := graphOrFallback([]byte(tc.raw), ev)
		if got.Diagnostic != tc.want {
			t.Errorf("%s: got %s", tc.raw, got.Diagnostic)
		}
	}
}
