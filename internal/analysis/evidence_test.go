package analysis

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEvidenceCollectionAndSecrets(t *testing.T) {
	d := repo(t)
	write(t, d, "main.go", []byte("package main\nfunc old() {}\n"))
	write(t, d, "deleted.java", []byte("class Old {}\n"))
	write(t, d, "rename.css", []byte(".x { color: red; }\n"))
	gitTest(t, d, "add", ".")
	gitTest(t, d, "commit", "-m", "initial")
	write(t, d, "main.go", []byte("package main\nfunc next() {}\n"))
	os.Remove(filepath.Join(d, "deleted.java"))
	gitTest(t, d, "mv", "rename.css", "new.css")
	write(t, d, "new.py", []byte("print('hello')\n"))
	write(t, d, "view.tsx", []byte("export const x = 1;\n"))
	write(t, d, ".env", []byte("TOKEN=value\n"))
	write(t, d, "leak.js", []byte("const api_key = 'abcdefghijk';\n"))
	write(t, d, "large.js", []byte(strings.Repeat("x", MaxEvidenceFileBytes+1)))
	write(t, d, "image.bin", []byte{0, 1, 2})
	outside := filepath.Join(t.TempDir(), "outside")
	os.WriteFile(outside, []byte("outside secret"), 0600)
	if err := os.Symlink(outside, filepath.Join(d, "link.js")); err != nil {
		t.Fatal(err)
	}
	ev, err := Collect(d)
	if err != nil {
		t.Fatal(err)
	}
	all := map[string]EvidenceFile{}
	for _, f := range ev.Files {
		all[f.Path] = f
	}
	for _, p := range []string{"main.go", "deleted.java", "new.py", "view.tsx"} {
		f, ok := all[p]
		if !ok || len(f.Lines) == 0 {
			t.Fatalf("missing %q evidence: %+v skipped %+v", p, ev.Files, ev.Skipped)
		}
	}
	if strings.Contains(all["new.py"].Excerpt, d) {
		t.Fatal("untracked diff exposed an absolute local path")
	}
	if !strings.Contains(all["main.go"].Excerpt, "+func next") || !strings.Contains(all["deleted.java"].Excerpt, "-class Old") {
		t.Fatal("missing diff content")
	}
	for _, p := range []string{".env", "leak.js", "large.js", "image.bin", "link.js"} {
		if _, ok := all[p]; ok {
			t.Fatalf("sensitive/unsafe %s included", p)
		}
	}
	if len(ev.Skipped) < 5 {
		t.Fatalf("skips missing: %+v", ev.Skipped)
	}
	if _, err = safePath(d, "../outside"); err == nil {
		t.Fatal("path traversal accepted")
	}
	if _, err = safePath(d, "link.js"); err == nil {
		t.Fatal("symlink accepted")
	}
}
func TestEvidenceLimitsAndLastCommit(t *testing.T) {
	d := repo(t)
	write(t, d, "first.rs", []byte("fn old() {}\n"))
	gitTest(t, d, "add", ".")
	gitTest(t, d, "commit", "-m", "one")
	write(t, d, "first.rs", []byte("fn next() {}\n"))
	gitTest(t, d, "add", ".")
	gitTest(t, d, "commit", "-m", "two")
	ev, err := Collect(d)
	if err != nil || ev.Scope != "Last Commit" || len(ev.Files) != 1 || !strings.Contains(ev.Files[0].Excerpt, "fn next") {
		t.Fatalf("last commit: %+v %v", ev, err)
	}
	for i := 0; i < 25; i++ {
		write(t, d, strings.Repeat("x", i+1)+".php", []byte("hello\n"))
	}
	ev, err = Collect(d)
	if err != nil || len(ev.Files) != MaxEvidenceFiles || len(ev.Skipped) < 5 {
		t.Fatalf("limits: %d %d %v", len(ev.Files), len(ev.Skipped), err)
	}
}
func TestGitFiltersNotExecuted(t *testing.T) {
	d := repo(t)
	marker := filepath.Join(d, "executed")
	gitTest(t, d, "config", "filter.evil.clean", "touch "+marker)
	gitTest(t, d, "config", "filter.evil.required", "true")
	write(t, d, ".gitattributes", []byte("*.go filter=evil\n"))
	write(t, d, "x.go", []byte("code\n"))
	_, err := Collect(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("Git filter ran")
	}
}

func TestEvidenceSkipsOversizedDiffAndSecretOutsideHunk(t *testing.T) {
	d := repo(t)
	write(t, d, "large.go", []byte(strings.Repeat("line\n", 10000)))
	write(t, d, "config.go", []byte("const api_key = 'abcdefghijk'\n"+strings.Repeat("ordinary\n", 20)+"old\n"))
	write(t, d, "clean.go", []byte("old\n"))
	gitTest(t, d, "add", ".")
	gitTest(t, d, "commit", "-m", "base")
	os.Remove(filepath.Join(d, "large.go"))
	write(t, d, "config.go", []byte("const api_key = 'abcdefghijk'\n"+strings.Repeat("ordinary\n", 20)+"new\n"))
	write(t, d, "clean.go", []byte("new\n"))
	ev, err := Collect(d)
	if err != nil {
		t.Fatal(err)
	}
	if len(ev.Files) != 1 || ev.Files[0].Path != "clean.go" {
		t.Fatalf("evidence leaked or valid file skipped: %+v", ev)
	}
	if strings.Contains(ev.Files[0].Excerpt, "api_key") || len(ev.Skipped) != 2 {
		t.Fatalf("unexpected evidence %+v", ev)
	}
}

func TestImportContext(t *testing.T) {
	d := repo(t)
	write(t, d, "main.py", []byte("import os\n"+strings.Repeat("# filler\n", 8)+"old = 1\n"))
	gitTest(t, d, "add", ".")
	gitTest(t, d, "commit", "-m", "base")
	write(t, d, "main.py", []byte("import os\n"+strings.Repeat("# filler\n", 8)+"new = 1\n"))
	ev, err := Collect(d)
	if err != nil || len(ev.Files) != 1 {
		t.Fatalf("collect: %+v %v", ev, err)
	}
	if !strings.Contains(ev.Files[0].Excerpt, "1: import os") {
		t.Fatalf("import absent: %s", ev.Files[0].Excerpt)
	}
	found := false
	for _, l := range ev.Files[0].Lines {
		if l.Side == "new" && l.Number == 1 && l.Text == "import os" {
			found = true
		}
	}
	if !found {
		t.Fatal("import reference missing")
	}
}
