# VibeCodeFolks

**AI wrote it. Do you understand it?**

VibeCodeFolks is an open-source developer tool for understanding changes made by AI coding agents. The planned MVP will inspect Git diffs, explain changes and relationships using code evidence, identify uncertainty, and ask questions that help developers check their understanding.

This first phase is a local dashboard shell. It does not yet analyze code, generate diagrams or questions, or use AI. It runs a Go HTTP server on a loopback address and embeds its HTML, CSS, and JavaScript in the executable. No database or account is required.

## Run

Install Go 1.21 or later, then from this project directory:

```sh
go run .
go run . /path/to/project
```

The optional argument selects a project directory; it defaults to the current directory. The directory is validated and printed, but is not analyzed in this phase. The browser opens automatically when supported; otherwise, open the printed URL. Stop the server with Ctrl+C.

## Build and test

```sh
go test ./...
go build -o vibecodefolks .
./vibecodefolks /path/to/project
```

The executable contains the dashboard assets and requires no Node.js, Python, or Electron runtime.

## License

MIT. See [LICENSE](LICENSE).
