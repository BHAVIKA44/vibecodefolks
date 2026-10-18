package analysis

import (
	"os"
	"strings"
)

// Ignore inherited Git repository overrides so the selected directory remains authoritative.
func filteredEnv() []string {
	var env []string
	for _, v := range os.Environ() {
		key, _, _ := strings.Cut(v, "=")
		switch key {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_PREFIX", "GIT_EXTERNAL_DIFF", "GIT_DIFF_OPTS", "LC_ALL", "GIT_OPTIONAL_LOCKS", "GIT_TERMINAL_PROMPT":
			continue
		}
		env = append(env, v)
	}
	return env
}
