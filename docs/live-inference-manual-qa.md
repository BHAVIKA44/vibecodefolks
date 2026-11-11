# Live AI Inference Manual QA Report

This report documents the live end-to-end browser evaluations conducted on VibeCodeFolks using real AI model inference via Groq and local harnesses.

## 1. Test Environment & Methodology

- **Repository**: `https://github.com/BHAVIKA44/vibecodefolks`
- **Branch**: `qa/argon-manual-qa`
- **Frontend Viewports**: Desktop (1440x900) and Mobile (390x844)
- **Browser Execution**: Automated CDP runner driving real Google Chrome sessions against local `vibecodefolks` servers.
- **Rules**: Zero mocked responses for live evaluation runs. Real requests made through the running UI to the Groq API endpoint.

---

## 2. Test Scenarios Evaluated

### Scenario 1: `oss-order-ledger` (Go Backend)
- **Context**: Added warehouse-aware locking and idempotency caching to inventory reservation service.
- **Model**: `openai/gpt-oss-120b`
- **Explanation**: 5,675ms response time; 18 valid source citations; identified idempotency cache and warehouse routing changes.
- **Quiz**: 3,052ms; 2/2 valid multiple-choice questions; verified step-by-step logic tracing; scored 1/2 in simulated interactive test.

### Scenario 2: `oss-collab-web` (React / TypeScript Frontend & API)
- **Context**: Optimistic conflict resolution in document editor, presence polling hook, and collaborator bar filtering.
- **Model**: `qwen/qwen3.8-27b` (Chrome Live Verified)
- **Explanation**: 4,237ms response time; 14 citations; detected `force` overwrite flag, 409 status payload enhancement, and in-place array mutation in `CollaboratorBar.tsx`.
- **Quiz**: 4,440ms response time; 3/3 valid questions with 4 distinct explanation options; tested wrong-answer feedback, score display (2/3), and zero-request local retry reset.

### Scenario 3: `oss-worker-stream` (Python Async Distributed Worker)
- **Context**: Batched asynchronous dispatch with retry queue handling and cursor-based pagination.
- **Model**: `openai/gpt-oss-120b`
- **Explanation**: 5,675ms response time; 17 citations; 3 change-flow graph nodes.
- **Quiz**: 4,244ms response time; 2/2 valid questions; accurately tested `TimeoutError` handling and pagination cursor computation.

### Scenario 4: `oss-token-bucket-refactor` (Go Concurrency / Refactor)
- **Context**: Extracted token refill logic into helper functions and separated rate limit header generation.
- **Model**: `openai/gpt-oss-120b`
- **Explanation**: 6,282ms response time; 18 citations; 4 change-flow graph nodes.
- **Quiz**: 3,237ms response time; 2 questions evaluating rate limit header calculations and token replenishment semantics.

### Scenario 5: `oss-java-billing-sql` (Java Business Logic & SQL Migration)
- **Context**: Multi-tenant billing ledger modifications, credit tracking migration, and tenant isolation locks.
- **Model**: `openai/gpt-oss-120b`
- **Explanation**: 6,892ms response time; 21 citations; 4 change-flow graph nodes.
- **Quiz**: Reached Groq daily token quota (200k TPD) after extensive runs.

---

## 3. Multi-Model Verification Summary

1. **`openai/gpt-oss-120b`**:
   - Primary model for code understanding.
   - Generated thorough, highly factual explanations and valid quizzes across Go, Python, and Java repositories.
   - Reached the 200,000 tokens/day free-tier quota on Groq after 8+ comprehensive multi-scenario runs.

2. **`qwen/qwen3.8-27b`**:
   - Verified end-to-end in real Google Chrome on `oss-collab-web`.
   - Generated 14 exact line citations, detailed summaries, and 3/3 high-quality comprehension quiz questions with distinct options and feedback.
   - Required omitting `max_completion_tokens` to satisfy Groq's 1000 OTPM pre-check while allowing full generation.

3. **`openai/gpt-oss-20b`**:
   - Exhibited unstable JSON schema output on Groq (intermittently omitting the required `graph` key or producing malformed JSON strings).
   - Properly caught by VibeCodeFolks' validation layer with clear user guidance advising users to switch models.

4. **`Codex CLI` (`v0.166.0`, model `gpt-6.1-sol`)**:
   - Successfully discovered on local PATH and authenticated with ChatGPT login.
   - Live verified in real Google Chrome on `oss-collab-web`:
     - 29.4s explanation response time, 20 valid citations, 9 structured findings.
     - Identified critical concurrency and API contract bugs: `serverRevision` vs `revision` mismatch in conflict handling, in-place sorting mutation of caller arrays, and boundary filtering exclusion.
     - 18.5s dedicated comprehension quiz: 2 multi-choice questions with full distractor misconception explanations. Verified wrong-answer interactive feedback and 0-request retry.


---

## 4. Key Fixes Implemented

1. **Schema Property Ordering**: Switched from unordered `map[string]any` to structured Go types (`orderedObject`) to ensure deterministic property serialization in JSON Schema.
2. **Nullable Graph and Relationship Normalization**: Enhanced handling of optional/inferred graph connections and single-citation relationships to avoid false rejections.
3. **Lazy Model Discovery**: Added non-billable `GET /models` endpoint querying with chat model filtering and non-chat model rejection.
4. **Session Key Persistence**: Retained API keys and verification state across model and provider changes within the dashboard session.
5. **Precise Error Classification**: Separated daily token quota exhaustion (`tpd`) from rolling rate limits (`tpm`/`otpm`) and HTTP 413 request size errors.

