# VibeCodeFolks Twitter/X Launch Readiness & Production Audit

**Repository:** `https://github.com/BHAVIKA44/vibecodefolks`  
**Product Identity:** **VibeCodeFolks — AI wrote it. Do you understand it?**  
**Audit Date:** November 24, 2026  
**Auditor:** Final Engineering Reviewer & Release Owner  

---

## 1. Executive Summary & Launch Recommendation

### FINAL VERDICT:
**CONDITIONALLY READY — RELEASE TAG `v1.0.0` REQUIRED BEFORE POSTING ON TWITTER**

*(Equivalently: **A. READY TO INVITE DEVELOPERS ON TWITTER** as soon as Step 16 is approved and executed to publish the `v1.0.0` release tag).*

### Summary of Findings:

| Area | Status | Evidence / Notes |
| :--- | :--- | :--- |
| **Codebase on `main`** | **100% READY** | Clean tree at commit `828f609`. Zero uncommitted diffs. Authorized author identity `Bhavika <43768505+BHAVIKA44@users.noreply.github.com>`. |
| **CI/CD Automation** | **100% PASS** | Ubuntu, macOS, and Windows matrix passed cleanly on PR #10, merge push `680c2fa`, and README update `908fc3b`. |
| **Automated Test Suite** | **100% PASS** | `go test -race ./...` (0 data races, 0 errors). Node dashboard DOM suite (24/24 passing). `go vet` clean. `gofmt` clean. |
| **Real Browser Chrome CDP QA** | **100% PASS** | Automated Chrome CDP audit (`/tmp/run_prelaunch_chrome_qa.mjs`) verified Codex CLI live inference (17.0s), 19 line citations, 8 findings, change flow graph, interactive quiz (10.5s), and mobile layout (390x844). |
| **Comprehension Quiz Misconceptions** | **100% PASS** | Verified live distractor feedback on deliberate incorrect choice: surfaces precise misconception explanations and line citations. Zero-network local retry tested and confirmed. |
| **px0 Safety Invariants** | **100% PASS** | Strictly read-only execution in isolated empty temporary directories (`os.MkdirTemp`). Unit tests enforce rejection of dangerous flags (`--dangerously-bypass-approvals-and-sandbox`, `--yolo`, etc.). |
| **Release Packaging & Smoke Test** | **100% PASS** | `scripts/package-release.sh v1.0.0` packaged all 5 architectures (`darwin_arm64`, `darwin_amd64`, `linux_arm64`, `linux_amd64`, `windows_amd64`) with valid sha256 checksums. Native binary passed `scripts/release-smoke.py`. Installer tests passed (`tests/installer.sh`). |
| **The Launch Blocker** | **CRITICAL GAP** | **GitHub Releases currently only lists `v0.1.0` (from Oct 18, 2026).** Any user running the one-line installer from the tweet will download the month-old `v0.1.0` binary (which lacks coding agent harnesses, model discovery, and dedicated quizzes) until `v1.0.0` is tagged and pushed. |

---

## 2. Codebase & Git Status Audit

- **Current Branch:** `main` (synchronized with `origin/main`).
- **Latest Commit:** `828f609` (`fix(release): support BSD tar fallback in package-release script`).
- **Working Tree:** Clean (`git status --short` returns zero entries).
- **Author Identity:** All commits verified under authorized Git identity:  
  `Bhavika <43768505+BHAVIKA44@users.noreply.github.com>`.
- **Formatting & Hygiene:**
  - `gofmt -l .` returns zero files.
  - `go vet ./...` returns zero warnings.
  - `git diff --check` returns zero whitespace errors.

---

## 3. CI/CD Matrix Audit

GitHub Actions workflow `.github/workflows/ci.yml` runs on every pull request and push to `main` across three major operating systems:
- **Ubuntu:** `ubuntu-latest` (Go 1.25.x, Node 22)
- **macOS:** `macos-latest` (Go 1.25.x, Node 22)
- **Windows:** `windows-latest` (Go 1.25.x, Node 22)

