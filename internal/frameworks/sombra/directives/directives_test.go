package directives

import (
	"strings"
	"testing"
)

func TestApplySkip(t *testing.T) {
	input := "a\n# sombra:skip\nb\n# sombra:end\nc\n"
	got, err := NewRegistry().Apply([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "a\nc\n" {
		t.Fatalf("Apply() = %q", got)
	}
}

func TestApplyNestedSkip(t *testing.T) {
	input := "a\n# sombra:skip\nb\n# sombra:skip\nc\n# sombra:end\nd\n# sombra:end\ne\n"
	got, err := NewRegistry().Apply([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "a\ne\n" {
		t.Fatalf("Apply() = %q", got)
	}
}

func TestApplyUnbalanced(t *testing.T) {
	if _, err := NewRegistry().Apply([]byte("a\n# sombra:skip\nb\n")); err == nil {
		t.Fatal("expected an error for an unclosed block")
	}
	if _, err := NewRegistry().Apply([]byte("a\n# sombra:end\n")); err == nil {
		t.Fatal("expected an error for a stray end marker")
	}
}

func TestApplyPreservesUnknownDirectives(t *testing.T) {
	input := "a\n# sombra:unknown\nb\n# sombra:end\nc\n"
	got, err := NewRegistry().Apply([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != input {
		t.Fatalf("Apply() = %q, want %q", got, input)
	}
}

func TestApplyPreservesCRLF(t *testing.T) {
	input := "a\r\n# sombra:skip\r\nb\r\n# sombra:end\r\nc\r\n"
	got, err := NewRegistry().Apply([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "a\r\nc\r\n" {
		t.Fatalf("Apply() = %q", got)
	}
}

func TestApplyWithoutMarkersReturnsInput(t *testing.T) {
	input := "plain text\n"
	got, err := NewRegistry().Apply([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != input {
		t.Fatalf("Apply() = %q", got)
	}
	if !strings.Contains(string(got), "plain") {
		t.Fatal("unexpected content")
	}
}
