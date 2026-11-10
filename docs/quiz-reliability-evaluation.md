# Quiz reliability evaluation

## Pipeline and diagnostics

The app collects bounded, filtered Git evidence and prepares an explicit provider request. Codex receives the evidence and a JSON schema in a temporary directory. The provider output is decoded, explanation findings are checked, and optional quiz data is validated against the same provider evidence before it is sent to the dashboard. A valid explanation survives an omitted or invalid quiz. The UI now reports the fixed quiz diagnostic category, without including source, prompt, credentials, model response, or stderr.

Categories are `available`, `missing`, `model_null`, `empty`, `invalid_structure`, and `invalid_reference`. Missing or invalid entire provider JSON remains an explanation-level error. The UI does not independently validate semantics and the server's validation cannot prove a correct answer or plausible distractors.

The previous nullable quiz schema allowed a model to choose null even when evidence supported questions. In the previous evaluation one captured response explicitly contained null. Other missing quizzes had no retained classification, so their causes cannot be reconstructed. The current strict schema requires a quiz object with a questions array. An empty array remains valid when no defensible question exists. This eliminates null as a schema-compliant Codex output, but cannot compel the model to produce a suitable question.

## Evidence eligibility

All 15 existing scenarios contain at least one locally testable code path or conditional trace, so they are provisionally eligible for at least one objective question with explicit assumptions. Eligibility does not imply that every requested topic or every question is supportable. Unseen routing, runtime, type contracts, database locking and external caller behavior remain uncertain. The ten additional fixtures were classified from the written references, not run in this evaluation. The unrelated README change in mixed-checkout is not itself evidence for an engineering-behavior quiz. No fixture was provisionally classified as ineligible, so the ineligible availability denominator is zero.

Eligible fixtures: authentication, business-logic, mixed-checkout, react-search, cursor-pagination, zero-stock, tenant-cache, webhook-retry, render-lookups, shared-normalization, react-state, java-validation, sql-inventory, async-gather and pure-refactor. Reference analyses are in v1-quality-evaluation.md.

## Bounded real Codex evaluation

No second model call was added to the application. All requests used the installed signed-in Codex CLI and its existing default model. The model ID was not exposed in the local configuration. Five eligible scenarios were run twice each. The machine-readable classification records contain only fixture name, attempt number, HTTP status, fixed diagnostic and question count. They are stored with owner-only permissions outside the source tree.

| Scenario | Requests | Accepted explanations | Empty question arrays | Validator rejections | Displayed quizzes |
| --- | ---: | ---: | ---: | ---: | ---: |
| business-logic | 2 | 2 | 2 | 0 | 0 |
| react-state | 2 | 2 | 2 | 0 | 0 |
| cursor-pagination | 2 | 2 | 2 | 0 | 0 |
| tenant-cache | 2 | 2 | 2 | 0 | 0 |
| webhook-retry | 2 | 2 | 2 | 0 | 0 |
| Total eligible | 10 | 10 | 10 | 0 | 0 |

The model explicitly returned empty arrays in all ten accepted results. There were 0 model-null results, 0 missing quiz fields, 0 malformed quiz structures, 0 invalid-reference rejections, and 0 explanation failures. The provider returned JSON accepted by the structured schema for 10/10 attempts. There were no displayed questions in this sample, so the answer-key and distractor-quality denominators are both zero. A separate real browser Codex request on business-logic also returned an explanation and an empty quiz. That request is not included in the ten-request table.

For comparison, the earlier original-prompt evaluation on these five scenarios returned 3 accepted explanations and 3 quizzes in 5 requests, with 2 explanation failures. A later revised prompt run returned 1 quiz in 3 eligible requests. These are small, non-paired samples and were run on different occasions. The current 0/10 result does not support a claim of improved quiz availability.

## Interaction and failure verification

A browser run with a scripted loopback provider verified a valid quiz, source references, incorrect-answer feedback, score and retry. Modifying the fictional fixture caused the next retry to reject the stale evidence; the file was then restored. A real Codex browser run confirmed that an empty quiz produces the model-omission message while the explanation remains available. The scripted result is not evidence of model quality.

Unit tests cover omitted fields, explicit null, empty arrays, malformed quiz data, invalid references and a valid quiz, plus preservation of the explanation. Codex execution now distinguishes missing executable, cancellation, timeout, a narrowly recognized sign-in message and other launch/process failures. Only fixed messages leave the server; bounded stderr is classified locally and never returned. Failure reason matching is intentionally conservative and may still report a generic process failure.

## Release assessment

The quiz is not ready for v1.0.0. Ten eligible single-request samples produced no questions. Semantic correctness, mutually exclusive answers, and plausibility cannot be guaranteed by structural validation, and there were no new questions to assess. No current answer-key or distractor pass rate can be calculated.

The smallest realistic next experiment is an explicit, user-initiated quiz-generation retry against the already reviewed evidence, with clear notice that it is another provider request and may incur cost. It should not happen automatically. Measure a bounded comparison before adopting it, since a second request may still omit a quiz or produce ambiguous questions. Another option is investigating why the active Codex model selects an empty array under the combined schema; the current evidence does not establish the cause.
