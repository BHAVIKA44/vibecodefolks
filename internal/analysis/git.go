package analysis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RepositoryInfo struct {
	Name     string `json:"name"`
	Root     string `json:"root"`
	Branch   string `json:"branch"`
	Detached bool   `json:"detached"`
}
type Hunk struct {
	OldStart int `json:"old_start"`
	OldLines int `json:"old_lines"`
	NewStart int `json:"new_start"`
	NewLines int `json:"new_lines"`
}
type ChangedFile struct {
	Path         string `json:"path"`
	PreviousPath string `json:"previous_path,omitempty"`
	Type         string `json:"type"`
	Added        *int   `json:"lines_added"`
	Deleted      *int   `json:"lines_deleted"`
	Binary       bool   `json:"binary"`
	Hunks        []Hunk `json:"hunks"`
	Details      string `json:"details,omitempty"`
}
type ChangeSet struct {
	Repository RepositoryInfo `json:"repository"`
	Scope      string         `json:"scope"`
	Files      []ChangedFile  `json:"files"`
	TotalFiles int            `json:"total_files"`
	Added      int            `json:"lines_added"`
	Deleted    int            `json:"lines_deleted"`
	Warnings   []string       `json:"warnings"`
}

type gitRepo struct{ root string }

func run(dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(cleanEnv(), "LC_ALL=C", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("git timed out: %w", ctx.Err())
		}
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errors.New("Git is not installed or not available on PATH")
		}
		return nil, fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
func cleanEnv() []string { return filteredEnv() }

func Discover(dir string) (RepositoryInfo, error) {
	root, err := run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return RepositoryInfo{}, fmt.Errorf("could not discover a Git repository in %q: %w", dir, err)
	}
	// --show-toplevel terminates its result with one newline; preserve any others in a path.
	path := strings.TrimSuffix(string(root), "\n")
	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(filepath.Join(dir, path))
		if err != nil {
			return RepositoryInfo{}, err
		}
	}
	branch, err := run(path, "symbolic-ref", "--quiet", "--short", "HEAD")
	detached := false
	if err != nil {
		head, e := run(path, "rev-parse", "--verify", "HEAD")
		if e != nil {
			return RepositoryInfo{}, fmt.Errorf("cannot determine branch: %w", err)
		}
		detached = true
		branch = []byte(strings.TrimSpace(string(head)))
		if len(branch) > 12 {
			branch = branch[:12]
		}
	}
	return RepositoryInfo{Name: filepath.Base(path), Root: path, Branch: strings.TrimSuffix(string(branch), "\n"), Detached: detached}, nil
}

func noIndex(dir string, option string, extraOrPath string, more ...string) ([]byte, error) {
	args := []string{"diff", "--no-index", "--no-ext-diff", "--no-textconv", option}
	if len(more) > 0 {
		args = append(args, extraOrPath)
		args = append(args, "--", os.DevNull, more[0])
	} else {
		args = append(args, "--", os.DevNull, extraOrPath)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(cleanEnv(), "LC_ALL=C", "GIT_OPTIONAL_LOCKS=0", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return nil, err
		}
	}
	return out, nil
}
