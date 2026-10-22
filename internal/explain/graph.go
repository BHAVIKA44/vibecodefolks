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
	Source     string      `json:"source"`
	Diagnostic string      `json:"diagnostic,omitempty"`
	Notice     string      `json:"notice"`
	Nodes      []GraphNode `json:"nodes"`
	Edges      []GraphEdge `json:"edges"`
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
	category := "missing"
	if len(raw) > 0 && string(raw) != "null" {
		category = "malformed"
		var graph Graph
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&graph); err == nil {
			var extra any
			if dec.Decode(&extra) == io.EOF {
				category = "invalid"
				if validateGraph(graph, ev) == nil {
					if len(graph.Edges) == 0 {
						category = "unsupported"
					} else {
						graph.Source = "ai"
						graph.Notice = "Connections are AI inferences, not verified runtime behavior."
						return graph
					}
				}
			}
		}
	}
	return Graph{Source: "unavailable", Diagnostic: category, Notice: "No verified change flow available. Review the explanation and changed files; no runtime relationships have been established.", Nodes: []GraphNode{}, Edges: []GraphEdge{}}
}
func bounded(s string, limit int) bool { return strings.TrimSpace(s) != "" && len(s) <= limit }
func validateGraph(g Graph, ev analysis.Evidence) error {
	if g.Source != "" || g.Notice != "" || g.Diagnostic != "" || len(g.Nodes) == 0 || len(g.Nodes) > MaxGraphNodes || len(g.Edges) > MaxGraphEdges {
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
					if f.Path == ref.Path && ((n.Status == "modified" && (f.Type == "modified" || f.Type == "renamed")) || (n.Status == "added" && ref.Side == "new" && (f.Type == "added" || changedReference(f, ref, '+'))) || (n.Status == "removed" && ref.Side == "old" && (f.Type == "deleted" || changedReference(f, ref, '-')))) {
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

// changedReference checks only actual hunk additions/removals, not imported context.
func changedReference(f analysis.EvidenceFile, ref Reference, marker byte) bool {
	old, current := 0, 0
	active := false
	for _, line := range strings.Split(f.Excerpt, "\n") {
		if strings.HasPrefix(line, "@@ ") {
			var oldCount, newCount int
			if _, err := fmt.Sscanf(line, "@@ -%d,%d +%d,%d", &old, &oldCount, &current, &newCount); err != nil {
				// Git can omit the count for one-line hunks.
				parts := strings.Fields(line)
				if len(parts) < 3 {
					active = false
					continue
				}
				fmt.Sscanf(strings.Split(strings.TrimPrefix(parts[1], "-"), ",")[0], "%d", &old)
				fmt.Sscanf(strings.Split(strings.TrimPrefix(parts[2], "+"), ",")[0], "%d", &current)
			}
			active = true
			continue
		}
		if !active || len(line) == 0 {
			continue
		}
		switch line[0] {
		case ' ':
			old++
			current++
		case '-':
			if marker == '-' && old >= ref.Start && old <= ref.End {
				return true
			}
			old++
		case '+':
			if marker == '+' && current >= ref.Start && current <= ref.End {
				return true
			}
			current++
		case '\\':
		default:
			active = false
		}
	}
	return false
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
