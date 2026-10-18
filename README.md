# VibeCodeFolks

**AI wrote it. Do you understand it?**

VibeCodeFolks is a local tool for understanding code changes made by AI coding agents. Point it at a Git repository, review the changes, and ask an AI provider to explain the evidence. Explore a change map and check your understanding with a short quiz.

One Go executable. A local browser dashboard. No database, frontend runtime, or account required.

> **Early beta:** AI explanations, connections, and quiz answers can be wrong. Review the cited code and run your own tests. Start with repositories that contain no sensitive information.

![Local Git dashboard](docs/dashboard.png)

The dashboard screenshot shows a fictional sample repository; the graph and quiz screenshots below use a scripted mock provider.

## Quick start

Install Git and Go 1.21 or later, then build from source:

```sh
git clone https://github.com/BHAVIKA44/vibecodefolks.git
cd vibecodefolks
go build -o vibecodefolks .
./vibecodefolks /path/to/project
```

The project path can point anywhere inside a Git repository. Omit it to inspect the current directory. The CLI opens a dashboard on `127.0.0.1`; if the browser does not open, visit the printed URL. Stop the server with Ctrl+C.

```sh
# Or run directly from the VibeCodeFolks checkout
go run . /path/to/project
```

The dashboard assets are embedded in the executable. Node.js, Python, Electron, a database, and accounts are not required.

## How it works

1. An AI coding agent changes your repository.
2. Run VibeCodeFolks and inspect the Git changes in your browser.
3. Configure an AI provider, review the evidence preview, and choose whether to send it.
4. Explore the explanation and change map, then try the optional quiz.

## Features

- Inspect working-tree Git changes or, when clean, the last commit.
- Review bounded code evidence before sending it to a configured provider.
- Explore an optional AI-generated change map and understanding quiz with cited source excerpts.
- Run locally without a database, cloud service, or frontend runtime dependency.

## Git changes

The dashboard shows the combined working-tree diff from `HEAD`, plus untracked non-ignored files. When the tree is clean it displays **Last Commit**, compared to its first parent. Initial commits, repositories without commits, detached HEAD, renames, and binary files are supported. Refresh reruns Git analysis. Changes canceled between the index and working tree can have no combined difference; a warning is shown. Staged deletions followed by an untracked replacement are shown once with unavailable net statistics.

## Optional AI explanation

Set these environment variables before starting the CLI:

```sh
export VIBECODEFOLKS_AI_ENDPOINT="http://127.0.0.1:11434/v1/chat/completions"
export VIBECODEFOLKS_AI_MODEL="your-model-name"
# For remote providers, also set VIBECODEFOLKS_AI_API_KEY in your shell.
./vibecodefolks /path/to/project
```

Use the provider's full chat-completions URL. Remote endpoints require HTTPS and an API key; loopback endpoints can use HTTP and do not require a key. Providers must support OpenAI-compatible chat completions and JSON object responses. No AI request is made when the dashboard opens. Click **Explain Changes**, review the evidence and provider details, then explicitly send it. Remote providers require the consent checkbox. Canceling a provider request cancels the HTTP call where possible.

The server collects unified diff hunks with three lines of surrounding context, including untracked additions, and up to eight straightforward single-line import statements from each current file. It sends the selected file paths, diff excerpts, line references, scope, and limitations to the provider. It excludes binary files, symlinks, common credential filenames, detected secret patterns, oversized files and diffs, and files beyond configured limits. At most 20 files, 32 KiB per file/diff, and 160 KiB total diff text are included. The serialized provider evidence is additionally capped at 512 KiB. Skipped files and reasons are displayed. Secret detection is **best effort** and cannot guarantee all sensitive information is removed: inspect the preview before sending. Your provider may retain submitted data. The local server does not log source snippets, prompts, or keys.

Git filter and filesystem monitor commands are disabled during analysis. The server listens only on `127.0.0.1`, checks the request host and origin, and does not accept arbitrary project paths via the API. It is designed for a trusted local machine, not as a multi-user server.

