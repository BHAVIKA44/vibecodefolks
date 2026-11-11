package explain

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHarnessSafetyFlagsAreNeverPassed(t *testing.T) {
	dummySchema := []byte(`{"type":"object"}`)
	for _, h := range SupportedHarnesses {
		for _, model := range []string{"", "custom-model"} {
			args := harnessArgs(h, model, dummySchema, "/tmp", map[string]bool{"--no-session-persistence": true, "--json-schema": true})
			cmdLine := " " + strings.Join(args, " ") + " "
			for _, forbidden := range forbiddenHarnessFlags {
				if strings.Contains(cmdLine, " "+forbidden+" ") || strings.Contains(cmdLine, " "+forbidden+"=") {
					t.Fatalf("harness %q with model %q passes dangerous flag %q: %v", h, model, forbidden, args)
				}
			}
		}
	}
}

func TestClaudeCodeHarnessMockExecutionAndValidation(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "claude")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	response := answer("main.go")
	// Claude Code emits JSON envelope with structured_output or result.
	source := fmt.Sprintf(`package main
import ("os"; "fmt")
func main() {
	for _, a := range os.Args {
		if a == "--help" {
			os.Stdout.WriteString("--print --output-format --tools --permission-mode --strict-mcp-config --model --json-schema --no-session-persistence\n")
			return
		}
		if a == "--version" {
			os.Stdout.WriteString("claude-code 0.2.29\n")
			return
		}
	}
	if os.Getenv("VIBE_TEST_FAIL") == "auth" {
		os.Stderr.WriteString("Please run /login to authenticate. secret-token-should-not-escape\n")
		os.Exit(1)
	}
	if os.Getenv("VIBE_TEST_FAIL") == "quota" {
		os.Stderr.WriteString("credit balance is too low to complete this request. secret-token-should-not-escape\n")
		os.Exit(1)
	}
	if os.Getenv("VIBE_TEST_FAIL") == "model" {
		os.Stderr.WriteString("model_not_found: The model does not exist. secret-token-should-not-escape\n")
		os.Exit(1)
	}
	// Verify strict read-only execution flags.
	foundTools, foundPerm := false, false
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--tools" && i+1 < len(os.Args) && os.Args[i+1] == "" { foundTools = true }
		if os.Args[i] == "--permission-mode" && i+1 < len(os.Args) && os.Args[i+1] == "default" { foundPerm = true }
	}
	if !foundTools || !foundPerm {
		os.Stderr.WriteString("missing required read-only flags\n")
		os.Exit(2)
	}
	fmt.Printf("{\"type\":\"result\",\"subtype\":\"success\",\"is_error\":false,\"result\":%%q,\"modelUsage\":{\"claude-3-7-sonnet\":{}}}\n", %q)
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
		t.Fatalf("build Claude mock: %v %s", err, output)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg, err := NewHarnessConfig("claude")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Model = "claude-3-7-sonnet"
	got, err := ProviderFor(cfg).Explain(context.Background(), fixture())
	if err != nil || got.Summary != "A change" || len(got.Changes) != 1 || got.ModelUsed != "claude-3-7-sonnet" {
		t.Fatalf("claude explanation: %+v err=%v", got, err)
	}

	for _, tc := range []struct{ mode, expected string }{
		{"auth", "sign-in is required"},
		{"quota", "usage or rate limit"},
		{"model", "selected model is not supported"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			t.Setenv("VIBE_TEST_FAIL", tc.mode)
			_, err := ProviderFor(cfg).Explain(context.Background(), fixture())
			if err == nil || !strings.Contains(err.Error(), tc.expected) || strings.Contains(err.Error(), "secret-token") {
				t.Fatalf("mode %s: unsafe or unexpected error: %v", tc.mode, err)
			}
		})
	}
}

func TestGeminiCLIHarnessMockExecutionAndValidation(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "gemini")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	response := answer("main.go")
	// Gemini CLI emits JSON envelope with response and stats.models.
	source := fmt.Sprintf(`package main
import ("os"; "fmt")
func main() {
	for _, a := range os.Args {
		if a == "--help" {
			os.Stdout.WriteString("--prompt --output-format --approval-mode --model\n")
			return
		}
		if a == "--version" {
			os.Stdout.WriteString("gemini-cli 0.1.5\n")
			return
		}
	}
	if os.Getenv("VIBE_TEST_FAIL") == "auth" {
		os.Stderr.WriteString("Please set the GEMINI_API_KEY environment variable. secret-key-should-not-escape\n")
		os.Exit(1)
	}
	if os.Getenv("VIBE_TEST_FAIL") == "quota" {
		os.Stderr.WriteString("ResourceExhausted: Quota exceeded for model. secret-key-should-not-escape\n")
		os.Exit(1)
	}
	if os.Getenv("VIBE_TEST_FAIL") == "model" {
		os.Stderr.WriteString("models/unknown-model is not found for api version. secret-key-should-not-escape\n")
		os.Exit(1)
	}
	foundAppr := false
	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--approval-mode" && i+1 < len(os.Args) && os.Args[i+1] == "default" { foundAppr = true }
	}
	if !foundAppr {
		os.Stderr.WriteString("missing approval-mode default\n")
		os.Exit(2)
	}
	fmt.Printf("{\"response\":%%q,\"stats\":{\"models\":{\"gemini-2.5-flash\":{}}}}\n", %q)
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
		t.Fatalf("build Gemini mock: %v %s", err, output)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg, err := NewHarnessConfig("gemini")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ProviderFor(cfg).Explain(context.Background(), fixture())
	if err != nil || got.Summary != "A change" || len(got.Changes) != 1 || got.ModelUsed != "gemini-2.5-flash" {
		t.Fatalf("gemini explanation: %+v err=%v", got, err)
	}

	for _, tc := range []struct{ mode, expected string }{
		{"auth", "sign-in is required"},
		{"quota", "usage or rate limit"},
		{"model", "selected model is not supported"},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			t.Setenv("VIBE_TEST_FAIL", tc.mode)
			_, err := ProviderFor(cfg).Explain(context.Background(), fixture())
			if err == nil || !strings.Contains(err.Error(), tc.expected) || strings.Contains(err.Error(), "secret-key") {
				t.Fatalf("mode %s: unsafe or unexpected error: %v", tc.mode, err)
			}
		})
	}
}

func TestProbeHarnessFlagsValidation(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "codex")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	// Missing --output-schema flag.
	source := `package main
import "os"
func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" { os.Stdout.WriteString("codex-cli 0.100.0\n"); return }
	os.Stdout.WriteString("--sandbox --ephemeral --model\n")
}`
	goTool, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(dir, "fixture.go")
	if err := os.WriteFile(sourcePath, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command(goTool, "build", "-o", binary, sourcePath).CombinedOutput(); err != nil {
		t.Fatalf("build broken CLI: %v %s", err, output)
	}
	t.Setenv("PATH", dir)

	status := ProbeHarness(context.Background(), "codex")
	if !status.Installed {
		t.Fatalf("expected installed, got %+v", status)
	}
	if status.Supported {
		t.Fatalf("expected unsupported due to missing required flags, got %+v", status)
	}
	if !strings.Contains(status.Reason, "--output-schema") {
		t.Fatalf("expected reason to mention missing flag, got %q", status.Reason)
	}
}
