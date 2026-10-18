# Manual real-provider evaluation

Status: **pending**. No real provider was configured in the development environment used for this hardening work. The scripted tests verify integration and validation, not model accuracy, prompt-injection immunity, or runtime behavior.

Use a small disposable repository containing only fictional, nonsensitive code. Review the provider's pricing, data retention, and configuration before making a request. Never put an API key in the repository, a report, or an issue. Do not run this procedure against private source unless you have permission to share it with the chosen provider.

1. Create a temporary Git repository with a fictional frontend login function and backend route. Commit the initial state, then modify both files. Include only code you are willing to send to the provider.
2. Configure `VIBECODEFOLKS_AI_ENDPOINT` with the provider's full chat-completions URL, `VIBECODEFOLKS_AI_MODEL` with its model, and, for a remote provider, `VIBECODEFOLKS_AI_API_KEY` in the shell. Use HTTPS for remote endpoints. Start `./vibecodefolks /path/to/sample`.
3. Select **Explain Changes**, review the exact evidence preview and destination, and explicitly consent before sending to a remote provider. Do not send if the preview contains anything sensitive. Repeat a few times only with authorization for any associated costs.
4. Compare every claim and quiz answer against the cited lines. Record whether the response is valid JSON, whether observations and inferences are distinguished, whether graph nodes and edges are justified, whether questions are relevant and their answer keys are correct, and whether the wording is clear to a beginner. Record factual errors and missing context without copying credentials or private data.
5. To check optional-data failure handling without relying on a model to malfunction, use the automated mocked-provider tests: malformed or absent graph/quiz data must preserve a valid text explanation. Compare results across repeated real requests, recording provider/model version and date without including keys.

## Report template

- Date and provider/model (no credentials):
- Sample changes and expected behavior:
- Requests attempted; valid responses:
- Observed changes and unsupported claims:
- Graph usefulness and unsupported edges:
- Quiz relevance and incorrect answer keys:
- Clarity, omissions, and follow-up actions:
- Limitations of the test, cost authorization, and data sent:

An evidence citation confirms that a line was included, not that the provider understood it. Do not describe the model as validated based solely on the scripted test suite.
