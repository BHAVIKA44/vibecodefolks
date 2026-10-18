package explain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
)

const MaxGraphNodes = 20
const MaxGraphEdges = 30

type Graph struct {
	Source string      `json:"source"`
	Notice string      `json:"notice"`
	Nodes  []GraphNode `json:"nodes"`
	Edges  []GraphEdge `json:"edges"`
}
type GraphNode struct {
	ID          string      `json:"id"`
	Label       string      `json:"label"`
	Description string      `json:"description"`
	Change      string      `json:"change"`
	Impact      string      `json:"impact"`
	Uncertainty string      `json:"uncertainty"`
	Status      string      `json:"status"`
	References  []Reference `json:"references"`
}
type GraphEdge struct {
	From        string      `json:"from"`
	To          string      `json:"to"`
	Label       string      `json:"label"`
	Description string      `json:"description"`
	Kind        string      `json:"kind"`
	References  []Reference `json:"references"`
}

var validID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Graphs are optional: malformed or unsupported graph data never discards valid text.
func graphOrFallback(raw json.RawMessage, ev analysis.Evidence) Graph {
	if len(raw) > 0 && string(raw) != "null" {
		var graph Graph
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if dec.Decode(&graph) == nil {
			var extra any
			if dec.Decode(&extra) == io.EOF && validateGraph(graph, ev) == nil {
				graph.Source = "ai"
				if graph.Edges == nil {
					graph.Edges = []GraphEdge{}
				}
				graph.Notice = "Connections are AI inferences, not verified runtime behavior."
				return graph
			}
		}
	}
	return fileGraph(ev)
}
func bounded(s string, limit int) bool { return strings.TrimSpace(s) != "" && len(s) <= limit }
func validateGraph(g Graph, ev analysis.Evidence) error {
	if g.Source != "" || g.Notice != "" || len(g.Nodes) == 0 || len(g.Nodes) > MaxGraphNodes || len(g.Edges) > MaxGraphEdges {
		return errors.New("invalid graph size or reserved fields")
	}
	nodes := map[string]GraphNode{}
	for _, n := range g.Nodes {
		if !validID.MatchString(n.ID) || !bounded(n.Label, 160) || !bounded(n.Description, 1200) || !bounded(n.Change, 1200) || !bounded(n.Impact, 1200) || !bounded(n.Uncertainty, 1200) || len(n.References) == 0 || len(n.References) > 10 {
			return errors.New("invalid graph node")
		}
		if _, exists := nodes[n.ID]; exists {
			return errors.New("duplicate graph node")
		}
		if n.Status != "added" && n.Status != "modified" && n.Status != "removed" && n.Status != "context" && n.Status != "unknown" {
			return errors.New("invalid node status")
		}
		for _, ref := range n.References {
			if !validRef(ref, ev) {
				return errors.New("invalid graph node reference")
			}
		}
		if n.Status == "context" && !hasUnchangedContext(n.References, ev) {
			return errors.New("context status lacks unchanged line evidence")
		}
		if n.Status != "unknown" && n.Status != "context" {
			matches := false
			for _, ref := range n.References {
				for _, f := range ev.Files {
					if f.Path == ref.Path && (f.Type == n.Status || f.Type == "deleted" && n.Status == "removed" || f.Type == "renamed" && n.Status == "modified") && (n.Status != "added" || ref.Side == "new") && (n.Status != "removed" || ref.Side == "old") {
						matches = true
					}
				}
			}
			if !matches {
				return errors.New("node status unsupported by Git evidence")
			}
		}
		nodes[n.ID] = n
	}
	for _, e := range g.Edges {
		from, okFrom := nodes[e.From]
		to, okTo := nodes[e.To]
		if !okFrom || !okTo || e.From == e.To || e.Kind != "inferred" || !bounded(e.Label, 100) || !bounded(e.Description, 1200) || len(e.References) < 2 || len(e.References) > 10 {
			return errors.New("invalid graph edge")
		}
		a, b := false, false
		distinct := map[Reference]bool{}
		for _, ref := range e.References {
			distinct[ref] = true
			if !validRef(ref, ev) {
				return errors.New("invalid graph edge reference")
			}
			for _, r := range from.References {
				if overlaps(ref, r) {
					a = true
				}
			}
			for _, r := range to.References {
				if overlaps(ref, r) {
					b = true
				}
			}
		}
		if !a || !b || len(distinct) < 2 {
			return errors.New("edge must cite evidence for both nodes")
		}
	}
	return nil
}
func overlaps(a, b Reference) bool {
	return a.Path == b.Path && a.Side == b.Side && a.Start <= b.End && b.Start <= a.End
}
func fileGraph(ev analysis.Evidence) Graph {
	g := Graph{Source: "files", Notice: "File-only view: no supported connections are available. The layout does not imply a flow.", Nodes: []GraphNode{}, Edges: []GraphEdge{}}
	for i, f := range ev.Files {
		if i >= MaxGraphNodes {
			g.Notice += " Some files are omitted due to the graph limit."
			break
		}
		if len(f.Lines) == 0 {
			continue
		}
		line := f.Lines[0]
		for _, l := range f.Lines {
			if f.Type != "deleted" && l.Side == "new" {
				line = l
				break
			}
		}
		status := f.Type
		if status == "deleted" {
			status = "removed"
		}
		if status != "added" && status != "modified" && status != "removed" {
			status = "unknown"
		}
		g.Nodes = append(g.Nodes, GraphNode{ID: fmt.Sprintf("file-%d", i), Label: f.Path, Description: "Changed file from Git evidence.", Change: "Git reports this file as " + f.Type + ".", Impact: "The effect on behavior has not been established.", Uncertainty: "Relationships and runtime behavior are not verified.", Status: status, References: []Reference{{Path: f.Path, Side: line.Side, Start: line.Number, End: line.Number}}})
	}
	return g
}

func hasUnchangedContext(refs []Reference, ev analysis.Evidence) bool {
	for _, ref := range refs {
		for _, f := range ev.Files {
			if f.Path != ref.Path {
				continue
			}
			for _, line := range f.Lines {
				if line.Side != ref.Side || line.Number < ref.Start || line.Number > ref.End {
					continue
				}
				for _, other := range f.Lines {
					if other.Side != line.Side && other.Text == line.Text {
						return true
					}
				}
			}
		}
	}
	return false
}
