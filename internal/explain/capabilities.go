package explain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// ModelCapabilities separates provider- and model-specific request formatting,
// structured-output modes, reasoning flags, and token budgets from shared
// explanation and quiz validation logic.
type ModelCapabilities struct {
	Provider            string
	Transport           string // "codex_cli", "chat_completions", "responses"
	StructuredMode      string // "strict_json_schema", "json_object"
	SupportsTemperature bool
	MaxCompletionTokens int
	ReasoningFields     map[string]any
	EvidenceByteBudget  int
	UnsupportedReason   string
}

func ProviderID(endpoint, harness string) string {
	if harness != "" {
		return strings.ToLower(strings.TrimSpace(harness))
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return "custom"
	}
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	switch {
	case host == "api.groq.com":
		return "groq"
	case host == "api.openai.com":
		return "openai"
	case host == "generativelanguage.googleapis.com":
		return "gemini"
	case host == "api.x.ai":
		return "xai"
	case (host == "localhost" || host == "127.0.0.1") && port == "11434":
		return "ollama"
	case (host == "localhost" || host == "127.0.0.1") && port == "1234":
		return "lmstudio"
	default:
		return "custom"
	}
}

func CapabilitiesFor(cfg Config) ModelCapabilities {
	provider := ProviderID(cfg.Endpoint, cfg.Harness)
	model := strings.TrimSpace(cfg.Model)
	lowerModel := strings.ToLower(model)

	if cfg.Harness != "" {
		return ModelCapabilities{
			Provider:           provider,
			Transport:          "codex_cli",
			StructuredMode:     "strict_json_schema",
			EvidenceByteBudget: 512 * 1024,
		}
	}

	transport := "chat_completions"
	if strings.HasSuffix(strings.TrimRight(strings.ToLower(cfg.Endpoint), "/"), "/responses") {
		transport = "responses"
	}

	caps := ModelCapabilities{
		Provider:            provider,
		Transport:           transport,
		StructuredMode:      "json_object",
		SupportsTemperature: true,
		EvidenceByteBudget:  512 * 1024,
	}

	if reason := nonChatModelReason(lowerModel); reason != "" {
		caps.UnsupportedReason = reason
		return caps
	}

	switch provider {
	case "groq":
		caps.EvidenceByteBudget = 24 * 1024
		caps.MaxCompletionTokens = 2000
		if groqSupportsStrictSchema(lowerModel) {
			caps.StructuredMode = "strict_json_schema"
		}
		switch {
		case strings.HasPrefix(lowerModel, "openai/gpt-oss-"):
			caps.ReasoningFields = map[string]any{
				"include_reasoning": false,
				"reasoning_effort":  "low",
			}
		case strings.HasPrefix(lowerModel, "qwen/"):
			// Live Groq results (2026-11-10): an explicit max_completion_tokens of 2000
			// is rejected up front by the free-tier output-tokens-per-minute check
			// (limit 1000), while 1000 truncates the structured document. Omitting the
			// field lets Groq size the output. reasoning_effort "none" produced
			// degenerate, unquoted JSON, so only the reasoning text is hidden.
			caps.MaxCompletionTokens = 0
			caps.ReasoningFields = map[string]any{
				"reasoning_format": "hidden",
			}
		case strings.HasPrefix(lowerModel, "deepseek-r1"):
			caps.ReasoningFields = map[string]any{
				"reasoning_format": "hidden",
			}
		}
	case "openai":
		if isOpenAIReasoningModel(lowerModel) {
			caps.SupportsTemperature = false
		}
		if transport == "chat_completions" && openAISupportsStrictSchema(lowerModel) {
			caps.StructuredMode = "strict_json_schema"
		}
	case "lmstudio":
		// LM Studio's OpenAI-compatible server accepts json_schema and rejects json_object.
		if transport == "chat_completions" {
			caps.StructuredMode = "strict_json_schema"
		}
	}

	return caps
}

func ValidateModelChoice(cfg Config) error {
	caps := CapabilitiesFor(cfg)
	if caps.UnsupportedReason != "" {
		return errors.New(caps.UnsupportedReason)
	}
	return nil
}

func nonChatModelReason(lowerModel string) string {
	if lowerModel == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(lowerModel, "whisper"):
		return fmt.Sprintf("Model %q is a speech-to-text audio model and cannot generate code explanations or quizzes. Choose a chat or code model", lowerModel)
	case strings.HasPrefix(lowerModel, "canopylabs/orpheus") || strings.HasPrefix(lowerModel, "tts-") || strings.Contains(lowerModel, "-tts"):
		return fmt.Sprintf("Model %q is a text-to-speech audio model and cannot generate code explanations or quizzes. Choose a chat or code model", lowerModel)
	case strings.Contains(lowerModel, "prompt-guard") || strings.Contains(lowerModel, "safeguard") || strings.Contains(lowerModel, "llama-guard") || strings.Contains(lowerModel, "moderation"):
		return fmt.Sprintf("Model %q is a safety classifier/guardrail model and cannot generate structured code explanations or quizzes. Choose a chat or code model", lowerModel)
	case strings.Contains(lowerModel, "embedding") || strings.HasPrefix(lowerModel, "text-embedding"):
		return fmt.Sprintf("Model %q is an embedding model and cannot generate code explanations or quizzes. Choose a chat or code model", lowerModel)
	case strings.HasPrefix(lowerModel, "dall-e") || strings.HasPrefix(lowerModel, "imagen") || strings.HasPrefix(lowerModel, "veo"):
		return fmt.Sprintf("Model %q is a media generation model and cannot generate code explanations or quizzes. Choose a chat or code model", lowerModel)
	default:
		return ""
	}
}

