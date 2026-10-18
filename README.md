# VibeCodeFolks

**AI wrote it. Do you understand it?**

VibeCodeFolks is an open-source developer tool for understanding changes made by AI coding agents. It currently discovers a local Git repository and displays language-independent file changes and Git diff statistics in a local dashboard. The planned MVP will explain changes and relationships with code evidence, identify uncertainty, and ask questions to check understanding.

## Run

Install Git and Go 1.21 or later, then from this project directory:

```sh
go run .
go run . /path/to/project
```

The optional argument selects a directory inside the repository; it defaults to the current directory. The browser opens automatically when supported; otherwise, open the printed URL. Stop the server with Ctrl+C.

## Build and test

```sh
go vet ./...
go test ./...
go build -o vibecodefolks .
./vibecodefolks /path/to/project
```

The executable embeds the dashboard assets and requires no Node.js, Python, or Electron runtime. The server listens only on `127.0.0.1`; `GET /api/changes` returns analysis for the selected repository. Refresh in the dashboard reruns the analysis.

## Analysis behavior and limitations

When changes are present, the dashboard shows the combined working tree difference from `HEAD`, plus untracked non-ignored files. When clean, it shows the most recent commit against its first parent as **Last Commit**, including an initial commit. A repository without commits is compared with an empty tree where possible. Changes canceled out between the index and working tree may have no combined difference; a warning is shown. A staged deletion followed by an untracked replacement at the same path is reported once, with net statistics unavailable. Git decides rename and binary detection. Binary files and other unavailable details do not contribute to line totals. Submodule changes and unusual Git states may have limited detail. Unmerged conflicts must be resolved before analysis. Files can change while analysis is running, so refresh after edits.

This phase reports files and diff evidence, not functions, APIs, runtime relationships, AI explanations, or quizzes. No repository content is sent to a cloud service. Git itself may invoke configured filters; users should trust their local Git configuration.

## License

MIT. See [LICENSE](LICENSE).
