package files

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func write(t *testing.T, root, name, content string) string {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return full
}

func TestMatch(t *testing.T) {
	service := NewService()
	cases := []struct {
		path    string
		pattern string
		want    bool
	}{
		{"/a/b_test.go", "**/*_test.go", true},
		{"/README.md", "*.md", true},
		{"/docs/x.md", "*.md", true},
		{"/docs/x.md", "/docs/", true},
		{"/docs/sub/x.md", "/docs/", true},
		{"/other/x.md", "/docs/", false},
		{"/docs/x.md", "docs/*.md", true},
		{"/docs/sub/x.md", "docs/*.md", false},
	}
	for _, tt := range cases {
		got, err := service.Match(tt.path, tt.pattern)
		if err != nil {
			t.Fatalf("Match(%q, %q): %v", tt.path, tt.pattern, err)
		}
		if got != tt.want {
			t.Fatalf("Match(%q, %q) = %v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}

func TestList(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "a")
	write(t, root, "dir/b.txt", "b")
	write(t, root, "dir/sub/c.txt", "c")
	write(t, root, "node_modules/x.js", "x")
	write(t, root, "dir/__pycache__/y.pyc", "y")

	service := NewService()

	files, err := service.List(root, entities.Entry{Path: "/dir/"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("List(/dir/) = %#v", files)
	}

	files, err = service.List(root, entities.Entry{Path: "/a.txt"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Path != "/a.txt" {
		t.Fatalf("List(/a.txt) = %#v", files)
	}

	files, err = service.List(root, entities.Entry{Path: "/"})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.Path == "/node_modules/x.js" || file.Path == "/dir/__pycache__/y.pyc" {
			t.Fatalf("ignored file listed: %s", file.Path)
		}
	}
	if len(files) != 3 {
		t.Fatalf("List(/) = %#v", files)
	}
}

func TestListSkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	target := write(t, root, "target.txt", "x")
	link := filepath.Join(root, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	files, err := NewService().List(root, entities.Entry{Path: "/"})
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.Path == "/link.txt" {
			t.Fatal("symlink was listed")
		}
	}
}

func TestStore(t *testing.T) {
	root := t.TempDir()
	service := NewService()

	if err := service.WriteAtomic(root, "/nested/file.txt", []byte("hello"), 0o755); err != nil {
		t.Fatal(err)
	}
	content, found, err := service.ReadIfExists(root, "/nested/file.txt")
	if err != nil || !found || string(content) != "hello" {
		t.Fatalf("ReadIfExists() = %q, %v, %v", content, found, err)
	}
	info, err := os.Stat(filepath.Join(root, "nested", "file.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %o", info.Mode().Perm())
	}

	_, found, err = service.ReadIfExists(root, "/missing")
	if err != nil || found {
		t.Fatalf("missing: found = %v, err = %v", found, err)
	}

	if err := service.Remove(root, "/nested/file.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "nested")); !os.IsNotExist(err) {
		t.Fatalf("empty directory was not removed: %v", err)
	}
}

func TestReadMissingIsNotFound(t *testing.T) {
	_, err := NewService().Read(t.TempDir(), "/missing")
	if !errors.Is(err, entities.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestHash(t *testing.T) {
	service := NewService()
	left := service.Hash([]byte("hello"))
	right := service.Hash([]byte("hello"))
	if left != right {
		t.Fatalf("hash is not stable: %q != %q", left, right)
	}
	if left == service.Hash([]byte("world")) {
		t.Fatal("different content produced the same hash")
	}
}