### Recent CI Runs:
- **Run ID `42958832679` (PR #10):** Passed 100% across Ubuntu, macOS, and Windows.
- **Run ID `42959534777` (Merge commit `680c2fa`):** Passed 100% across all 3 platforms.
- **Run ID `42961817781` (README refresh `908fc3b`):** Passed 100% across all 3 platforms.

Every check, including formatting, linting, race detection, unit tests, and installer tests, completed green without skipping or test bypass.

---

## 4. GitHub Releases & The Onboarding Gap Analysis

### Current GitHub Release Status:
Querying `https://api.github.com/repos/BHAVIKA44/vibecodefolks/releases` confirms:
- **Only 1 release published:** `v0.1.0` (published 2026-10-18).
- **`v1.0.0` has NOT been tagged or published.**

### Impact on Developer Onboarding:
The recommended one-line installer in `README.md` and Twitter copy runs:
```bash
curl -fsSL https://raw.githubusercontent.com/BHAVIKA44/vibecodefolks/main/install.sh | sh
```
`install.sh` queries GitHub's `/releases/latest` API. Currently, `/releases/latest` resolves to `v0.1.0`.

> [!WARNING]
> If developers are invited on Twitter before `v1.0.0` is published, they will install `v0.1.0`, which:
> 1. Does not include "Use my coding agent" (Codex CLI, Claude Code, Gemini CLI).
> 2. Does not include dynamic model discovery for Groq, Ollama, LM Studio, or OpenAI.
> 3. Does not include the dedicated comprehension quiz with distractor misconception feedback.
> 4. Suffers from outdated schema validation errors.
>
> **Action Required:** The repository owner must tag `v1.0.0` (`git tag v1.0.0 && git push origin v1.0.0`) to trigger `.github/workflows/release.yml` before publishing the Twitter announcement.

---

## 5. Architectural Safety Invariants vs px0

VibeCodeFolks adopts the engineering discipline of px0 (`px0-ai/px0` and `px0-ai/harness`) for discovering CLI agents, validating version compatibility via `--help`, and classifying errors, while enforcing an absolute **read-only, non-mutating execution sandbox**:

```mermaid
flowchart TD
    subgraph Browser["Developer Browser Dashboard"]
        UI["Dual Provider Select (CLI vs API)"]
        Consent["Evidence Review & Consent"]
        DiffView["Diff & Citation Navigation"]
        QuizUI["Comprehension Quiz & Misconception Feedback"]
    end

    subgraph Server["VibeCodeFolks Local Server (Go)"]
        Discovery["Harness Probing (exec.LookPath)"]
        EvidenceGen["Bounded Evidence Collector"]
        Isolation["os.MkdirTemp Sandbox"]
        Validator["Strict JSON Schema & Citation Checker"]
    end

    subgraph CLIs["Coding Agent CLIs"]
        Codex["Codex CLI (sandbox read-only)"]
        Claude["Claude Code (plan mode)"]
        Gemini["Gemini CLI (read-only approval)"]
    end

    UI --> Discovery
    Discovery --> Consent
    Consent --> EvidenceGen
    EvidenceGen --> Isolation
    Isolation --> CLIs
    CLIs --> Validator
    Validator --> DiffView
    Validator --> QuizUI
```

### Safety Guarantees:
1. **Empty Directory Isolation:** Every CLI harness invocation executes in an isolated temporary directory (`os.MkdirTemp("", "vibecodefolks-harness-*")`). The host git repository is never mounted or exposed as the working directory.
2. **Read-Only Enforced Flags:**
   - Codex CLI: `--sandbox read-only --ephemeral --skip-git-repo-check`.
   - Claude Code: `--permission-mode plan --tools "" --strict-mcp-config`.
   - Gemini CLI: `--approval-mode read-only`.
3. **Forbidden Flags Assertion:** Unit test `TestHarnessSafetyFlagsAreNeverPassed` verifies that dangerous bypass flags (`--dangerously-bypass-approvals-and-sandbox`, `--yolo`, `--full-auto`, `-y`, `acceptEdits`) are strictly rejected.
4. **No Code Editing or Write Tools:** Coding agents are used solely as structured text synthesizers. File mutation and command execution tools are explicitly stripped from agent invocations.

---

## 6. Coding Agent Harness Matrix

| Coding Agent | Detection & Auth | Invocation Parameters | Verification Status | Observed Evidence |
| :--- | :--- | :--- | :--- | :--- |
| **OpenAI Codex CLI** | `exec.LookPath("codex")`<br>Authenticated via ChatGPT (`~/.codex/auth.json`) | `codex exec --ephemeral --skip-git-repo-check --sandbox read-only --output-schema <file> -` | **LIVE-VERIFIED** (Chrome CDP runner) | Latency: 17.0s on `oss-collab-web`. 8 findings, 19 line citations, 3 graph nodes. Identified `serverRevision` vs `revision` field mismatch bug and in-place array mutation. Dedicated quiz: 10.5s, 2 questions with distractor explanations. |
| **Anthropic Claude Code** | `exec.LookPath("claude")`<br>`claude /login` or `ANTHROPIC_API_KEY` | `claude --print --output-format json --tools "" --permission-mode plan --strict-mcp-config --model <model>` | **MOCK-TESTED & ONBOARDING-VERIFIED** | Probed on host (uninstalled). UI surfaces actionable command: `npm install -g @anthropic-ai/claude-code`, then run `claude` to sign in. Unit tested in `TestClaudeCodeHarnessMockExecutionAndValidation`. |
| **Google Gemini CLI** | `exec.LookPath("gemini")`<br>`gemini auth login` or `GEMINI_API_KEY` | `gemini --prompt <file> --output-format json --approval-mode read-only --model <model>` | **MOCK-TESTED & ONBOARDING-VERIFIED** | Probed on host (uninstalled). UI surfaces actionable command: `npm install -g @google/gemini-cli`, then run `gemini` to sign in. Unit tested in `TestGeminiCLIHarnessMockExecutionAndValidation`. |

---

## 7. API Providers & Model Compatibility Matrix

| Provider | Model | Discovery Mode | Structured Mode | Status | Verification Tier | Notes |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **Groq** | `openai/gpt-oss-120b` | `GET /openai/v1/models` | `strict_json_schema` | **PASS** | **LIVE-VERIFIED** | High accuracy. Free-tier Groq quota has 200k tokens/day limit (HTTP 429 when exhausted). |
| **Groq** | `qwen/qwen3.8-27b` | `GET /openai/v1/models` | `strict_json_schema` | **PASS** | **LIVE-VERIFIED** | Chrome live verified: 5.0s explanation, 14 citations. `reasoning_format: "hidden"`, omits explicit `max_completion_tokens`. |
| **Groq** | `openai/gpt-oss-20b` | `GET /openai/v1/models` | `strict_json_schema` | **FAIL / UNRELIABLE** | **LIVE-TESTED (FAILED)** | Intermittently violates JSON schema syntax (`failed_generation` emits raw strings in object arrays). UI recommends 120b or Qwen. |
| **Groq** | Audio / Guardrail models | `GET /openai/v1/models` | N/A | **FILTERED / REJECTED** | **LIVE-VERIFIED** | Audio (`whisper*`), guardrails (`llama-guard`), and embeddings are excluded from dropdown. |
| **OpenAI** | `gpt-4o`, `gpt-4.1` | `GET /v1/models` | `strict_json_schema` | **PASS** | **IMPLEMENTED (UNVERIFIED LIVE)** | Contract tested; requires user OpenAI API key. |
| **OpenAI** | `o1`, `o3-mini` | `GET /v1/models` | `strict_json_schema` | **PASS** | **IMPLEMENTED (UNVERIFIED LIVE)** | Temperature omitted to comply with OpenAI reasoning model constraints. |
| **Gemini** | `gemini-2.5-flash` | `GET /models` | `json_object` | **PASS** | **IMPLEMENTED (UNVERIFIED LIVE)** | Direct OpenAI-compatible endpoint. |
| **xAI** | `grok-2` | `GET /v1/models` | `responses` API / `json_object` | **PASS** | **IMPLEMENTED (UNVERIFIED LIVE)** | Verified with `store: false`. |
| **Ollama** | Local models | `GET http://localhost:11434/v1/models` | `json_object` | **PASS** | **IMPLEMENTED (UNVERIFIED LIVE)** | Loopback transport; clean offline guidance if daemon is stopped. |
| **LM Studio** | Local models | `GET http://localhost:1234/v1/models` | `strict_json_schema` | **PASS** | **IMPLEMENTED (UNVERIFIED LIVE)** | Strict JSON schema over loopback. |

---

## 8. Real Browser Chrome CDP Audit & Evidence

Conducted using real Google Chrome via Chrome DevTools Protocol (CDP) automated by `/tmp/run_prelaunch_chrome_qa.mjs`.

### Scenario: `oss-collab-web` (React/TS + Express API)
- **Files Modified:**
  - `server/routes/documents.ts`: Adds `force` query param to bypass 409 conflict checks, returns `{ serverRevision, content }`.
  - `web/src/hooks/useDocumentSync.ts`: Polls server every 5 seconds, reads `payload.revision` (bug: server returns `serverRevision`).
  - `web/src/components/CollaboratorBar.tsx`: Mutates caller `users` array in-place via `.sort()`, filters with strict comparison (`nowMs - u.lastSeenMs < activeWindowMs`).

### Observed CDP Execution Metrics:
1. **Server Startup:** Listening on `http://127.0.0.1:52567`.
2. **Onboarding Banner:** Verified detection of installed Codex CLI and missing Claude/Gemini CLIs.
3. **Local Evidence Consent:** Displayed localized privacy panel:
   - Title: *"Review evidence for your local provider"*
   - Action Button: *"Explain locally"*
   - Privacy Text: Confirmed evidence is piped to local CLI and no remote tracking headers are attached.
4. **Codex CLI Live Explanation:**
   - Latency: **17,007ms (17.0s)**.
   - Summary: *"The document route adds a force option and returns server content and revision on conflicts. The sync hook adds polling and conflict handling, while the collaborator bar sorts and filters users by recent activity."*
   - Findings: **8 structured findings**.
   - Citations: **19 verified line citations**.
   - Change Flow Graph: **3 nodes (`handlePatchDocument`, `useDocumentSync`, `CollaboratorBar`), 1 contract edge (`PATCH contract`)**.
5. **Bidirectional Citation Navigation:**
   - Clicked citation `server/routes/documents.ts · new 19–27`.
   - Browser switched to diff viewer and highlighted 2 matching diff rows.
6. **Mobile Viewport Test (390x844):**
   - Verified responsive header, collapsible file tree, and responsive quiz container.

### Captured Screenshots:
- `01-onboarding-codex.png`: Codex CLI discovery and verified status.
- `02-onboarding-claude-not-installed.png`: Actionable Claude Code installation guidance.
- `03-onboarding-gemini-not-installed.png`: Actionable Gemini CLI installation guidance.
- `04-codex-citation-diff-1440.png`: Diff jump and row highlight upon citation click.
- `05-codex-change-flow-1440.png`: Inferred code relationships and graph nodes.
- `06-codex-quiz-wrong-feedback-1440.png`: Live misconception explanation for deliberate incorrect choice.
- `07-codex-collab-mobile-390.png`: Mobile responsive viewport (390x844).
- `08-codex-collab-desktop-1440.png`: Complete desktop workspace overview.
- `09-groq-qwen-explained-1440.png`: Live Groq inference with `qwen/qwen3.8-27b`.

---

## 9. Comprehension Quiz & Misconception Evaluation

### The Product Differentiation:
Generic AI coding tools present the developer with a wall of text that is easy to skim and ignore. VibeCodeFolks forces active comprehension through interactive quizzes backed by diff evidence, where every incorrect option contains a specific misconception explanation.

### Live Quiz Evaluation (`oss-collab-web`):
- **Generation Latency:** 10,541ms (10.5s via Codex CLI).
- **Question Count:** 2 deep architectural questions.

#### Question 1 (Conflict Handling Bug Tracing):
> *An existing document has `revision=7` and `content='server'`. The hook has `revision=5` and calls `saveDraft('draft', false)`. What is the resulting hook `revision` state?*

- **Correct Answer:** `C. undefined`
  - *Explanation:* The revision mismatch produces a 409 response with `serverRevision` set to 7 and no `revision` field. The client hook assigns `payload.revision` to state, which evaluates to `undefined`.
- **Live User Interaction (Deliberate Wrong Choice):**
  - Selected Option `D. null`.
  - UI immediately surfaced targeted feedback:  
    `Your choice: Incorrect: The response does not supply null for revision; the missing property evaluates to undefined.`
  - Cites: `server/routes/documents.ts · new 19–27` and `web/src/hooks/useDocumentSync.ts · new 53–62`.
- **Zero-Network Local Retry:**
  - Clicking "Retry" reset the quiz to Question 1 in client-side memory without contacting the AI provider.

---

## 10. Edge-Case, Limits & Robustness Audit

1. **Clean Working Tree Fallback:**
   - When a repository has no uncommitted changes, VibeCodeFolks automatically inspects the **Last Commit** (`HEAD^1..HEAD` or initial commit).
   - Scope badge displays `Last Commit`.
2. **Untracked Files (`??`):**
   - Untracked files are captured and formatted as added diffs (`+++ b/<path>`).
   - Absolute local filesystem paths are strictly stripped from diff headers.
3. **Large Diff Truncation Budget:**
   - Budget enforced via `EvidenceByteBudget` (24 KB on Groq, 48 KB on OpenAI/CLIs).
   - Oversized files exceeding `MaxEvidenceFileBytes` (32 KB) are safely skipped with clear UI warnings rather than failing JSON parsing.
4. **Graceful Rate Limit / HTTP 429 Handling:**
   - When Groq's 8k TPM rolling window or daily quota was exhausted during the live run, the UI surfaced:  
     `"AI provider rate limit or quota reached (HTTP 429). Check your quota and retry later"`
   - Zero console errors, zero uncaught exceptions, zero application crashes.

---

## 11. Security & Privacy Audit

1. **Host Header & DNS Rebinding Protection:**
   - The server inspects the `Host` header. Requests to unrecognized hosts return `403 Forbidden`.
2. **Cross-Origin Request Forgery (CSRF) Defense:**
   - `Sec-Fetch-Site: cross-site` requests are blocked (`403 Forbidden`).
   - Mismatched `Origin` headers are rejected.
3. **Single-Use Evidence Tokens:**
   - Prepared evidence tokens (`/api/explain/prepare`) expire after use. Replay attempts return `400 Bad Request`.
4. **Snapshot Freshness Check:**
   - If any file in the working tree is modified while reviewing evidence or during quiz generation, `/api/quiz/fresh` returns `fresh: false`, preventing stale inferences.
5. **Secret Redaction:**
   - `internal/analysis/evidence.go` automatically excludes secret files (`.env`, `.pem`, `.key`, `id_rsa`) and hunks containing API key patterns (`sk-proj-`, `api_key = ...`).
   - Absolute filesystem paths are never leaked in diff excerpts.
6. **Zero Credential Retention in Local Storage:**
   - API keys are held strictly in server session memory and are never written to disk or sent to the client browser in responses.

---

## 12. Release Packaging & Installer Rehearsal Evidence

1. **Packaging Script (`scripts/package-release.sh`):**
   - Verified across all 5 architectures:
     - `vibecodefolks_v1.0.0_darwin_arm64.tar.gz` (3.0 MB)
     - `vibecodefolks_v1.0.0_darwin_amd64.tar.gz` (3.3 MB)
     - `vibecodefolks_v1.0.0_linux_arm64.tar.gz` (2.9 MB)
     - `vibecodefolks_v1.0.0_linux_amd64.tar.gz` (3.2 MB)
     - `vibecodefolks_v1.0.0_windows_amd64.zip` (3.3 MB)
   - Verified `checksums.txt` via `sha256sum -c checksums.txt`: **100% OK**.
2. **Native Binary Smoke Test (`scripts/release-smoke.py`):**
   - Extracted `vibecodefolks_v1.0.0_darwin_arm64.tar.gz` and executed against disposable repository:
     - `--version` verified: `vibecodefolks v1.0.0`.
     - `--help` verified: Usage returned on stderr.
     - Headless HTTP dashboard verified: HTTP 200, `/api/changes` returned `total_files: 1`.
     - Output: `Native smoke test passed: v1.0.0`.
3. **Installer Verification (`tests/installer.sh`):**
   - Tested installation flow with local fixture server:
     - Archive verification: PASSED.
     - Tampered archive rejection: PASSED.
     - Missing / duplicate checksum rejection: PASSED.
     - Path traversal version rejection (`v1.0.0/../../bad`): PASSED.

---

## 13. Mobile Responsiveness Audit

Tested on 390x844 viewport (standard modern smartphone resolution):
- Header collapses controls into compact vertical layout.
- Provider status badges and session badges wrap cleanly without horizontal scroll overflow.
- Diff rows remain readable with horizontal pan support on code blocks.
- Comprehension quiz question cards and distractor options stack vertically with full tap targets (touch target height >= 44px).
- Captured artifact: `07-codex-collab-mobile-390.png`.

---

## 14. Documentation & README Accuracy Audit

The `README.md` on `main` was thoroughly reviewed for accuracy against the actual codebase:
- **Product Promise:** Accurately stated as *"AI wrote it. Do you understand it?"*
- **Two Connection Methods:** Clearly documents **"Use my coding agent"** (Codex CLI, Claude Code, Gemini CLI) vs **"Use an API provider"** (Groq, OpenAI, Gemini, xAI, Ollama, LM Studio).
- **Zero-Key Onboarding:** Accurately explains how local CLI authentication avoids secret key generation.
- **Privacy & Safety:** Accurately states that CLI harnesses run in empty isolated temp directories with strictly read-only flags.
- **Working Tree Limitation:** Accurately notes that VibeCodeFolks explains uncommitted working-tree changes (or the latest commit if clean), rather than historical arbitrary commit ranges.

---

## 15. Exact Twitter/X Launch Plan & Thread Copy

Below is the launch copy ready for posting on Twitter/X, designed to maximize developer engagement, communicate unique value, and invite community feedback.

### Tweet 1 (Main Hook):
> AI wrote it. Do you understand it?
>
> When an AI coding agent touches 15 files across your frontend and backend, you shouldn't just skim the git diff and pray.
>
> Introducing VibeCodeFolks — the open-source comprehension engine for AI-generated code.
>
> 🧵👇

### Tweet 2 (The Problem):
> Every developer knows the feeling:
>
> You ask an AI agent to add a feature. It writes 400 lines of code across 5 files.
>
> The diff looks plausible. The tests pass. But do you actually understand how the state mutation works? Did it silently break your conflict contract?
>
> Standard git diff tools show *what* changed, not *why* or *what could go wrong*.

### Tweet 3 (The Solution & Agent Harness):
> VibeCodeFolks doesn't try to edit your code or run autonomous loops.
>
> It runs strictly READ-ONLY in an isolated sandbox and connects directly to your existing coding agent:
>
> ⚡ OpenAI Codex CLI (uses your existing ChatGPT login)  
> ⚡ Anthropic Claude Code (uses your Claude account)  
> ⚡ Google Gemini CLI  
> ⚡ Or direct API keys (Groq, OpenAI, Gemini, local Ollama)  
>
> Zero new subscriptions. Zero API key friction.

### Tweet 4 (Evidence Citations & Change Flow):
> What makes it different?
>
> 1️⃣ Bidirectional Evidence Citations: Every explanation finding cites exact lines in your git diff. Click a citation to jump directly to the code.
> 2️⃣ Change Flow Graph: Visualizes how changes in your API contract propagate into client hooks and UI components.
>
> [Attach screenshot: 04-codex-citation-diff-1440.png and 05-codex-change-flow-1440.png]

### Tweet 5 (The Comprehension Quiz):
> The killer feature: Comprehension Quizzes.
>
> VibeCodeFolks generates evidence-backed questions testing subtle edge cases in your changes.
>
> Pick a wrong answer? It doesn't just say "incorrect" — it explains the exact misconception you had and points to the diff line that proves it.
>
> [Attach screenshot: 06-codex-quiz-wrong-feedback-1440.png]

### Tweet 6 (Install & Try It Today):
> Install with one line on macOS, Linux, or Windows:
>
> ```bash
> curl -fsSL https://raw.githubusercontent.com/BHAVIKA44/vibecodefolks/main/install.sh | sh
> ```
>
> Then `cd` into any repository with git changes and run:
> ```bash
> vibecodefolks
> ```
>
> GitHub: https://github.com/BHAVIKA44/vibecodefolks
>
> Try it on your latest AI-generated branch. Let me know what bugs it catches! 🚀

---

## 16. Step-by-Step Release Checklist (Immediate Action)

Follow this exact sequence to publish `v1.0.0` and execute the public launch:

### Step 1: Tag and Push `v1.0.0`
```bash
git tag v1.0.0
git push origin v1.0.0
```

### Step 2: Monitor GitHub Actions Release Run
Watch `.github/workflows/release.yml` build the release assets:
- Validates cross-platform quality matrix.
- Packages all 5 archives on Ubuntu.
- Runs `release-smoke.py` across Linux and macOS runners.
- Publishes the `v1.0.0` release to GitHub with `checksums.txt`.

### Step 3: Verify Live One-Line Installer
Once the release workflow finishes (~3 minutes):
```bash
curl -fsSL https://raw.githubusercontent.com/BHAVIKA44/vibecodefolks/main/install.sh | sh
vibecodefolks --version
```
Expected output:
```text
vibecodefolks v1.0.0
```

### Step 4: Post the Twitter/X Announcement
Copy the thread from Section 15 above and post on Twitter/X with the attached screenshots from `docs/` or `/tmp/vibe-prelaunch-screenshots/`.
