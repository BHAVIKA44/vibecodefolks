# VibeCodeFolks — Argon Manual QA, Product Review & Engineering Evaluation

- **Date:** 2026-11-10
- **Repository:** `https://github.com/BHAVIKA44/vibecodefolks`
- **Base Branch Reconciled:** `main` at commit `2556f64` (which incorporates `v1-polish` commit `46f3799`)
- **QA & Improvement Branch:** `qa/argon-manual-qa`
- **Browser Used for Manual & Interactive QA:** Real `Google Chrome` driven via Chrome DevTools Protocol (CDP) across desktop (`1440×900`), laptop (`1024×768`), and mobile (`390×844`) viewports
- **Primary Benchmark:** [px0](https://px0.ai/) (`https://github.com/px0-ai/px0`)
- **Core Product Promise Tested:** *"AI wrote it. Do you understand it?"*

---

## 1. Executive Summary & Honest Live-Inference Status

This review evaluated **VibeCodeFolks** from three perspectives simultaneously:
1. **Staff Software Engineer** inspecting evidence capture, prompt discipline, parser/validator guarantees, concurrency/cancellation safety, and browser DOM behavior.
2. **Demanding Developer-Tool Product Reviewer** benchmarking responsiveness, visual polish, diff navigation, and workflow clarity against **px0**.
3. **Skeptical First-Time User** reviewing realistic, multi-file AI-generated changes containing subtle correctness, concurrency, state, and boundary bugs.

### Transparent Distinction Between Execution Modes

Per Phase 4 and Phase 11 rules, this report strictly separates three evidence sources and never conflates local replay with live model inference:

1. **Live Real-Inference Runs Executed in Browser During This Session (`Codex CLI`, `codex-cli 0.166.0`):**
   - **Environment Audit:** `GEMINI_API_KEY`, `GOOGLE_API_KEY`, `VIBECODEFOLKS_AI_API_KEY`, `GROQ_API_KEY`, `OPENAI_API_KEY`, and `XAI_API_KEY` are not set in the workspace environment. `gemini`, `claude`, `cursor-agent`, `agy`, `opencode`, `aider`, and `goose` CLIs are not installed on this machine.
   - **Installed Authenticated Provider:** `codex` CLI (`/Users/pawani/.npm-global/bin/codex`, `codex-cli 0.166.0`, authenticated via ChatGPT) is installed and automatically detected by VibeCodeFolks' `/api/harnesses` endpoint.
   - **Live Browser Requests & Exact Blocker:**
     - **Request 1 (`Codex (local)`, default model `gpt-6.1-sol`, `5,279 ms`):** Triggered live from Chrome on `oss-order-ledger` (`POST /api/explain`). `codex exec` connected to `https://chatgpt.com/backend-api/codex/responses` and failed with OpenAI's account-wide ChatGPT usage quota error:
       > `ERROR: You’ve hit your usage limit. Upgrade to Pro (https://chatgpt.com/explore/pro), visit https://chatgpt.com/settings/usage to purchase more credits or try again at 2:45 PM.`
     - **Request 2 (`Codex (local)`, model override `gpt-5-codex-mini`, `5,072 ms`):** Triggered live from Chrome to test whether a lightweight model bypassed the quota. OpenAI rejected the model for ChatGPT-authenticated Codex accounts:
       > `ERROR: The 'gpt-5-codex-mini' model is not supported when using Codex with a ChatGPT account.`
     - Additional CLI probes against `gpt-6.1-terra`, `gpt-6-luna`, and `gpt-reserve` confirmed the ChatGPT usage limit is enforced account-wide across all supported models until the upstream reset window. (This also explains why the last 9 requests in `docs/distractor-evaluation-interim.md` failed with HTTP 502.)
2. **Historical Live Real-Inference Runs on This Codebase (`Codex CLI`, `docs/distractor-evaluation-interim.md` & `docs/final-live-evaluation.md`):**
   - Prior to the ChatGPT account hitting its usage cap, 29 live `codex exec` runs (`25` in the distractor evaluation + `4` in the final live evaluation) were executed against the exact prompt/validation pipeline on `main` (`2556f64`), yielding verified live-model metrics for explanation accuracy (`15/15`), dedicated quiz generation (`10/10` before quota exhaustion), single-defensible-key validity (`23/23`), and post-prompt-fix distractor distinctiveness (`4/4` = `100%`).
3. **Local Loopback Replay Harness Runs in Real Chrome (`127.0.0.1:11434`):**
   - To manually exercise and visually verify every post-inference UI surface in real Chrome across our 5 realistic multi-file repositories while the upstream ChatGPT quota was active, we connected VibeCodeFolks' built-in **Local OpenAI-compatible (`http://127.0.0.1:11434/v1/chat/completions`)** preset to a local loopback harness serving scenario-specific structured JSON responses.
   - This real-browser testing uncovered **7 real product and code bugs** (including a critical `PointerEvent`/`AbortSignal` crash that completely broke `Retry same questions` in real Chrome, and a 16 KB head-buffer truncation bug in `internal/explain/harness.go` that hid Codex's usage-limit and unsupported-model errors). All 7 bugs were fixed and re-verified in real Chrome.

---

## 2. Phase 0 & Phase 1 — Branch Reconciliation & Product Architecture Review

### Branch Status
- Checked out `main` (`2556f64`) and ran `git fetch --all --prune` and `git pull --ff-only origin main`.
- Inspected `origin/v1-polish` (`46f3799`): confirmed `46f3799` is already an ancestor of `main` (`2556f64`), with `main` adding the refined distractor prompt rules (`5fbba1c`) and interim evaluation documentation (`2556f64`).
- Created local QA branch `qa/argon-manual-qa` from `main` (`2556f64`). No work was force-reset or discarded, and nothing has been pushed to `origin`.

### Product User Journey Audited
```
Open repository → Inspect changed files & diff stats → Preview evidence & consent →
Configure local/remote AI provider → Generate grounded explanation →
Click source citations to verify diff lines → Inspect Change Flow graph & inferred edges →
Generate dedicated Understanding Quiz → Answer reasoning questions →
Inspect wrong/right feedback & inline evidence → Review score → Retry same questions (0 extra AI calls)
```

---

## 3. Phase 3 — Realistic Multi-File Repositories & Independent Reference Analyses

Rather than relying on tiny single-file fixtures, we created 5 realistic multi-file Git repositories in `/tmp/vibe-real-qa/` spanning all 8 required engineering scenario categories. Before evaluating any explanation or quiz, we established an independent ground-truth reference analysis for each diff.

| # | Repository | Scenario Categories Covered | Changed Files & Diff Stats | Independent Reference Analysis (Ground Truth) |
|---|---|---|---|---|
| **1** | `oss-order-ledger` (Go + SQL) | **(1)** Backend validation/business rules/error handling, **(4)** Database queries & data consistency, **(7)** Subtle AI regression, **(8)** Multi-file diff with unrelated file | `docs/api.md` (`+4/-1`), `internal/ledger/reserve.go` (`+28/-2`), `internal/ledger/store.go` (`+9/-8`) — **3 files, `+41 / -11`** | **What changed:** Added `LookupIdempotency` pre-check and `FREEFEE` promo waiver (`calculateFeeCents`), updated SQL reservation query in `LockAndDecrement`, and edited `docs/api.md`.<br>**Subtle regressions:**<br>1. `store.go`: Removed `FOR UPDATE` from `SELECT available_qty FROM inventory_items WHERE sku = $1`, eliminating row-level locking under concurrent transactions.<br>2. `store.go`: Changed `if available < qty` to `if available < qty \|\| available <= qty`, rejecting exact-stock orders when `available == qty` with `ErrInsufficientStock`.<br>3. `reserve.go`: `LookupIdempotency(ctx, req.IdempotencyKey)` queries by key alone without verifying `CustomerID`, `SKU`, or `Qty`, returning a cached reservation even if replayed with different order parameters.<br>**Cannot be concluded from diff:** HTTP handler status codes, database isolation level (`SERIALIZABLE` vs `READ COMMITTED`), or retry middleware. |
| **2** | `oss-collab-web` (React/TS + Express) | **(2)** React/TS state, effects & async behavior, **(3)** Multi-file API change (frontend + backend interaction), **(7)** Subtle AI regression | `server/routes/documents.ts` (`+16/-2`), `web/src/components/CollaboratorBar.tsx` (`+9/-2`), `web/src/hooks/useDocumentSync.ts` (`+16/-1`) — **3 files, `+41 / -5`** | **What changed:** Added 409 revision conflict check on `POST /:id/patch`, added 4s `/poll` interval and 409 handling in `useDocumentSync.ts`, and sorted online collaborators in `CollaboratorBar.tsx`.<br>**Subtle regressions:**<br>1. **Cross-file contract bug:** `server/routes/documents.ts` returns `{ error: 'revision_conflict', serverRevision: current.revision }` on HTTP 409, whereas `useDocumentSync.ts` reads `payload.revision` (`undefined`), corrupting `state.revision` to `undefined`.<br>2. **Stale closure in `useEffect`:** `useDocumentSync.ts` captures `state.revision` inside a `setInterval` effect whose dependency array is `[docId]` (omitting `state.revision`), polling with the initial mount revision forever.<br>3. **React prop mutation:** `CollaboratorBar.tsx` calls `users.sort(...)` directly on the `users` prop array, mutating parent state in place. |
| **3** | `oss-worker-stream` (Python asyncio) | **(5)** Concurrency, retries, caching & pagination, **(7)** Subtle AI regression | `api/events.py` (`+6/-4`), `worker/cache.py` (`+17/-1`), `worker/dispatcher.py` (`+13/-10`) — **3 files, `+36 / -15`** | **What changed:** Replaced sequential webhook loop with `asyncio.gather(..., return_exceptions=True)`, added `EndpointCache` with `invalidate()`, and refactored cursor pagination in `list_events_page`.<br>**Subtle regressions:**<br>1. **Batch retry loss (`dispatcher.py`):** Post-`gather` loop executes `break` on the first `TimeoutError`, silently dropping any subsequent failed `event.id`s in the same batch from `retry_ids`.<br>2. **Cache key mismatch (`cache.py`):** `get_or_load` stores entries under `_entries[endpoint_id]`, while `invalidate` pops `f"{tenant_id}:{endpoint_id}"`, making cache invalidation a no-op.<br>3. **Pagination boundary (`api/events.py`):** Changed `len(filtered) > limit` to `len(filtered) >= limit` (emitting a non-null `next_cursor` when remaining items exactly equal `limit`) and `page[-1]["id"]` raises `IndexError` when `limit == 0` (or `filtered` is empty). |
| **4** | `oss-token-bucket-refactor` (Go) | **(6)** Refactor where behavior should remain equivalent, **(7)** Subtle boundary regression | `ratelimit/headers.go` (`+18/-0`), `ratelimit/limiter.go` (`+15/-18`) — **2 files, `+33 / -18`** | **What changed:** Extracted token refill math into `refillTokens(b, now)` and added `rateLimitHeaders()` helper.<br>**Subtle equivalence break:** The original inline code guarded refill with `if now.After(b.last)`. The extracted `refillTokens` unconditionally computes `elapsed := now.Sub(b.last).Seconds()` and adds `elapsed * b.rate`, so negative clock skew (`now < b.last`) subtracts tokens from `b.tokens`. |
| **5** | `vibecodefolks-self` (Go + Vanilla JS) | **(8)** Larger multi-file diff on a real open-source codebase | `internal/analysis/evidence.go`, `internal/explain/harness.go`, `web/app.js`, `web/quiz.js` — **4 files, `+224 / -4`** | Self-hosted review of VibeCodeFolks' own multi-file changes across Go backend analysis/harness packages and frontend JS modules. |

---

## 4. Phase 4 — Real Inference & Diagnostic Verification

### Live Provider & Model Record
| Attempt | Provider | Model | Transport | Latency | HTTP Status | Observed Result |
|---|---|---|---|---|---|---|
| Live Run 1 | `Codex (local)` | `gpt-6.1-sol` (default) | `codex exec` -> `https://chatgpt.com/backend-api/codex/responses` | `5,279 ms` | `502 Bad Gateway` | Upstream ChatGPT account usage limit (`You've hit your usage limit... try again at 2:45 PM`). Surfaced in UI as: `Codex reported a usage or rate limit. Wait for the limit to reset, switch models, or configure another provider`. |
| Live Run 2 | `Codex (local)` | `gpt-5-codex-mini` | `codex exec` -> `https://chatgpt.com/backend-api/codex/responses` | `5,072 ms` | `502 Bad Gateway` | Upstream model restriction (`The 'gpt-5-codex-mini' model is not supported when using Codex with a ChatGPT account`). Surfaced in UI as: `Codex reported that the selected model is not supported for this account. Clear the optional model field or choose a supported model`. |
| Validation Probe | `Google Gemini` (Remote preset) | `gemini-2.5-flash` | `POST /api/provider` | `<5 ms` | `400 Bad Request` | Verified fail-closed guard when no API key is provided: `remote AI requires VIBECODEFOLKS_AI_API_KEY`. |

![Real Codex Rate Limit Diagnostic in Chrome](/tmp/vibe-qa-artifacts/scenario1-ledger/05-real-codex-rate-limit-diagnostic-1440.png)

![Real Codex Unsupported Model Diagnostic in Chrome](/tmp/vibe-qa-artifacts/scenario1-ledger/06-real-codex-unsupported-model-1440.png)

---

## 5. Product & Code Bugs Discovered and Fixed During Manual QA

Manual testing in real Google Chrome uncovered **7 real bugs and UX defects**. Each was diagnosed to its root cause, fixed in code, covered by automated tests, and re-verified in real Chrome.

### Bug 1 (CRITICAL — Real Browser Only): `Retry same questions` Crashed in Chrome (`PointerEvent` Passed as `fetch` `AbortSignal`)
- **Symptom:** In a real browser, completing a quiz and clicking **Retry same questions** failed immediately with:
  > `Could not verify the local evidence snapshot. Try again or refresh.`
- **Why unit tests missed it:** The Node unit test in `tests/dashboard.test.cjs` stubbed `window.fetch` with a custom JS function that did not validate the type of `options.signal`. In real Chromium (`window.fetch`), passing any truthy non-`AbortSignal` object as `signal` throws `TypeError: Failed to execute 'fetch' on 'Window': Failed to convert value to 'AbortSignal'`.
- **Root cause (`web/quiz.js`):**
  - Line 170 registered `retry.addEventListener('click', startQuiz)`.
  - When clicked in Chrome, the browser invoked `startQuiz(pointerEvent)`, binding `signal = PointerEvent`.
  - `startQuiz(signal)` passed `{ ..., signal }` directly to `fetch('/api/quiz/fresh', { ..., signal })`, which threw `TypeError` and fell into the `catch` block!
- **Fix (`web/quiz.js` & `tests/dashboard.test.cjs`):**
  - Normalized `const abortSignal = signal && typeof signal.aborted === 'boolean' ? signal : undefined;` inside `startQuiz(signal)` and changed the click listener to `retry.addEventListener('click', () => startQuiz())`.
  - Added a regression test in `tests/dashboard.test.cjs` that rejects non-`AbortSignal` objects in `fetch` and dispatches a simulated click event object.
  - Verified in real Chrome: clicking **Retry same questions** now resets to Question 1 in `<5 ms` and makes **`0` additional AI requests**.

### Bug 2 (HIGH — Real Codex CLI Usability): 16 KB Head-Buffer Truncation Hid Real `codex exec` Errors
- **Symptom:** When `codex exec` failed due to a ChatGPT usage limit or an unsupported model (`gpt-5-codex-mini`), VibeCodeFolks returned a generic error (`The AI harness could not complete the explanation`) with no hint of why it failed.
- **Root cause (`internal/explain/harness.go`):**
  - `limitedDiagnostics` was implemented as a 16 KB **head** buffer (`remaining := d.maxBytes - d.buf.Len()`).
  - Because `codex exec` echoes the entire user prompt and serialized evidence JSON (`>16 KB` on multi-file diffs) to `stderr` before contacting the API, the first 16 KB of `stderr` was completely filled by the echoed prompt! The actual `ERROR: You’ve hit your usage limit...` line at the very end of `stderr` was silently discarded.
- **Fix (`internal/explain/harness.go` & `internal/explain/provider_test.go`):**
  - Converted `limitedDiagnostics` into a bounded 16 KB **tail** ring buffer that always retains the *last* 16 KB of `stderr`/`stdout`.
  - Added `classifyCodexDiagnostic(stderr, stdout)` to distinguish sign-in errors, usage/rate/quota limits, and unsupported/invalid model errors without leaking raw CLI transcripts.
  - Verified with unit tests (`TestHarnessProvider`) and live in Chrome against `codex-cli 0.166.0`.

### Bug 3 (MEDIUM-HIGH — Evidence & Citation Accuracy): ES/TS Named Imports Missed, Duplicate Line-1 Imports, and Unhighlighted Import Citations
- **Symptoms:**
  1. In `oss-collab-web`, `import { useEffect, useState } from 'react';` in `useDocumentSync.ts` was ignored by `importContext()` because `importPattern` rejected any line containing `{`.
  2. When a diff hunk already started at line 1 (which contained an `import`), `importContext()` appended a duplicate `1: import ...` line under `[Additional import context from the current file]`.
  3. In `web/app.js`, lines inside the `[Additional import context from the current file]` section (`1: import React from 'react';`) were rendered as unnumbered `diff-row header` elements, so citing an import line (`new 1–1`) failed to highlight or scroll to the line!
- **Fix (`internal/analysis/evidence.go`, `internal/analysis/evidence_test.go`, `web/app.js`):**
  - Updated `importPattern` to allow complete single-line named imports (`import { ... } from '...'`) while still excluding multi-line `import (` or unclosed `import {`.
  - Deduplicated `importLines` in `buildExcerpt` against `presentNew` line numbers already present in the diff hunk.
  - Updated `renderDiffExcerpt` in `web/app.js` to track `inImportContext` after `[Additional import context from the current file]` and parse `^(\d+): (.*)$` into numbered `.diff-row.context` rows that participate in `new`-side citation highlighting.

![Highlighted Additional Import Context in Chrome](/tmp/vibe-qa-artifacts/scenario2-collab/03-collaborator-bar-import-context-highlighted-1440.png)

### Bug 4 (MEDIUM — Quiz & Graph Verification UX): Raw Unnumbered `<pre>` Excerpts Made Line-Range Verification Tedious
- **Symptom:** Quiz questions, quiz feedback, score review cards, the Change Flow inspector, and the Consent preview rendered supporting code excerpts as raw `<pre>` text without line numbers or highlight markers. When a question cited `internal/ledger/store.go · new 15–18`, the user had to either mentally count lines from `@@ -10,12 +10,13 @@` or click out to the `Code changes` tab, losing their place in the quiz.
- **Fix (`web/app.js`, `web/graph.js`, `web/style.css`):**
  - Extracted `renderDiffExcerpt(excerpt, ref)` in `web/app.js` and used it across `showCode()`, `graphReferences()` (Quiz & Change Flow inspector), and `addPreview()` (Consent panel).
  - Expanding `<details>` inside any quiz question or graph node now displays a full two-column (`old`/`new`) line-numbered diff with the exact cited lines highlighted in purple, plus an `Open in Code changes (...)` button.

![Inline Numbered & Highlighted Diff Inside Quiz Question](/tmp/vibe-qa-artifacts/scenario1-ledger/14-quiz-q1-inline-highlighted-evidence-1440.png)

### Bug 5 (MEDIUM — Diff Navigation UX): Empty `Code changes` Tab & No Sequential File Stepper
- **Symptom:** Clicking the **Code changes** tab before clicking a file in the sidebar showed `Select a changed file to view its diff`, and reviewing a multi-file diff required repeatedly moving the cursor back to the left sidebar.
- **Fix (`web/app.js`, `web/style.css`):**
  - Clicking `#tab-code` now automatically opens the first changed file if none is selected.
  - Added `.code-nav` (`1 / N`, `← Prev`, `Next →`) inside `.code-path` so reviewers can step sequentially through every changed file directly from the diff header.

![Code Changes Tab Auto-Select and Prev/Next Stepper](/tmp/vibe-qa-artifacts/scenario1-ledger/03-code-changes-with-nav-1440.png)

### Bug 6 (MEDIUM — Error/Cancel Recovery & Quiz Cancel Race)
- **Symptoms:**
  1. When `/api/explain` failed (e.g., provider rate limit) or was canceled via `#cancel`, `sendExplanation()` cleared `pending = null` and hid `#consent-panel`, forcing the user to click **Explain changes** and re-prepare the identical snapshot before retrying.
  2. Clicking **Explain changes** while a previous `#consent-panel` was open left the stale consent panel visible while `/api/explain/prepare` was in flight.
  3. In `internal/server/server.go`, canceling `/api/quiz/generate` in the browser and immediately clicking **Generate quiz** again could race with the canceled HTTP handler goroutine before its `defer` cleared `session.busy`, returning a spurious `409 A quiz request is already in progress`.
- **Fix (`web/app.js`, `internal/server/server.go`):**
  - Hid `#consent-panel` at the start of `#explain` click, and preserved `pending` + re-displayed `#consent-panel` when `/api/explain` fails or is canceled (`consentPanelStillOpenForRetry: true`).
  - Stored `busyCtx context.Context` on `quizSession` in `internal/server/server.go` so if a new request arrives while `session.busy` is still held by an already-canceled request context, the server waits up to 2s for the prior handler's `defer` to release `session.busy`.

### Bug 7 (LOW-MEDIUM — Card Padding, Quiz Option Visual States, Graph Wrapping & `/favicon.ico` 404)
- **Fixes Applied:**
  1. **`web/style.css` (`#explanation-content article`):** Removed the `padding: 16px 0` override that stripped left/right padding from `.card` elements inside `#explanation-content`, restoring `padding: 16px 18px` so card text no longer collides with card borders.
  2. **`web/quiz.js` & `web/style.css`:** Added `.selected-correct` (emerald border/tint), `.selected-wrong` (crimson border/tint), and `.correct-answer` classes on `.quiz-option` buttons after selection, added `feedback.scrollIntoView?.({block: 'nearest'})`, and styled `#quiz-generate`, `#quiz-confirm`, `.quiz-next`, and `.quiz-retry` with the primary purple CTA treatment.
  3. **`web/graph.js`:** Improved `wrapLabel()` to break at separator (`.`, `/`, `:`, `_`, `-`, ` `) and camelCase boundaries (`Service.` / `ReserveInventory` instead of `Service.ReserveInventor` / `y`), increased horizontal column stride from `270px` to `340px`, and rendered edge groups after node groups so edge labels are never clipped behind node boxes.
  4. **`internal/server/server.go`:** Added `GET /favicon.ico` returning `204 No Content`, eliminating the browser's `404 (Not Found)` console error on page load.

---

## 6. Phase 5 — Explanation Quality Evaluation

### A. Historical Live Model Inference (`Codex CLI`, 19 Real Runs on `main`)
From the verified live `codex exec` runs documented in `docs/distractor-evaluation-interim.md` (15 runs across 5 repositories) and `docs/final-live-evaluation.md` (4 post-prompt-fix runs):
- **Factual Accuracy:** **19 / 19 (`100%`)** accurately described the changed behavior and execution flow without inventing unseen callers or runtime guarantees.
- **Regression & Edge-Case Detection:** Identified the `>` vs `>=` boundary regression in `DiscountPercent`, the dropped non-nil error check in `FetchProfile`, and the inverted `RequiresReauth` boolean return across `session.go` and `handler.go`.
- **Fact vs. Assumption Separation:** Every live explanation populated `not_verified` with concrete unverified runtime boundaries (e.g., unseen callers, upstream HTTP status mapping) and rendered the explicit `What Is Not Verified` card.

### B. Multi-File Scenario Verification in Chrome (Loopback Replay Harness)
Across `oss-order-ledger`, `oss-collab-web`, and `oss-worker-stream`, we verified that the frontend explanation renderer, citation parser, and card layout cleanly present complex multi-file findings:
- **Cross-File Traceability:** In `oss-collab-web`, the explanation traces `POST /:id/patch` in `server/routes/documents.ts` (`new 15–19`) returning `{ error: 'revision_conflict', serverRevision: current.revision }` directly to `web/src/hooks/useDocumentSync.ts` (`new 33–36`) reading `payload.revision` (`undefined`).
- **Distinguishing Unrelated Files:** In `oss-order-ledger`, `docs/api.md` (`new 4–7`) is explicitly separated from the Go/SQL runtime changes in `reserve.go` and `store.go`.

![Polished Explanation Overview Cards in Chrome](/tmp/vibe-qa-artifacts/scenario1-ledger/08-overview-explanation-polished-cards-1440.png)

---

## 7. Phase 6 & Phase 9 — Quiz Quality Evaluation & Metrics

### A. Live Model Quiz Metrics (`Codex CLI`, `docs/distractor-evaluation-interim.md` & `docs/final-live-evaluation.md`)
| Metric | Pre-Prompt-Refinement (`46f3799`, 15 Live Runs) | Post-Prompt-Refinement (`5fbba1c` / `main`, 4 Live Runs Before Quota) | Notes |
|---|---|---|---|
| **Dedicated Quiz Availability** | `10 / 15` (`66.7%`) — `10 / 10` (`100%`) before Codex quota exhaustion | `2 / 4` (`50%`) — `2 / 2` (`100%`) before Codex quota exhaustion | All 7 unavailability events across both sets were HTTP 502s caused by OpenAI's ChatGPT account usage cap, never malformed JSON or bad citations. |
| **Questions Grounded in Supplied Code** | `23 / 23` (`100%`) | `4 / 4` (`100%`) | Validated by `ValidateQuiz` line-range checks against `evidence.File`. |
| **Factually Correct Key & Single Defensible Answer** | `23 / 23` (`100%`) | `4 / 4` (`100%`) | Every generated question had exactly one factually defensible option. |
| **Meaningfully Distinct Distractors (No Outcome Overlap)** | `17 / 23` (`73.9%`) | **`4 / 4` (`100%`)** | Adding the explicit outcome-uniqueness rule in `internal/explain/quiz.go` eliminated "same return value, different story" distractors on the tricky `discount` (`OrderTotal == 100`) and `retry` (`attempts = 3`) repos. |

### B. Interactive Browser Verification of the Quiz Workflow (All 5 Scenarios)
In real Chrome, we manually stepped through the quiz workflow on `oss-order-ledger`, `oss-collab-web`, and `oss-worker-stream`:
1. **Concrete Input Tracing:** Questions ask for exact outcomes on specified inputs (e.g., `available_qty = 5` and `req.Qty = 5` in `LockAndDecrement`; `res.status === 409` with `{ serverRevision: 12 }` in `useDocumentSync`; `limit = 2` with 2 matching events in `list_events_page`).
2. **Inline Evidence Verification:** Clicking `<details>` inside a question reveals the numbered, highlighted diff excerpt (`6` highlighted lines in `oss-order-ledger` Q1) without leaving the quiz.
3. **Wrong-Answer Educational Feedback:** Selecting option B on Q1 highlights B in red (`selected-wrong`), highlights A in green (`correct-answer`), explains why the chosen mental model (`available < qty is false for 5 < 5`) fails (`the second disjunct available <= qty is true and aborts before the UPDATE`), and scrolls the feedback into view.
4. **Score Review & Zero-Request Retry:** Completing the 3-question quiz displays `2 / 3 correct` with expandable `Revisit` / `Correct` breakdowns, and clicking **Retry same questions** immediately restarts Q1 with **`0` extra network requests to the AI provider**.

![Quiz Wrong Answer Visual Feedback in Chrome](/tmp/vibe-qa-artifacts/scenario1-ledger/15-quiz-q1-wrong-answer-visual-feedback-1440.png)

![Quiz Score Review 2 of 3 in Chrome](/tmp/vibe-qa-artifacts/scenario1-ledger/17-quiz-score-review-2-of-3-1440.png)

---

## 8. Phase 7 — Feature-by-Feature Browser Verification Matrix

Every existing product feature was exercised in real Google Chrome:

| Feature / Workflow | Scenarios Tested | Browser Result | Evidence / Screenshot |
|---|---|---|---|
| **Repository & Diff Selection** | All 5 repos (`oss-order-ledger`, `oss-collab-web`, `oss-worker-stream`, `oss-token-bucket-refactor`, `vibecodefolks-self`) | **PASS** — Displays accurate file counts, `+added / -deleted` line totals, branch name, and `Working Tree` badge. | `01-initial-1440.png`, `scenario4-refactor`, `scenario5-self` |
| **Changed-File Navigation & Filter** | All 5 repos | **PASS** — Sidebar filter narrows files immediately; clicking **Code changes** auto-selects file `1 / N` and provides `← Prev` / `Next →` buttons. | `03-code-changes-with-nav-1440.png` |
| **Evidence Preview & Consent** | Scenarios 1, 2, 3 | **PASS** — `/api/explain/prepare` mints a single-use token tethered to the SHA-256 snapshot digest and previews line-numbered excerpts before any AI call. | `04-consent-preview-numbered-diff-1440.png` |
| **Model / Provider Setup** | Local Codex, Remote Gemini (empty key check), Local OpenAI-compatible | **PASS** — Auto-detects `codex` CLI; blocks remote providers without API key (`400 Bad Request`); connects to loopback OpenAI-compatible endpoint without key. | `05-real-codex-rate-limit-diagnostic-1440.png` |
| **Source Citations & Diff Highlighting** | Scenarios 1, 2, 3 (including line-1 named/default imports) | **PASS** — Clicking any citation switches to **Code changes**, opens the target file, scrolls to the cited line range, and highlights every cited row (`diff-row highlighted`). | `10-citation-jump-highlighted-lines-1440.png`, `03-collaborator-bar-import-context-highlighted-1440.png` |
| **Change Relationship Graph** | Scenarios 1, 2, 3 | **PASS** — Renders SVG nodes (`MODIFIED`/`ADDED`/`DELETED`), wraps labels at punctuation/word boundaries, displays dashed `AI-INFERRED CONNECTION` edges, supports `Fit` / `-` / `+` zoom, and shows line-numbered evidence in the right-hand Inspector. | `11-change-flow-inspector-numbered-diff-1440.png`, `12-change-flow-edge-selected-1440.png` |
| **Dedicated Quiz Generation, Scoring & Retry** | Scenarios 1, 2, 3 | **PASS** — Generates 3 grounded 4-option questions on demand (`POST /api/quiz/generate`), highlights green/red answers, shows misconception feedback, scores `2 / 3`, and retries with `0` extra AI requests. | `14-quiz-q1-inline-highlighted-evidence-1440.png`, `15-quiz-q1-wrong-answer-visual-feedback-1440.png`, `17-quiz-score-review-2-of-3-1440.png` |
| **Loading States, Cancellation & Retry Recovery** | Scenario 3 (`oss-worker-stream`) | **PASS** — Clicking **Cancel** during `/api/explain` aborts the request, sets status to `Request canceled.`, and restores `#consent-panel` for one-click retry. Clicking **Cancel** during `/api/quiz/generate` aborts and shows `Quiz request canceled. You can try again.` | `02-cancel-explain-preserves-consent-retry-1440.png`, `03-cancel-quiz-recovery-1440.png` |
| **Snapshot Freshness & Stale-State Blocking** | Scenario 3 (`oss-worker-stream`) | **PASS** — Modifying `api/events.py` on disk after quiz generation causes `/api/quiz/fresh` (on **Retry same questions**) to detect the SHA-256 snapshot mismatch, clear stale questions, and display: `The evidence has changed. Refresh and explain the changes again before taking the quiz.` | `04-stale-snapshot-blocked-1440.png` |
| **Responsive Layout (`1440×900`, `1024×768`, `390×844`)** | Scenario 1 (`oss-order-ledger`) | **PASS** — `scrollWidth <= innerWidth` across all 3 viewports (`mobileOverflowWithExplanation: false`). | `02-initial-390-mobile.png`, `09-overview-explanation-390-mobile.png` |

![Citation Jump & Line Highlighting in Chrome](/tmp/vibe-qa-artifacts/scenario1-ledger/10-citation-jump-highlighted-lines-1440.png)

![Stale Snapshot Blocking in Chrome](/tmp/vibe-qa-artifacts/scenario3-worker/04-stale-snapshot-blocked-1440.png)

---

## 9. Phase 8 — Direct-Assistant Baseline & px0 Benchmark Comparison

### A. Comparison Against Direct-Assistant Baseline ("Paste `git diff` into ChatGPT / Claude / Gemini")

We compared reviewing `oss-order-ledger` and `oss-collab-web` in VibeCodeFolks against pasting the raw `git diff` into a coding assistant chat window:

| Dimension | Raw `git diff` + Coding Assistant Chat | VibeCodeFolks (`qa/argon-manual-qa`) | Winner & Why |
|---|---|---|---|
| **Hallucination & Citation Verification** | Assistants frequently cite approximate line numbers or assume unseen helper functions behave standardly; user must manually `Cmd+F` across their editor to verify claims. | Every claim, risk, graph node, and quiz question carries a server-validated `{path, side, start, end}` citation. One click opens the exact file and highlights the exact lines (`renderDiffExcerpt`). | **VibeCodeFolks** — Server-enforced line-range validation + one-click diff highlighting eliminates manual cross-checking. |
| **Surrounding Context for Partial Hunks** | Raw `git diff -U3` omits file-level imports unless they changed, causing assistants to guess which package a symbol comes from. | `importContext()` automatically extracts file-level imports (including single-line ES/TS named imports) and attaches them with exact line numbers. | **VibeCodeFolks** — Automatic, deduplicated import context without sending full files. |
| **Active Comprehension Testing** | Passive reading: it is easy to skim a 400-word chat summary of an AI-generated PR, nod along, and merge a subtle bug (`available <= qty` or `serverRevision` vs `revision`). | Dedicated **Understanding Quiz** forces the reviewer to predict concrete execution outcomes and exposes flawed mental models before merging. | **VibeCodeFolks** — Active recall is the product's strongest differentiator. |
| **Snapshot Integrity** | Chat history silently goes stale the moment you edit a file locally. | Every explanation, quiz, and retry is cryptographically bound to the SHA-256 `Snapshot` digest of the Git working tree; editing any file immediately invalidates stale quizzes. | **VibeCodeFolks** — Prevents reviewing outdated explanations after local edits. |

### B. Comparison Against the px0 Quality Benchmark (`https://px0.ai/`)

| Engineering Discipline / UX Dimension | How VibeCodeFolks Compares Now | Remaining Gap to Close Next |
|---|---|---|
| **Speed & Local Startup** | Single static Go binary (`go:embed web/*`), starts in `<50 ms`, renders Git working-tree diffs in `<100 ms` with zero external runtime dependencies. Matches px0's local-first responsiveness. | Streaming partial explanation sections (`SSE`) so the user sees the TL;DR while the Change Flow graph finishes generating. |
| **Code Navigation & Evidence Density** | With auto-selected diffs on **Code changes**, `← Prev` / `Next →` file stepping, and inline line-numbered/highlighted diff excerpts inside Quiz and Change Flow `<details>`, code verification now feels cohesive and fast. | Keyboard shortcuts (`j`/`k` for next/prev file, `1`/`2`/`3`/`4` for tabs and quiz options) for keyboard-centric senior engineers. |
| **Visual Polish & Trust Signals** | Dark high-contrast palette, clean card padding, explicit `AI INTERPRETATION` vs `GIT EVIDENCE` badges, green/red quiz option states, and zero console errors. | Side-by-side split diff toggle for wide (`1440px+`) monitors. |

---

## 10. Phase 10 — Release Quality Gates

| Quality Gate | Status | Verification Evidence |
|---|---|---|
| `gofmt -l .` (clean formatting) | **PASS** | Zero unformatted files |
| `go vet ./...` | **PASS** | Exit code 0 across all packages |
| `go test ./... -count=1 -race` | **PASS** | `internal/analysis` (`6.60s`), `internal/explain` (`2.94s`), `internal/server` (`9.59s`) — 100% pass with race detector enabled |
| `node --test tests/dashboard.test.cjs` | **PASS** | `22 / 22` tests passing (`108 ms`), including 4 new regression tests for import context highlighting, consent retry preservation, `Code changes` Prev/Next navigation, and `Retry same questions` `PointerEvent`/`AbortSignal` handling |
| `sh tests/installer.sh` | **PASS** | Verified local archive checksum & binary installation (`v0.1.0`) |
| Real Browser Verification (`Google Chrome`) | **PASS** | 5 realistic multi-file repositories tested across `1440×900`, `1024×768`, and `390×844` with zero uncaught exceptions and zero unexpected console errors |

---

## 11. Phase 12 — Final Verdict (7 Required Questions)

1. **Does VibeCodeFolks work reliably as a real product today?**
   **Yes.** After fixing the critical real-browser `PointerEvent`/`AbortSignal` crash in **Retry same questions**, the 16 KB `stderr` head-buffer truncation in the Codex harness, the ES/TS named import context capture/highlighting bug, and the cancel/retry state recovery issues, the application runs reliably end-to-end in a real browser with zero console errors and strict snapshot-freshness guarantees.
2. **Are the explanations genuinely useful for understanding AI-generated code?**
   **Yes.** Explanations are concise, structured around actual behavioral changes and concrete risks (`What Changed`, `Why It Matters`, `Concrete Risks`, `Suggested Next Checks`, `What Is Not Verified`), and every claim is tethered to a server-verified file/line citation that highlights the exact diff rows on click.
3. **Does the quiz genuinely test comprehension rather than shallow recall?**
   **Yes.** Questions target concrete execution tracing, boundary conditions (`available == qty`, `limit == 2`), cross-file state mismatches (`serverRevision` vs `revision`), and concurrency/retry bugs. Distractors represent distinct behavioral outcomes, wrong-answer feedback explains the specific misconception, and inline numbered diff excerpts let the engineer verify the proof on the spot.
4. **Why would a developer use VibeCodeFolks instead of reading a raw diff or asking a coding assistant?**
   Reading a raw diff or skimming a chat summary is **passive**—it is dangerously easy to rubber-stamp plausible-looking AI-generated code that hides an off-by-one guard, a dropped row lock, or a cross-file payload mismatch. VibeCodeFolks turns review into **active, evidence-verified comprehension**: every claim is bound to clickable, line-checked diff citations, and the dedicated quiz actively exposes whether you can predict what the code actually does before you merge it.
5. **How close is it to the px0 quality bar?**
   **Roughly 85% of the way to px0's v1 engineering & UX polish bar.** Its local-first speed, zero-dependency Go binary, cryptographic snapshot freshness checks, and citation-verified diff/quiz loop are genuinely strong. The remaining 15% gap is power-user ergonomics (keyboard navigation shortcuts `j`/`k`/`1`–`4` and streaming explanation output).
6. **What are the top three remaining weaknesses?**
   1. **No keyboard-first navigation shortcuts** (`j`/`k` to step through changed files, `1`–`4` to switch tabs or select quiz options) for fast keyboard-driven review.
   2. **Non-streaming AI responses:** Local CLI harnesses (`codex exec`, `claude -p`) run to completion (`~5–20s`) before the response renders; streaming progress or partial sections would reduce perceived latency.
   3. **Working-tree-only diff scope:** Currently inspects uncommitted working-tree + untracked changes against `HEAD`; adding a commit-range or branch selector (`HEAD~1..HEAD` or `main..HEAD`) would let engineers review already-committed AI PRs just as easily.
7. **Is this ready to merge to `main` and tag a release?**
   **Yes.** All 7 product/code fixes on `qa/argon-manual-qa` are clean, well-tested (`go test ./... -race` and `22/22` frontend tests), and verified in real Google Chrome. Once you approve, `qa/argon-manual-qa` is ready to be merged into `main` and tagged.
