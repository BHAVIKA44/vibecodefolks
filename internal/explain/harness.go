package explain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/BHAVIKA44/vibecodefolks/internal/analysis"
)

// HarnessProvider follows the local-harness model: the selected CLI owns its
// authentication and model choice. VibeCodeFolks supplies only the reviewed
// evidence and validates the returned JSON before displaying it.
type HarnessProvider struct{ name, model string }

func NewHarnessProvider(name string) *HarnessProvider { return &HarnessProvider{name: name} }

func (p *HarnessProvider) Explain(ctx context.Context, ev analysis.Evidence) (Explanation, error) {
	content, err := p.run(ctx, ev, instruction, explanationSchema)
	if err != nil {
		return Explanation{}, err
	}
	return decodeExplanation(content, ev)
}

func (p *HarnessProvider) run(ctx context.Context, ev analysis.Evidence, promptInstruction string, outputSchema map[string]any) (string, error) {
	if p.name != "codex" {
		return "", errors.New("unsupported local harness")
	}
	binary, err := exec.LookPath("codex")
	if err != nil {
		return "", errors.New("Codex CLI is not installed or available on PATH")
	}
	evidence, err := json.Marshal(struct {
		Scope       string                  `json:"scope"`
		Files       []analysis.EvidenceFile `json:"files"`
		Limitations []string                `json:"limitations"`
	}{ev.Scope, ev.Files, ev.Limitations})
	if err != nil {
		return "", errors.New("could not encode evidence")
	}
	schema, err := json.Marshal(outputSchema)
	if err != nil {
		return "", errors.New("could not create local harness schema")
	}
	dir, err := os.MkdirTemp("", "vibecodefolks-codex-")
	if err != nil {
		return "", errors.New("could not prepare local harness")
	}
	defer os.RemoveAll(dir)
	schemaPath, outputPath := filepath.Join(dir, "schema.json"), filepath.Join(dir, "answer.json")
	if os.WriteFile(schemaPath, schema, 0600) != nil {
		return "", errors.New("could not prepare local harness")
	}
	prompt := promptInstruction + "\n\nReviewed evidence JSON follows. Return the requested JSON only.\n" + string(evidence)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	args := []string{"exec", "--ephemeral", "--skip-git-repo-check", "-s", "read-only", "--output-schema", schemaPath, "-o", outputPath}
	if p.model != "" {
		args = append(args, "--model", p.model)
	}
	args = append(args, "-")
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(prompt)
	var stderr bytes.Buffer
	cmd.Stderr = &limitedDiagnostics{buffer: &stderr, remaining: 16384}
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return "", errors.New("Codex request canceled")
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", errors.New("Codex request timed out. Try again with a smaller change")
		}
		// Only fixed classifications leave this process; stderr can contain source or credentials.
		diagnostic := strings.ToLower(stderr.String())
		if strings.Contains(diagnostic, "not logged in") || strings.Contains(diagnostic, "please run codex login") {
			return "", errors.New("Codex reports that sign-in is required. Run codex login and try again")
		}
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return "", errors.New("Codex exited without generating an explanation. Check its login and model configuration, then retry")
		}
		return "", errors.New("Could not start the Codex CLI. Check its installation and try again")
	}
	file, err := os.Open(outputPath)
	if err != nil {
		return "", errors.New("Codex returned no explanation")
	}
	defer file.Close()
	const maxOutput = 1024 * 1024
	content, err := io.ReadAll(io.LimitReader(file, maxOutput+1))
	if err != nil || len(content) > maxOutput {
		return "", errors.New("Codex output unavailable or too large")
	}
	return string(content), nil
}

// Capture bounded diagnostics solely for local classification, without exposing them.
type limitedDiagnostics struct {
	buffer    *bytes.Buffer
	remaining int
}

func (w *limitedDiagnostics) Write(p []byte) (int, error) {
	n := len(p)
	if len(p) > w.remaining {
		p = p[:w.remaining]
	}
	_, _ = w.buffer.Write(p)
	w.remaining -= len(p)
	return n, nil
}
