package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
)

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	dir := t.TempDir()
	run(t, dir, "init", "-q")
	run(t, dir, "config", "user.email", "test@example.com")
	run(t, dir, "config", "user.name", "Test User")
	run(t, dir, "remote", "add", "origin", "git@github.com:acme/sombra-cli.git")
	return dir
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListFilesGit(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "go.mod", "module github.com/acme/sombra-cli\n\ngo 1.26.0\n")
	write(t, dir, "README.md", "# readme\n")
	write(t, dir, ".gitignore", "ignored.txt\n")
	write(t, dir, "ignored.txt", "ignored\n")
	write(t, dir, "untracked.txt", "untracked\n")
	write(t, dir, ".sombra/template.yaml", "schema: 2\n")
	run(t, dir, "add", "go.mod", "README.md", ".gitignore")
	run(t, dir, "commit", "-qm", "init")

	files, err := NewService().ListFiles(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i] < files[j] })
	want := []string{"/.gitignore", "/README.md", "/go.mod"}
	if len(files) != len(want) {
		t.Fatalf("ListFiles() = %#v, want %#v", files, want)
	}
	for index, file := range files {
		if string(file) != want[index] {
			t.Fatalf("ListFiles()[%d] = %s, want %s", index, file, want[index])
		}
	}

	files, err = NewService().ListFiles(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, file := range files {
		if file == "/untracked.txt" {
			found = true
		}
	}
	if !found {
		t.Fatalf("untracked file missing: %#v", files)
	}
}

func TestListFilesFallback(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.txt", "a")
	write(t, dir, "node_modules/x.js", "x")
	write(t, dir, "sombra.yaml", "templates: []\n")

	files, err := NewService().ListFiles(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "/a.txt" {
		t.Fatalf("ListFiles() = %#v", files)
	}
}

func TestIdentity(t *testing.T) {
	dir := initRepo(t)
	write(t, dir, "go.mod", "module github.com/acme/sombra-cli\n\ngo 1.26.0\n")

	identity := NewService().Identity(dir)
	if identity.Module != "github.com/acme/sombra-cli" {
		t.Fatalf("module = %q", identity.Module)
	}
	if identity.Repository != "acme/sombra-cli" {
		t.Fatalf("repository = %q", identity.Repository)
	}
	if identity.Project != "sombra-cli" {
		t.Fatalf("project = %q", identity.Project)
	}
	if identity.Author != "Test User" {
		t.Fatalf("author = %q", identity.Author)
	}
}

func TestIdentityWithoutGit(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "README.md", "# readme\n")

	identity := NewService().Identity(dir)
	if identity.Module != "" || identity.Repository != "" {
		t.Fatalf("identity = %#v", identity)
	}
	if identity.Project != filepath.Base(dir) {
		t.Fatalf("project = %q", identity.Project)
	}
}

func TestNormalizeRemote(t *testing.T) {
	cases := map[string]string{
		"git@github.com:acme/repo.git":          "acme/repo",
		"https://github.com/acme/repo.git":      "acme/repo",
		"ssh://git@github.com/acme/repo.git":    "acme/repo",
		"https://gitlab.com/group/sub/repo.git": "group/sub/repo",
		"":                                      "",
	}
	for input, want := range cases {
		if got := normalizeRemote(input); got != want {
			t.Fatalf("normalizeRemote(%q) = %q, want %q", input, got, want)
		}
	}
}
