package analysis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxEvidenceFiles = 20
const MaxEvidenceFileBytes = 32 * 1024
const MaxEvidenceBytes = 160 * 1024

type EvidenceFile struct {
	Path         string         `json:"path"`
	PreviousPath string         `json:"previous_path,omitempty"`
	Type         string         `json:"type"`
	Excerpt      string         `json:"excerpt"`
	Lines        []EvidenceLine `json:"lines"`
}
type EvidenceLine struct {
	Side   string `json:"side"`
	Number int    `json:"number"`
	Text   string `json:"text"`
}
type SkippedFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}
type Evidence struct {
	Scope       string         `json:"scope"`
	Files       []EvidenceFile `json:"files"`
	Skipped     []SkippedFile  `json:"skipped"`
	Limitations []string       `json:"limitations"`
}

var secretPattern = regexp.MustCompile(`(?i)(-----BEGIN [A-Z ]*PRIVATE KEY-----|\b(?:sk|ghp|github_pat|xox[baprs])[-_][A-Za-z0-9_-]{12,}|\bAKIA[0-9A-Z]{16}\b|(?:password|passwd|secret|api[_-]?key|access[_-]?token|client[_-]?secret)\s*[:=]\s*["'][^"'\s]{8,})`)

func secretPath(p string) bool {
	for _, part := range strings.Split(strings.ToLower(filepath.ToSlash(p)), "/") {
		if part == ".env" || strings.HasPrefix(part, ".env.") || part == ".ssh" || part == ".aws" || part == "credentials" || part == "secrets" || strings.Contains(part, "credential") || strings.Contains(part, "secret") || strings.Contains(part, "private_key") || strings.HasSuffix(part, ".pem") || strings.HasSuffix(part, ".key") || strings.HasSuffix(part, ".p12") || strings.HasSuffix(part, ".pfx") {
			return true
		}
	}
	return false
}

// Reject traversal and symlinks, including parent directories, before inspecting a path.
func safePath(root, p string) (string, error) {
	if p == "" || filepath.IsAbs(p) {
		return "", errors.New("unsafe path")
	}
	clean := filepath.Clean(p)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("unsafe path")
	}
	cur := root
	for _, part := range strings.Split(clean, string(filepath.Separator)) {
		cur = filepath.Join(cur, part)
		fi, err := os.Lstat(cur)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("symlink excluded")
		}
	}
	return cur, nil
}

type limitedBuffer struct {
	b        bytes.Buffer
	limit    int
	overflow bool
}

