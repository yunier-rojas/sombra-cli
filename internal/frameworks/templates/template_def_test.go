package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestRenderReplaceRendersVariables(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".sombra", "snippets"), 0777); err != nil {
		t.Fatal(err)
	}
	fn := filepath.Join(dir, ".sombra", "snippets", "hello.txt")
	if err := os.WriteFile(fn, []byte("Hello, {{ .project }}!"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := NewDefService().RenderReplace(dir, "snippets/hello.txt", entities.Mappings{"project": "my-app"})
	if err != nil {
		t.Fatalf("RenderReplace() unexpected error: %v", err)
	}
	if string(got) != "Hello, my-app!" {
		t.Fatalf("RenderReplace() = %q, want %q", got, "Hello, my-app!")
	}
}

func TestRenderReplaceMissingFile(t *testing.T) {
	_, err := NewDefService().RenderReplace(t.TempDir(), "missing.txt", entities.Mappings{})
	if err == nil {
		t.Fatal("RenderReplace() expected error for a missing file")
	}
}

func TestRenderReplaceRejectsPathEscape(t *testing.T) {
	_, err := NewDefService().RenderReplace(t.TempDir(), "../outside.txt", entities.Mappings{})
	if err == nil {
		t.Fatal("RenderReplace() expected error for a path escaping .sombra")
	}
}
