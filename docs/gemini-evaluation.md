# Gemini evaluation

Status: **pending** (2026-10-18). No API key was available in the evaluation process. No model availability check or inference request was made; no Gemini model has been tested. The requested candidate is `gemini-3.8-flash`, subject to an authenticated availability check.

## Reproducible fictional scenarios

Create disposable repositories outside the VibeCodeFolks checkout:

```sh
python3 tests/fixtures/create_gemini_scenarios.py /tmp/vibecodefolks-gemini-samples
```

The destination must not already exist. The generator creates committed baseline files and leaves deterministic working-tree changes. It does not execute sample code.

- **Authentication:** the frontend sends `username` and `password` as JSON to `/api/login`; the backend checks whether both are present, returning a 400 error when either is missing. The success path is fictional and does not verify passwords. There is no route registration in this sample, so a running connection is not established by the evidence.
- **Business logic:** the Go discount function changes its return signature to include an error, rejects negative totals, returns no discount below 100 cents, and otherwise returns one tenth of the total using integer division.

These fixtures contain no real credentials. Before sending, review the application's evidence preview and consent to the indicated remote provider. See [provider-evaluation.md](provider-evaluation.md) for the full procedure. Do not place API keys in fixtures or reports.

## Results to complete after a real run

- Model and availability: pending.
- Real inference requests: **0** (maximum authorized for this evaluation: 3).
- Request outcomes and structured JSON validation: pending.
- Explanation accuracy, graph usefulness, quiz correctness, and beginner clarity: not scored; no model output was examined.
- Factual errors, targeted fixes, and further testing: pending.

Automated mocked tests do not demonstrate model accuracy or provider compatibility. A successful structural validation does not prove claims or quiz answers are correct.
