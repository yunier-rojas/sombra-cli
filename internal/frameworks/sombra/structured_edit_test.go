package sombra

import (
	"strings"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func structured(find, as, replace string) entities.Rewrite {
	return entities.Rewrite{Find: find, As: as, Replace: &replace}
}

func TestStructuredJSON(t *testing.T) {
	transformer := NewTransformer()
	content := []byte("{\n  \"scripts\": {\n    \"test\": \"old\"\n  }\n}\n")

	got, err := transformer.Content(content, []entities.Rewrite{structured("scripts.test", "json", "new")}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "\"test\": \"new\"") {
		t.Fatalf("Content() = %s", got)
	}
}

func TestStructuredYAML(t *testing.T) {
	transformer := NewTransformer()
	content := []byte("jobs:\n  build:\n    runs-on: ubuntu-20.04\n")

	got, err := transformer.Content(content, []entities.Rewrite{structured("jobs.build.runs-on", "yaml", "ubuntu-latest")}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "ubuntu-latest") {
		t.Fatalf("Content() = %s", got)
	}
}

func TestStructuredINI(t *testing.T) {
	transformer := NewTransformer()
	content := []byte("[core]\neditor = vim\n")

	got, err := transformer.Content(content, []entities.Rewrite{structured("core.editor", "ini", "nano")}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "editor = nano") {
		t.Fatalf("Content() = %s", got)
	}
}

func TestStructuredMissingPath(t *testing.T) {
	transformer := NewTransformer()
	if _, err := transformer.Content([]byte("{}"), []entities.Rewrite{structured("missing.path", "json", "x")}, false); err == nil {
		t.Fatal("expected a missing path error")
	}
}
