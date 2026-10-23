package explain

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
)

// HarnessProvider follows the local-harness model: the selected CLI owns its
// authentication and model choice. VibeCodeFolks supplies only the reviewed
// evidence and validates the returned JSON before displaying it.
type HarnessProvider struct{ name string }

func NewHarnessProvider(name string) *HarnessProvider { return &HarnessProvider{name: name} }

func (p *HarnessProvider) Explain(ctx context.Context, ev analysis.Evidence) (Explanation, error) {
	if p.name != "codex" {
		return Explanation{}, errors.New("unsupported local harness")
	}
	binary, err := exec.LookPath("codex")
	if err != nil {
		return Explanation{}, errors.New("Codex CLI is not installed or available on PATH")
	}
	evidence, err := json.Marshal(struct {
		Scope       string                  `json:"scope"`
		Files       []analysis.EvidenceFile `json:"files"`
		Limitations []string                `json:"limitations"`
	}{ev.Scope, ev.Files, ev.Limitations})
	if err != nil {
		return Explanation{}, errors.New("could not encode evidence")
	}
	schema, err := json.Marshal(groqExplanationSchema)
	if err != nil {
		return Explanation{}, errors.New("could not create local harness schema")
	}
	dir, err := os.MkdirTemp("", "vibecodefolks-codex-")
	if err != nil {
		return Explanation{}, errors.New("could not prepare local harness")
	}
	defer os.RemoveAll(dir)
	schemaPath, outputPath := filepath.Join(dir, "schema.json"), filepath.Join(dir, "answer.json")
	if os.WriteFile(schemaPath, schema, 0600) != nil {
		return Explanation{}, errors.New("could not prepare local harness")
	}
	prompt := instruction + "\n\nReviewed evidence JSON follows. Return the requested JSON only.\n" + string(evidence)
	cmd := exec.CommandContext(ctx, binary, "exec", "--ephemeral", "--skip-git-repo-check", "-s", "read-only", "--output-schema", schemaPath, "-o", outputPath, "-")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(prompt)
	if output, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return Explanation{}, ctx.Err()
		}
		_ = output // Do not surface harness logs: they can contain model or environment details.
		return Explanation{}, errors.New("Codex could not generate an explanation; check that Codex is signed in and try again")
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		return Explanation{}, errors.New("Codex returned no explanation")
	}
	var result Explanation
	if json.Unmarshal(content, &result) != nil {
		return Explanation{}, errors.New("Codex returned malformed structured JSON")
	}
	if err := Validate(result, ev); err != nil {
		return Explanation{}, err
	}
	result.Scope, result.Limitations = ev.Scope, ev.Limitations
	graph := graphOrFallback(nil, ev)
	result.Graph = &graph
	return result, nil
}