### Visual change flows

After the explanation is ready, the change map uses the same evidence snapshot to show suggested components and connections. Select a card or arrow with the mouse or keyboard to see its description, uncertainty, and supporting diff excerpts. For example, a change to `web/login.js` and `api/auth.py` might show **Login form → Authentication API** when both files contain relevant evidence. The arrow is an AI inference; matching paths do not prove the request works at runtime. The screenshot below uses a scripted mock provider and a sample repository.

![Sample change flow using a mock provider](docs/change-flow.png)

The snapshot timestamp and identifier describe the evidence sent to the provider. Refreshing Git changes clears the previous explanation and diagram; run **Explain Changes** again to analyze the new snapshot. If the provider omits a graph or returns an invalid one, the explanation remains available and the map shows disconnected file cards based on the collected evidence. It does not invent connections. Currently all displayed connections are AI inferences; no deterministic relationship extraction is performed. The map accepts at most 20 nodes and 30 edges, so larger changes may be summarized or fall back to file cards. Scroll the diagram horizontally for wide flows.

### Understanding quiz

After an explanation is generated, choose **Okay bro, did you actually understand it? 💀** to practice with up to five multiple-choice questions about the supplied code evidence. The same AI request produces the explanation, diagram, and optional quiz; starting or retrying a quiz does not send code to the AI provider again. If the evidence or model response cannot support a quiz, the dashboard says so and the explanation remains available.

Each answer receives immediate feedback, an AI-generated explanation, and expandable source references. The final score is the number of correct choices out of the questions presented. Answers and scores remain in the current browser session. Retry starts a new attempt with the same questions. The app checks the local evidence snapshot before starting or retrying; refresh clears the current quiz and score.

![Understanding quiz with feedback from a scripted mock provider](docs/understanding-quiz.png)

Questions and answer keys are AI interpretations. The server checks their format and cited lines, but cannot guarantee factual correctness or verify behavior at runtime. Review the cited code, and run appropriate tests. The score is a learning aid, not a certification.

### Example output

For a diff that adds a `validateEmail` function, an explanation might say:

- **What changed:** Added a function that checks whether an email is present, citing the added lines.
- **Why it matters:** The check may prevent empty submissions; this is an inference.
- **What we couldn't verify:** Whether all callers use the function and how the application behaves at runtime.

AI output is an interpretation, not a test result. The server checks that each finding cites a real line included in its evidence; a valid citation does not guarantee the model interpreted that line correctly.

## Limitations

Evidence is language-independent text and can include React/TypeScript, Python, Java, Go, and other text-based projects. It is not a full parser for every language or framework, and semantic accuracy has not been established equally across stacks. Only changed diff context is supplied; unchanged files, large changes, pure renames without textual hunks, binary content, submodules, symlinks, and other skipped material may be missing. The provider may fail, return invalid JSON or invalid references, or misunderstand valid evidence; invalid structured responses are rejected. Filters are heuristic, and files can change during analysis. Unmerged conflicts must be resolved first. The app does not execute or test the project and cannot confirm runtime behavior. Citations establish that the referenced lines exist in the supplied evidence, not that an inferred relationship is correct.

## Development and contributing

Install Git, Go 1.21 or newer, and Node.js for the frontend interaction tests. Node.js is a development dependency only.

```sh
gofmt -l .
go vet ./...
go test ./...
go build ./...
node --check web/app.js
node --check web/graph.js
node --check web/quiz.js
node --test tests/dashboard.test.cjs
```

Contributions are welcome: open an issue to discuss substantial changes, then submit a focused pull request with tests and a description of how you verified it. CI is configured for Linux, macOS, and Windows. See the [Gemini evaluation](docs/gemini-evaluation.md), [beta testing notes](docs/beta-testing.md), [provider evaluation procedure](docs/provider-evaluation.md), and [future release plan](docs/release-plan.md). Provider quality remains an open question beyond the small samples tested.

## License

MIT. See [LICENSE](LICENSE).
