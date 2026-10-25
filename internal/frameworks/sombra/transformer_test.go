package sombra

import (
	"strings"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func rule(find, replace string) entities.Rewrite {
	return entities.Rewrite{Find: find, Replace: &replace}
}

func TestTransformerTextOrder(t *testing.T) {
	transformer := NewTransformer()
	rules := []entities.Rewrite{
		rule("a", "b"),
		rule("b", "c"),
	}
	got, err := transformer.Text("a", rules)
	if err != nil {
		t.Fatal(err)
	}
	if got != "c" {
		t.Fatalf("Text() = %q, want c", got)
	}
}

func TestTransformerWordBoundaries(t *testing.T) {
	transformer := NewTransformer()
	word := rule("sombra", "app")
	word.As = "word"

	got, err := transformer.Text("run sombra now, sombra-cli and sombra.yaml", []entities.Rewrite{word})
	if err != nil {
		t.Fatal(err)
	}
	want := "run app now, sombra-cli and sombra.yaml"
	if got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
}

func TestTransformerWordIgnoreCase(t *testing.T) {
	transformer := NewTransformer()
	word := rule("sombra", "app")
	word.As = "word"
	word.IgnoreCase = true

	got, err := transformer.Text("Sombra and SOMBRA", []entities.Rewrite{word})
	if err != nil {
		t.Fatal(err)
	}
	if got != "app and app" {
		t.Fatalf("Text() = %q", got)
	}
}

func TestTransformerRegexCaptures(t *testing.T) {
	transformer := NewTransformer()
	regex := rule(`cmd/(\w+)@latest`, "cmd/$1")
	regex.As = "regex"

	got, err := transformer.Text("go install cmd/sombra@latest", []entities.Rewrite{regex})
	if err != nil {
		t.Fatal(err)
	}
	if got != "go install cmd/sombra" {
		t.Fatalf("Text() = %q", got)
	}
}

func TestTransformerStructuredJSON(t *testing.T) {
	transformer := NewTransformer()
	jsonRule := rule("scripts.test", "go test ./...")
	jsonRule.As = "json"

	content := []byte("{\n  \"scripts\": {\n    \"test\": \"old\"\n  }\n}\n")
	got, err := transformer.Content(content, []entities.Rewrite{jsonRule}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "\"test\": \"go test ./...\"") {
		t.Fatalf("Content() = %s", got)
	}
}

func TestTransformerLeavesBinaryFiles(t *testing.T) {
	transformer := NewTransformer()
	content := append([]byte("prefix"), 0, 'b', 'i', 'n')
	got, err := transformer.Content(content, []entities.Rewrite{rule("prefix", "changed")}, true)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("binary content changed: %q", got)
	}
}

func TestTransformerDirectives(t *testing.T) {
	transformer := NewTransformer()
	content := []byte("keep\n# sombra:skip\nremove\n# sombra:end\nkeep\n")

	got, err := transformer.Content(content, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "keep\nkeep\n" {
		t.Fatalf("Content() = %q", got)
	}

	got, err = transformer.Content(content, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatalf("directives applied when disabled: %q", got)
	}
}
