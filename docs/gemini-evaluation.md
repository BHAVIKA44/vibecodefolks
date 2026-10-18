# Gemini evaluation

Date: 2026-10-18. Model: `gemini-3.8-flash`, using Google's OpenAI-compatible chat-completions endpoint. Two real inference requests were made, both through the dashboard after reviewing and consenting to fictional evidence. Both returned responses accepted by the existing structured JSON and reference validation. Account-specific model listing was not checked separately; the successful requests establish that this model was available to the configured key during the evaluation. There were no automatic retries. The user reported that free quota was available.

## Reproducible fictional scenarios

Create disposable repositories outside the VibeCodeFolks checkout:

```sh
python3 tests/fixtures/create_gemini_scenarios.py /tmp/vibecodefolks-gemini-samples
```

The destination must not already exist. The generator creates committed baseline files and leaves deterministic working-tree changes. It does not execute sample code.

- **Authentication:** the frontend sends `username` and `password` as JSON to `/api/login`; the backend checks whether both are present, returning a 400 error when either is missing. The success path is fictional and does not verify passwords. There is no route registration in this sample, so a running connection is not established by the evidence.
- **Business logic:** the Go discount function changes its return signature to include an error, rejects negative totals, returns no discount below 100 cents, and otherwise returns one tenth of the total using integer division.

These fixtures contain no real credentials. Review the application's evidence preview and consent before sending it to a remote provider. See [provider-evaluation.md](provider-evaluation.md) for the full procedure.

## Results

Scores are subjective assessments from one response per scenario, on a 0–5 scale. They do not measure consistency across repeated responses.

| Scenario | Explanation accuracy | Graph usefulness | Quiz correctness | Beginner clarity |
| --- | ---: | ---: | ---: | ---: |
| Authentication | 4/5 | 4/5 | 5/5 | 4/5 |
| Go discount | 5/5 | 2/5 | 5/5 | 4/5 |

**Authentication:** The explanation identified the JSON request, the 400 response for missing fields, the success response, and the change to a parsed JSON return. It correctly noted that the routing configuration was absent and that runtime behavior was unverified. The graph showed two meaningful nodes and an inferred edge grounded in both files. The three questions covered status code, parsed return value, and payload keys; all three answer keys were correct against the source. The answer could have more prominently stated that the sample does not verify the password: the response described a success condition, but its graph's 'sends JSON credentials' connection may sound more certain than the missing route evidence permits. The interface did label that connection as inferred.

**Go discount:** The explanation correctly identified the signature change, negative-input error, threshold and integer division, and noted that callers may need changes. All three quiz answer keys were correct: the new signature, `(0, nil)` for 50, and the negative-total error message. The dashboard displayed a file-only fallback graph, which added little to the explanation. The retained response was not captured before optional graph validation, so it is unknown whether Gemini omitted its graph or validation rejected it; no claim about the cause is justified.

## Limitations and follow-up

No provider compatibility fix was required by these two requests; existing OpenAI-style requests worked. No prompt or validation controls were changed. The explanation and quiz remained available when the second graph fell back to Git evidence. Optional graph and quiz rejection paths are additionally covered by mocked tests; neither provider failure mode was forced in these real requests.

Two small samples and six questions are insufficient to establish general accuracy, consistency, security, or production readiness. Citation validation checks that references exist in supplied evidence, not that claims are true. Secret filtering and prompt-injection defenses remain best effort. Test larger and more varied changes, malformed optional outputs, model repeatability, and usability with outside developers before a broader beta. A small supervised trial on nonsensitive repositories may be useful with these limits clearly explained.
