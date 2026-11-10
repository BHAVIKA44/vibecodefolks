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
	var stderr, stdout bytes.Buffer
	cmd.Stderr = &limitedDiagnostics{buffer: &stderr, maxBytes: 16384}
	cmd.Stdout = &limitedDiagnostics{buffer: &stdout, maxBytes: 16384}
	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return "", errors.New("Codex request canceled")
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", errors.New("Codex request timed out. Try again with a smaller change")
		}
		// Only fixed classifications leave this process; output can contain source or credentials.
		if classified := classifyCodexDiagnostic(stderr.String() + "\n" + stdout.String()); classified != "" {
			return "", errors.New(classified)
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

func classifyCodexDiagnostic(raw string) string {
	diagnostic := strings.ToLower(raw)
	switch {
	case strings.Contains(diagnostic, "not logged in") || strings.Contains(diagnostic, "please run codex login") || strings.Contains(diagnostic, "sign-in is required"):
		return "Codex reports that sign-in is required. Run codex login and try again"
	case strings.Contains(diagnostic, "usage limit") || strings.Contains(diagnostic, "rate limit") || strings.Contains(diagnostic, "quota") || strings.Contains(diagnostic, "purchase more credits"):
		return "Codex reported a usage or rate limit. Wait for the limit to reset, switch models, or configure another provider"
	case strings.Contains(diagnostic, "model is not supported") || strings.Contains(diagnostic, "model not found") || strings.Contains(diagnostic, "invalid model"):
		return "Codex reported that the selected model is not supported for this account. Clear the optional model field or choose a supported model"
	default:
		return ""
	}
}

// Capture bounded tail diagnostics solely for local classification, without exposing them.
type limitedDiagnostics struct {
	buffer   *bytes.Buffer
	maxBytes int
}

func (w *limitedDiagnostics) Write(p []byte) (int, error) {
	n := len(p)
	if w.maxBytes <= 0 {
		return n, nil
	}
	if len(p) >= w.maxBytes {
		w.buffer.Reset()
		_, _ = w.buffer.Write(p[len(p)-w.maxBytes:])
		return n, nil
	}
	if overflow := w.buffer.Len() + len(p) - w.maxBytes; overflow > 0 {
		keep := append([]byte(nil), w.buffer.Bytes()[overflow:]...)
		w.buffer.Reset()
		_, _ = w.buffer.Write(keep)
	}
	_, _ = w.buffer.Write(p)
	return n, nil
}
