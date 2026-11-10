# Multi-Model Compatibility Matrix & Provider Architecture

This document defines the provider and model capability architecture in VibeCodeFolks, detailing model discovery, structured response compatibility, parameter handling, and live inference verification status.

## 1. Compatibility Matrix

| Provider / Harness | Model | Discovery Method | Structured Output Mode | Reasoning / Parameter Flags | Explanation Status | Dedicated Quiz Status | Verification Tier | Notes & Known Limitations |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Groq** | `openai/gpt-oss-120b` | `GET /openai/v1/models` | `strict_json_schema` | `include_reasoning: false`, `reasoning_effort: "low"`, `max_completion_tokens: 2000` | **PASS** (5/5 scenarios) | **PASS** (3 scenarios verified) | **LIVE-VERIFIED** | High accuracy. Free-tier Groq quota has 200k tokens/day limit; returns HTTP 429 when quota is reached. |
| **Groq** | `qwen/qwen3.8-27b` | `GET /openai/v1/models` | `strict_json_schema` | `reasoning_format: "hidden"`, `max_completion_tokens: 0` (omitted) | **PASS** (Chrome verified, 4.2s, 14 citations) | **PASS** (Chrome verified, 4.4s, 3/3 valid questions) | **LIVE-VERIFIED** | Explicit `max_completion_tokens: 2000` is rejected by Groq's 1000 OTPM check. Omitting the field allows full structured output. Do not set `reasoning_effort: "none"`. |
| **Groq** | `openai/gpt-oss-20b` | `GET /openai/v1/models` | `strict_json_schema` | `include_reasoning: false`, `reasoning_effort: "low"` | **FAIL / UNRELIABLE** | **FAIL / UNRELIABLE** | **LIVE-TESTED (FAILED)** | Intermittently violates JSON schema syntax on Groq (`failed_generation` emits raw strings in object lists or drops required `graph` property). Recommends 120b or Qwen instead. |
| **Groq** | `whisper-large-v3`, `llama-guard`, `canopylabs/orpheus` | `GET /openai/v1/models` | N/A | N/A | **REJECTED** | **REJECTED** | **LIVE-VERIFIED** | Filtered by model discovery and rejected on selection as non-chat/audio/guardrail models with actionable guidance. |
| **Local CLI** | `Codex CLI` (`codex-cli 0.166.0`) | `exec.LookPath("codex")` | CLI `--output-schema` | Read-only sandbox, non-interactive execution | **PASS** (native harness tests) | **PASS** (native harness tests) | **MOCK-TESTED & LIVE-PROBED** | Discovered on PATH. Live probe against local Codex backend returned rate limit, which was classified into friendly user guidance without escaping secrets. |
| **OpenAI** | `gpt-4o`, `gpt-4.1`, `gpt-4o-mini` | `GET /v1/models` | `strict_json_schema` | Standard parameters, `temperature: 0` | **PASS** (contract verified) | **PASS** (contract verified) | **IMPLEMENTED (UNVERIFIED LIVE)** | Full API endpoint and schema mapping implemented; requires user OpenAI API key for live inference. |
| **OpenAI** | `o1`, `o3`, `o3-mini`, `o4-mini` | `GET /v1/models` | `strict_json_schema` | `SupportsTemperature: false` (`temperature` omitted) | **PASS** (contract verified) | **PASS** (contract verified) | **IMPLEMENTED (UNVERIFIED LIVE)** | Temperature setting omitted to comply with OpenAI reasoning model requirements. |
| **Gemini** | `gemini-2.5-flash`, `gemini-2.5-pro` | `GET .../models` | `json_object` | Standard OpenAI-compatible format | **PASS** (contract verified) | **PASS** (contract verified) | **IMPLEMENTED (UNVERIFIED LIVE)** | Direct OpenAI-compatible endpoint implemented; earlier revision tested on Gemini. |
| **xAI** | `grok-2`, `grok-beta` | `GET /v1/models` | `responses` endpoint / `json_object` | `/v1/responses` format with `store: false` | **PASS** (contract verified) | **PASS** (contract verified) | **IMPLEMENTED (UNVERIFIED LIVE)** | Responses API payload format fully tested. |
| **Ollama** | Local models (`llama3.2`, `qwen2.5-coder`) | `GET http://localhost:11434/v1/models` | `json_object` | Local loopback transport, no auth required | **PASS** (contract verified) | **PASS** (contract verified) | **IMPLEMENTED (UNVERIFIED LIVE)** | Tested error handling when Ollama daemon is offline; provides clean setup instructions. |
| **LM Studio** | Local models | `GET http://localhost:1234/v1/models` | `strict_json_schema` | Local loopback transport, no auth required | **PASS** (contract verified) | **PASS** (contract verified) | **IMPLEMENTED (UNVERIFIED LIVE)** | Uses `strict_json_schema` as required by LM Studio's OpenAI-compatible server. |

---

## 2. Model-Agnostic Capability Architecture

The separation of model concerns is handled in `internal/explain/capabilities.go`:

1. **Structured Mode Selection**:
   - `strict_json_schema`: Uses OpenAI/Groq strict JSON Schema validation (`response_format: { type: "json_schema", json_schema: { strict: true, ... } }`).
   - `json_object`: Used for providers or local models that do not support strict schema validation or fail constrained decoding.

2. **Schema Ordering (`orderedObject`)**:
   - JSON Schema property ordering is guaranteed using Go struct definitions instead of unordered `map[string]any`.
   - Ensures models generate `text`, `options`, `explanation`, and `option_explanations` *before* `correct_index`, preventing index misalignment during streaming generation.

3. **Reasoning Parameters**:
   - Groq GPT-OSS models receive `include_reasoning: false` and `reasoning_effort: "low"`.
   - Groq Qwen models receive `reasoning_format: "hidden"` and omit `max_completion_tokens` to prevent OTPM rejections.
   - OpenAI reasoning models (`o1`, `o3`) automatically omit the `temperature` parameter.
   - Non-reasoning models receive no proprietary reasoning fields.

4. **Model Discovery & Filtering**:
   - Non-billable `GET /models` inspection.
   - Automatic exclusion of non-chat models (audio models like `whisper*`, `canopylabs/orpheus*`, safety guardrails like `prompt-guard*`, embeddings like `*embedding*`, and image models like `dall-e*`).
   - Attempting to configure a non-chat model produces clear, actionable UI guidance.

5. **Provider Session Memory**:
   - API keys and verification statuses are retained in server memory for the duration of the session.
   - Users can switch between models and providers without having to re-enter credentials each time.