func groqSupportsStrictSchema(lowerModel string) bool {
	if lowerModel == "" {
		return true
	}
	for _, prefix := range []string{
		"openai/gpt-oss-120b",
		"openai/gpt-oss-20b",
		"qwen/qwen3",
		"moonshotai/kimi-k2",
		"meta-llama/llama-4-scout",
		"meta-llama/llama-4-maverick",
	} {
		if strings.HasPrefix(lowerModel, prefix) {
			return true
		}
	}
	return false
}

func isOpenAIReasoningModel(lowerModel string) bool {
	for _, prefix := range []string{"o1", "o3", "o4", "gpt-5"} {
		if lowerModel == prefix || strings.HasPrefix(lowerModel, prefix+"-") {
			return true
		}
	}
	return false
}

func openAISupportsStrictSchema(lowerModel string) bool {
	for _, prefix := range []string{"gpt-4o", "gpt-4.1", "o3", "o4", "gpt-5"} {
		if lowerModel == prefix || strings.HasPrefix(lowerModel, prefix+"-") {
			return true
		}
	}
	return false
}

// ModelsEndpoint derives the read-only OpenAI-compatible GET /models URL from
// a configured chat/responses endpoint on the same origin.
func ModelsEndpoint(endpoint string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("invalid provider endpoint")
	}
	cleanPath := strings.TrimRight(u.Path, "/")
	switch {
	case strings.HasSuffix(cleanPath, "/chat/completions"):
		u.Path = strings.TrimSuffix(cleanPath, "/chat/completions") + "/models"
	case strings.HasSuffix(cleanPath, "/responses"):
		u.Path = strings.TrimSuffix(cleanPath, "/responses") + "/models"
	default:
		return "", errors.New("model discovery is not supported for this endpoint path; enter a model ID manually")
	}
	return u.String(), nil
}

// DiscoverModels queries the provider's non-billable GET /models endpoint to list
// available chat/code models without making an inference request.
func DiscoverModels(ctx context.Context, cfg Config) ([]string, error) {
	modelsURL, err := ModelsEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	p := NewProvider(cfg)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, modelsURL, nil)
	if err != nil {
		return nil, errors.New("invalid model discovery request")
	}
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	res, err := p.Client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, errors.New("Model discovery timed out. Check that the provider is reachable")
		}
		if cfg.Local {
			return nil, errors.New("Could not connect to the local model server. Make sure Ollama or LM Studio is running on this machine")
		}
		return nil, errors.New("Could not reach the AI provider to list models")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		switch res.StatusCode {
		case http.StatusUnauthorized:
			return nil, errors.New("AI provider rejected the API key (HTTP 401). Enter a valid API key and try again")
		case http.StatusForbidden:
			return nil, errors.New("AI provider denied model listing (HTTP 403). Check the API key permissions")
		case http.StatusNotFound:
			return nil, errors.New("Model discovery endpoint not found (HTTP 404). Enter a model ID manually")
		case http.StatusTooManyRequests:
			return nil, errors.New("AI provider rate limit reached (HTTP 429). Wait briefly or enter a model ID manually")
		default:
			return nil, fmt.Errorf("AI provider model discovery returned HTTP %d", res.StatusCode)
		}
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 512*1024+1))
	if err != nil || len(raw) > 512*1024 {
		return nil, errors.New("Model list response unavailable or too large")
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Models []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"models"`
	}
	if json.Unmarshal(raw, &payload) != nil {
		return nil, errors.New("AI provider returned an unrecognized model list format")
	}
	seen := map[string]bool{}
	var models []string
	addModel := func(id string) {
		id = strings.TrimSpace(id)
		id = strings.TrimPrefix(id, "models/")
		if id == "" || seen[id] || len(id) > 200 || strings.ContainsAny(id, "\r\n") {
			return
		}
		if nonChatModelReason(strings.ToLower(id)) != "" {
			return
		}
		seen[id] = true
		models = append(models, id)
	}
	for _, m := range payload.Data {
		addModel(m.ID)
	}
	for _, m := range payload.Models {
		if m.ID != "" {
			addModel(m.ID)
		} else {
			addModel(m.Name)
		}
	}
	provider := ProviderID(cfg.Endpoint, cfg.Harness)
	sort.Slice(models, func(i, j int) bool {
		ri, rj := modelPriority(provider, models[i]), modelPriority(provider, models[j])
		if ri != rj {
			return ri < rj
		}
		return models[i] < models[j]
	})
	return models, nil
}

func modelPriority(provider, model string) int {
	lower := strings.ToLower(model)
	switch provider {
	case "groq":
		switch {
		case lower == "openai/gpt-oss-120b":
			return 1
		case strings.HasPrefix(lower, "qwen/qwen3"):
			return 2
		case lower == "openai/gpt-oss-20b":
			return 3
		case strings.Contains(lower, "llama-3.3-70b") || strings.Contains(lower, "llama-4"):
			return 4
		case strings.Contains(lower, "kimi-k2"):
			return 5
		default:
			return 10
		}
	case "openai":
		switch {
		case lower == "gpt-4.1-mini" || lower == "gpt-4.1" || lower == "gpt-4o":
			return 1
		case strings.HasPrefix(lower, "gpt-4.1") || strings.HasPrefix(lower, "gpt-4o") || strings.HasPrefix(lower, "o4-mini") || strings.HasPrefix(lower, "o3"):
			return 2
		default:
			return 10
		}
	case "gemini":
		switch {
		case strings.HasPrefix(lower, "gemini-2.5-flash") || strings.HasPrefix(lower, "gemini-2.5-pro"):
			return 1
		case strings.HasPrefix(lower, "gemini-2.0-flash"):
			return 2
		default:
			return 10
		}
	default:
		return 10
	}
}