func (w *limitedBuffer) Write(p []byte) (int, error) {
	if w.b.Len()+len(p) > w.limit {
		w.overflow = true
		return 0, errors.New("output limit exceeded")
	}
	return w.b.Write(p)
}
func evidenceGit(root string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	safe, err := safeGitArgs(root, args)
	if err != nil {
		return nil, err
	}
	c := exec.CommandContext(ctx, "git", safe...)
	c.Dir = root
	c.Env = append(cleanEnv(), "LC_ALL=C", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
	w := &limitedBuffer{limit: MaxEvidenceFileBytes}
	c.Stdout = w
	err = c.Run()
	if w.overflow {
		return nil, errors.New("diff exceeds size limit")
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		var ex *exec.ExitError
		if !(len(args) > 1 && args[0] == "diff" && args[1] == "--no-index" && errors.As(err, &ex) && ex.ExitCode() == 1) {
			return nil, fmt.Errorf("diff unavailable or exceeds size limit")
		}
	}
	return w.b.Bytes(), nil
}

func Collect(dir string) (Evidence, error) {
	changes, err := analyze(dir, true)
	if err != nil {
		return Evidence{}, err
	}
	result := Evidence{Scope: changes.Scope, Files: []EvidenceFile{}, Skipped: []SkippedFile{}, Limitations: append([]string{}, changes.Warnings...)}
	total := 0
	_, headErr := run(changes.Repository.Root, "rev-parse", "--verify", "HEAD")
	for _, f := range changes.Files {
		skip := func(reason string) { result.Skipped = append(result.Skipped, SkippedFile{f.Path, reason}) }
		if len(result.Files) >= MaxEvidenceFiles {
			skip("file count limit")
			continue
		}
		if f.Binary {
			skip("binary file")
			continue
		}
		if secretPath(f.Path) || secretPath(f.PreviousPath) {
			skip("sensitive filename")
			continue
		}
		full, e := safePath(changes.Repository.Root, f.Path)
		if e != nil {
			skip("unsafe path or symlink")
			continue
		}
		if f.PreviousPath != "" {
			if _, e = safePath(changes.Repository.Root, f.PreviousPath); e != nil {
				skip("unsafe previous path or symlink")
				continue
			}
		}
		if f.Details != "" {
			skip("diff details unavailable")
			continue
		}
		var source []byte
		st, statErr := os.Lstat(full)
		if statErr == nil {
			if !st.Mode().IsRegular() || st.Size() > MaxEvidenceFileBytes {
				skip("non-regular or oversized file")
				continue
			}
			file, openErr := os.Open(full)
			if openErr != nil {
				skip("file unavailable")
				continue
			}
			opened, openErr := file.Stat()
			if openErr != nil || !os.SameFile(st, opened) {
				file.Close()
				skip("file changed during inspection")
				continue
			}
			data, readErr := io.ReadAll(io.LimitReader(file, MaxEvidenceFileBytes+1))
			file.Close()
			if readErr != nil || len(data) > MaxEvidenceFileBytes {
				skip("file unavailable or exceeds size limit")
				continue
			}
			if secretPattern.Match(data) {
				skip("possible secret in file")
				continue
			}
			if utf8.Valid(data) && !bytes.Contains(data, []byte{0}) {
				source = data
			}
		} else if !os.IsNotExist(statErr) {
			skip("file unavailable")
			continue
		}

		var patch []byte
		if changes.Scope == "Working Tree" && (headErr != nil || f.Type == "added") {
			// Try tracked diff first: an added file can be staged or untracked.
			if headErr == nil {
				patch, e = evidenceGit(changes.Repository.Root, "diff", "--no-ext-diff", "--no-textconv", "--no-color", "--unified=3", "HEAD", "--", ":(literal)"+f.Path)
			}
			if len(patch) == 0 && e == nil {
				patch, e = evidenceGit(changes.Repository.Root, "diff", "--no-index", "--no-ext-diff", "--no-textconv", "--no-color", "--unified=3", "--", os.DevNull, full)
			}
		} else if changes.Scope == "Working Tree" {
			args := []string{"diff", "--no-ext-diff", "--no-textconv", "--no-color", "--find-renames", "--unified=3", "HEAD", "--", ":(literal)" + f.Path}
			if f.PreviousPath != "" {
				args = append(args, ":(literal)"+f.PreviousPath)
			}
			patch, e = evidenceGit(changes.Repository.Root, args...)
		} else {
			args := []string{"show", "--format=", "--root", "--no-ext-diff", "--no-textconv", "--no-color", "--find-renames", "--unified=3", "HEAD", "--", ":(literal)" + f.Path}
			if f.PreviousPath != "" {
				args = append(args, ":(literal)"+f.PreviousPath)
			}
			if _, pe := run(changes.Repository.Root, "rev-parse", "--verify", "HEAD^1"); pe == nil {
				args = []string{"diff", "--no-ext-diff", "--no-textconv", "--no-color", "--find-renames", "--unified=3", "HEAD^1", "HEAD", "--", ":(literal)" + f.Path}
				if f.PreviousPath != "" {
					args = append(args, ":(literal)"+f.PreviousPath)
				}
			}
			patch, e = evidenceGit(changes.Repository.Root, args...)
		}
		if e != nil {
			skip("diff unavailable or exceeds per-file limit")
			continue
		}
		if bytes.Contains(patch, []byte("Binary files ")) && bytes.Contains(patch, []byte(" differ")) {
			skip("binary file")
			continue
		}
		if !utf8.Valid(patch) || bytes.Contains(patch, []byte{0}) {
			skip("non-text diff")
			continue
		}
		if bytes.Contains(patch, []byte("120000")) && (bytes.Contains(patch, []byte("mode 120000"))) {
			skip("symlink diff")
			continue
		}
		if secretPattern.Match(patch) {
			skip("possible secret in diff")
			continue
		}
		hunkStart := hunkPattern.FindIndex(patch)
		if hunkStart == nil {
			skip("no text hunk available")
			continue
		}
		// Git's no-index headers can contain absolute local paths; only send hunk content.
		patch = patch[hunkStart[0]:]
		lines := parseEvidenceLines(patch)
		if len(lines) == 0 {
			skip("no text hunk available")
			continue
		}
		// Single-line imports are optional context, not semantic relationship claims.
		contextLines := importLines(source)
		if len(contextLines) > 0 {
			patch = append(patch, []byte("\n[Additional import context from the current file]\n")...)
			for _, line := range contextLines {
				patch = append(patch, []byte(fmt.Sprintf("%d: %s\n", line.Number, line.Text))...)
				lines = append(lines, line)
			}
		}
		if total+len(patch) > MaxEvidenceBytes {
			skip("total context limit")
			continue
		}
		total += len(patch)
		result.Files = append(result.Files, EvidenceFile{Path: f.Path, PreviousPath: f.PreviousPath, Type: f.Type, Excerpt: string(patch), Lines: lines})
	}
	result.Limitations = append(result.Limitations, "Evidence contains limited diff context; static analysis does not verify runtime behavior. Secret detection is best-effort.")
	return result, nil
}
func parseEvidenceLines(patch []byte) []EvidenceLine {
	var out []EvidenceLine
	old, nw := 0, 0
	active := false
	for _, line := range strings.Split(string(patch), "\n") {
		if m := hunkPattern.FindStringSubmatch(line); m != nil {
			fmt.Sscanf(m[1], "%d", &old)
			fmt.Sscanf(m[3], "%d", &nw)
			active = true
			continue
		}
		if !active || len(line) == 0 {
			continue
		}
		switch line[0] {
		case ' ':
			out = append(out, EvidenceLine{"old", old, line[1:]}, EvidenceLine{"new", nw, line[1:]})
			old++
			nw++
		case '-':
			out = append(out, EvidenceLine{"old", old, line[1:]})
			old++
		case '+':
			out = append(out, EvidenceLine{"new", nw, line[1:]})
			nw++
		}
	}
	return out
}

var importPattern = regexp.MustCompile(`^\s*(?:import\s+[^({]|from\s+\S+\s+import\s+|use\s+[A-Za-z_]|require(?:_once)?\s*\()`)

func importLines(source []byte) []EvidenceLine {
	var lines []EvidenceLine
	for n, line := range strings.Split(string(source), "\n") {
		if len(line) <= 300 && importPattern.MatchString(line) {
			lines = append(lines, EvidenceLine{Side: "new", Number: n + 1, Text: line})
			if len(lines) == 8 {
				break
			}
		}
	}
	return lines
}
