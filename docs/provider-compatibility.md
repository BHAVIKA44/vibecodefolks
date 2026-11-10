# Provider compatibility

Discovery checks executables on the server's PATH. It does not test login, model access, or whether an installed tool can safely run. Setup does not trigger an inference request. Codex is the only implemented coding harness. It runs from a temporary directory with its CLI read-only sandbox, an output schema, a timeout, and the user's existing CLI sign-in. An empty model uses the CLI's configuration. A model ID entered in the dashboard is forwarded to the CLI. The temporary directory is not an operating-system isolation boundary for arbitrary executables.

| Harness | Executable discovery | Model and authentication | Explanation and quiz | Testing |
| --- | --- | --- | --- | --- |
| Codex | codex | CLI default or custom ID; existing CLI login | Structured schema, validated against evidence | Fictional checkout live inference in previous run; native mock executable and schema test |
| Claude Code | claude | Unavailable | Not implemented | Discovery only |
| Gemini CLI | gemini | Unavailable | Not implemented | Discovery only |
| Cursor Agent | cursor-agent | Unavailable | Not implemented | Discovery only |
| Antigravity | agy | Unavailable | Not implemented | Discovery only |
| OpenCode | opencode | Unavailable | Not implemented | Discovery only |
| Aider | aider | Unavailable | Not implemented | Discovery only |
| Goose | goose | Unavailable | Not implemented | Discovery only |

Other harnesses need a verified noninteractive read-only invocation, authentication and model behavior, cancellation, and structured response integration before they can be enabled. PATH discovery does not prove those properties.

| Direct provider | Endpoint family | Model and authentication | Output and test status |
| --- | --- | --- | --- |
| Groq | Chat completions | Editable model ID, session API key | Strict schema for Groq, generic mocked HTTP coverage, not provider-specific contract verification; earlier live attempts returned 400 and 429 |
| OpenAI | Chat completions | Editable model ID, session API key | JSON object requested, generic mocked HTTP coverage, not provider-specific contract verification; live inference unverified |
| Gemini | OpenAI-compatible chat completions | Editable model ID, session API key | JSON object requested, generic mocked HTTP coverage, not provider-specific contract verification; earlier Gemini sample on a prior revision, current live inference unverified |
| xAI | Responses API | Editable model ID, session API key | JSON object requested, generic mocked HTTP coverage, not provider-specific contract verification; live inference unverified |
| Ollama | Loopback chat completions | Editable model ID, optional key | JSON object requested, generic mocked HTTP coverage, not provider-specific contract verification; live inference unverified |
| LM Studio | Loopback chat completions | Editable model ID, optional key | JSON object requested, generic mocked HTTP coverage, not provider-specific contract verification; live inference unverified |

Structured responses and citations are validated locally, but schema compliance and line references do not establish factual or semantic accuracy. Availability of each endpoint, model and response format depends on the running provider. No automatic inference, model fallback or credential validation request is made during configuration.
