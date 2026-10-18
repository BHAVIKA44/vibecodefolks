package analysis

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var hunkPattern = regexp.MustCompile(`(?m)^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

func parseHunks(data []byte) []Hunk {
	out := []Hunk{}
	for _, m := range hunkPattern.FindAllSubmatch(data, -1) {
		values := [4]int{}
		for i := 0; i < 4; i++ {
			if len(m[i+1]) == 0 {
				values[i] = 1
			} else {
				values[i], _ = strconv.Atoi(string(m[i+1]))
			}
		}
		out = append(out, Hunk{values[0], values[1], values[2], values[3]})
	}
	return out
}
func tokens(data []byte) []string {
	parts := strings.Split(string(data), "\x00")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}
func changeType(code byte) string {
	switch code {
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'T':
		return "type changed"
	case 'C':
		return "copied"
	case 'U':
		return "unmerged"
	default:
		return "modified"
	}
}
func parseNames(data []byte) ([]ChangedFile, error) {
	t := tokens(data)
	files := []ChangedFile{}
	for i := 0; i < len(t); {
		status := t[i]
		i++
		if status == "" || i >= len(t) {
			return nil, fmt.Errorf("invalid Git name-status output")
		}
		f := ChangedFile{Type: changeType(status[0]), Hunks: []Hunk{}}
		if status[0] == 'R' || status[0] == 'C' {
			if i+1 >= len(t) {
				return nil, fmt.Errorf("invalid Git rename output")
			}
			f.PreviousPath = t[i]
			i++
		}
		f.Path = t[i]
		i++
		files = append(files, f)
	}
	return files, nil
}
func parseStats(data []byte) (map[string][2]string, error) {
	t := tokens(data)
	result := map[string][2]string{}
	for i := 0; i < len(t); i++ {
		parts := strings.SplitN(t[i], "\t", 3)
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid Git numstat output")
		}
		path := parts[2]
		if path == "" {
			if i+2 >= len(t) {
				return nil, fmt.Errorf("invalid Git rename numstat output")
			}
			i += 2
			path = t[i]
		}
		result[path] = [2]string{parts[0], parts[1]}
	}
	return result, nil
}
func applyStats(f *ChangedFile, s [2]string) {
	if s[0] == "-" || s[1] == "-" {
		f.Binary = true
		return
	}
	a, e1 := strconv.Atoi(s[0])
	d, e2 := strconv.Atoi(s[1])
	if e1 != nil || e2 != nil {
		f.Details = "Line statistics unavailable"
		return
	}
	f.Added = &a
	f.Deleted = &d
}

func Analyze(dir string) (ChangeSet, error) { return analyze(dir, false) }

func analyze(dir string, evidenceOnly bool) (ChangeSet, error) {
	info, err := Discover(dir)
	if err != nil {
		return ChangeSet{}, err
	}
	result := ChangeSet{Repository: info, Files: []ChangedFile{}, Warnings: []string{}}
	status, err := run(info.Root, "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=none")
	if err != nil {
		return result, err
	}
	raw := tokens(status)
	var untracked []string
	var initialPaths []string
	for i := 0; i < len(raw); i++ {
		entry := raw[i]
		if len(entry) < 4 {
			return result, fmt.Errorf("invalid Git status output")
		}
		code := entry[:2]
		path := entry[3:]
		if strings.Contains(code, "U") || code == "AA" || code == "DD" {
			return result, fmt.Errorf("unmerged files are not supported; resolve conflicts and refresh")
		}
		if code == "??" {
			untracked = append(untracked, path)
		} else {
			initialPaths = append(initialPaths, path)
		}
		if strings.ContainsAny(code, "RC") {
			i++
			if i >= len(raw) {
				return result, fmt.Errorf("invalid Git status rename")
			}
		}
	}
	_, headErr := run(info.Root, "rev-parse", "--verify", "HEAD")
	if len(raw) > 0 {
		result.Scope = "Working Tree"
	} else if headErr == nil {
		result.Scope = "Last Commit"
	} else {
		result.Scope = "Working Tree"
		result.Warnings = append(result.Warnings, "Repository has no commits or changes.")
		return result, nil
	}
	if headErr != nil { // All existing tracked paths are additions in an unborn repository.
		result.Warnings = append(result.Warnings, "Repository has no commits; changes are compared with an empty tree. Staged-only content that no longer exists in the working tree is unavailable.")
		untracked = append(untracked, initialPaths...)
	} else {
		command := []string{"diff", "--no-ext-diff", "--no-textconv", "--find-renames"}
		if result.Scope == "Last Commit" {
			if _, e := run(info.Root, "rev-parse", "--verify", "HEAD^1"); e == nil {
				command = append(command, "HEAD^1")
			} else {
				command = []string{"show", "--format=", "--root", "--no-ext-diff", "--no-textconv", "--find-renames"}
			}
		}
		names, err := run(info.Root, append(append([]string{}, command...), "--name-status", "-z", "HEAD", "--")...)
		if err != nil {
			return result, err
		}
		files, err := parseNames(names)
		if err != nil {
			return result, err
		}
		stats, err := run(info.Root, append(append([]string{}, command...), "--numstat", "-z", "HEAD", "--")...)
		if err != nil {
			return result, err
		}
		parsed, err := parseStats(stats)
		if err != nil {
			return result, err
		}
		for i := range files {
			f := &files[i]
			if s, ok := parsed[f.Path]; ok {
				applyStats(f, s)
			} else {
				f.Details = "Line statistics unavailable"
			}
			if evidenceOnly {
				result.Files = append(result.Files, *f)
				continue
			}
			args := append(append([]string{}, command...), "--unified=0", "HEAD", "--", ":(literal)"+f.Path)
			if f.PreviousPath != "" {
				args = append(args, ":(literal)"+f.PreviousPath)
			}
			patch, e := run(info.Root, args...)
			if e != nil {
				f.Details = "Hunk locations unavailable"
			} else {
				f.Hunks = parseHunks(patch)
			}
			result.Files = append(result.Files, *f)
		}
	}
	seen := map[string]bool{}
	for _, f := range result.Files {
		seen[f.Path] = true
	}
	for _, path := range untracked {
		if seen[path] {
			// A staged deletion followed by an untracked replacement shares a path.
			// Git's diff reports only the deletion, so do not present it as the net result.
			for i := range result.Files {
				if result.Files[i].Path == path {
					result.Files[i].Type = "modified"
					result.Files[i].Added, result.Files[i].Deleted = nil, nil
					result.Files[i].Binary = false
					result.Files[i].Hunks = []Hunk{}
					result.Files[i].Details = "Staged deletion with untracked replacement; net diff unavailable"
				}
			}
			continue
		}
		seen[path] = true
		f := ChangedFile{Path: path, Type: "added", Hunks: []Hunk{}}
		full := filepath.Join(info.Root, path)
		stat, e := os.Lstat(full)
		if e != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: working-tree content unavailable", path))
			continue
		}
		if !stat.Mode().IsRegular() {
			f.Details = "Line statistics unavailable for non-regular file"
			result.Files = append(result.Files, f)
			continue
		}
		if evidenceOnly {
			result.Files = append(result.Files, f)
			continue
		}
		// Exit status 1 from --no-index means differences, so its output is collected separately.
		statData, e := noIndex(info.Root, "--numstat", "-z", full)
		if e != nil {
			f.Details = "Line statistics unavailable"
		} else {
			p, e := parseStats(statData)
			if e == nil {
				for _, s := range p {
					applyStats(&f, s)
					break
				}
			} else {
				f.Details = "Line statistics unavailable"
			}
		}
		patch, e := noIndex(info.Root, "--unified=0", full)
		if e == nil {
			f.Hunks = parseHunks(patch)
		} else if f.Details == "" {
			f.Details = "Hunk locations unavailable"
		}
		result.Files = append(result.Files, f)
	}
	sort.Slice(result.Files, func(i, j int) bool { return result.Files[i].Path < result.Files[j].Path })
	for _, f := range result.Files {
		if f.Binary || f.Added == nil || f.Deleted == nil {
			result.Warnings = append(result.Warnings, "Line totals exclude binary files or files with unavailable statistics.")
			break
		}
	}
	result.TotalFiles = len(result.Files)
	for _, f := range result.Files {
		if f.Added != nil {
			result.Added += *f.Added
		}
		if f.Deleted != nil {
			result.Deleted += *f.Deleted
		}
	}
	if len(result.Files) == 0 && len(raw) > 0 {
		result.Warnings = append(result.Warnings, "Changes exist in the index or working tree, but their combined difference from HEAD is empty or unavailable.")
	}
	return result, nil
}
