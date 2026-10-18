package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(filteredEnv(), "HOME="+t.TempDir(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com", "GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}
func write(t *testing.T, dir, path string, data []byte) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, data, 0644); err != nil {
		t.Fatal(err)
	}
}
func repo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	d := t.TempDir()
	gitTest(t, d, "init", "-b", "main")
	return d
}
func find(t *testing.T, c ChangeSet, path string) ChangedFile {
	t.Helper()
	for _, f := range c.Files {
		if f.Path == path {
			return f
		}
	}
	t.Fatalf("%q not found: %+v", path, c.Files)
	return ChangedFile{}
}
func TestWorkingTree(t *testing.T) {
	d := repo(t)
	for _, p := range []string{"main.go", "code.py", "Old Name.java", "remove.tsx", "site.js", "style.css"} {
		write(t, d, p, []byte("one\ntwo\nthree\n"))
	}
	write(t, d, ".gitignore", []byte("ignored/\n"))
	gitTest(t, d, "add", ".")
	gitTest(t, d, "commit", "-m", "initial")
	write(t, d, "main.go", []byte("one\ntwo\nthree\nstaged\n"))
	gitTest(t, d, "add", "main.go")
	write(t, d, "main.go", []byte("one\ntwo\nthree\nstaged\nunstaged\n"))
	write(t, d, "code.py", []byte("different\n"))
	gitTest(t, d, "mv", "Old Name.java", "New Name.java")
	if err := os.Remove(filepath.Join(d, "remove.tsx")); err != nil {
		t.Fatal(err)
	}
	write(t, d, "new file.tsx", []byte("hello\nworld\n"))
	write(t, d, "image.bin", []byte{0, 1, 2})
	write(t, d, "ignored/secret.js", []byte("ignored"))
	nested := filepath.Join(d, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	c, err := Analyze(nested)
	if err != nil {
		t.Fatal(err)
	}
	actualRoot, _ := filepath.EvalSymlinks(d)
	if c.Scope != "Working Tree" || c.Repository.Root != actualRoot || c.Repository.Branch != "main" {
		t.Fatalf("unexpected result: %+v", c)
	}
	if c.TotalFiles != 6 {
		t.Fatalf("expected 6 files, got %+v", c.Files)
	}
	f := find(t, c, "main.go")
	if f.Added == nil || *f.Added != 2 || len(f.Hunks) == 0 {
		t.Fatalf("combined diff: %+v", f)
	}
	f = find(t, c, "New Name.java")
	if f.Type != "renamed" || f.PreviousPath != "Old Name.java" {
		t.Fatalf("rename: %+v", f)
	}
	if find(t, c, "remove.tsx").Type != "deleted" {
		t.Fatal("missing deletion")
	}
	f = find(t, c, "new file.tsx")
	if f.Type != "added" || f.Added == nil || *f.Added != 2 || len(f.Hunks) == 0 {
		t.Fatalf("untracked: %+v", f)
	}
	if !find(t, c, "image.bin").Binary {
		t.Fatal("binary not detected")
	}
}
func TestLastCommitAndInitial(t *testing.T) {
	d := repo(t)
	c, err := Analyze(d)
	if err != nil || c.TotalFiles != 0 || c.Scope != "Working Tree" {
		t.Fatalf("empty: %+v %v", c, err)
	}
	write(t, d, "first.go", []byte("hello\n"))
	gitTest(t, d, "add", "first.go")
	c, err = Analyze(d)
	if err != nil || find(t, c, "first.go").Type != "added" {
		t.Fatalf("unborn: %+v %v", c, err)
	}
	gitTest(t, d, "commit", "-m", "first")
	c, err = Analyze(d)
	if err != nil || c.Scope != "Last Commit" || find(t, c, "first.go").Type != "added" {
		t.Fatalf("initial commit: %+v %v", c, err)
	}
	write(t, d, "first.go", []byte("hello\nsecond\n"))
	gitTest(t, d, "add", ".")
	gitTest(t, d, "commit", "-m", "second")
	c, err = Analyze(d)
	if err != nil || c.Scope != "Last Commit" || *find(t, c, "first.go").Added != 1 {
		t.Fatalf("last commit: %+v %v", c, err)
	}
	gitTest(t, d, "checkout", "--detach")
	c, err = Analyze(d)
	if err != nil || !c.Repository.Detached {
		t.Fatalf("detached: %+v %v", c, err)
	}
}
func TestInvalidRepo(t *testing.T) {
	_, err := Analyze(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "Git repository") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestSpecialNamesAndUnavailableGit(t *testing.T) {
	d := repo(t)
	write(t, d, "base.js", []byte("base\n"))
	gitTest(t, d, "add", ".")
	gitTest(t, d, "commit", "-m", "base")
	name := ":odd\tline\n.py"
	write(t, d, name, []byte("text\n"))
	c, err := Analyze(d)
	if err != nil {
		t.Fatal(err)
	}
	f := find(t, c, name)
	if f.Added == nil || *f.Added != 1 {
		t.Fatalf("special filename: %+v", f)
	}
	gitTest(t, d, "add", "--", ":(literal)"+name)
	c, err = Analyze(d)
	if err != nil {
		t.Fatal(err)
	}
	f = find(t, c, name)
	if f.Added == nil || *f.Added != 1 || len(f.Hunks) != 1 {
		t.Fatalf("staged special filename: %+v", f)
	}
	t.Setenv("PATH", t.TempDir())
	_, err = Analyze(d)
	if err == nil || !strings.Contains(err.Error(), "Git is not installed") {
		t.Fatalf("missing Git: %v", err)
	}
}

func TestStagedDeletionWithUntrackedReplacement(t *testing.T) {
	d := repo(t)
	write(t, d, "file.js", []byte("old\n"))
	gitTest(t, d, "add", ".")
	gitTest(t, d, "commit", "-m", "base")
	gitTest(t, d, "rm", "file.js")
	write(t, d, "file.js", []byte("new\n"))
	c, err := Analyze(d)
	if err != nil {
		t.Fatal(err)
	}
	f := find(t, c, "file.js")
	if c.TotalFiles != 1 || f.Type != "modified" || f.Added != nil || f.Details == "" {
		t.Fatalf("replacement: %+v", c)
	}
}
