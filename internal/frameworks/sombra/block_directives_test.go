package sombra

import (
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestStripBlockDirectives(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
		wantErr  bool
	}{
		{
			name:     "removes skip block",
			content:  "a\n# sombra:skip\nb\n# sombra:end\nc\n",
			expected: "a\nc\n",
		},
		{
			name:     "nested skip blocks",
			content:  "a\n# sombra:skip\nb\n# sombra:skip\nc\n# sombra:end\nd\n# sombra:end\ne\n",
			expected: "a\ne\n",
		},
		{
			name:     "crlf is preserved for kept lines",
			content:  "a\r\n# sombra:skip\r\nb\r\n# sombra:end\r\nc\r\n",
			expected: "a\r\nc\r\n",
		},
		{
			name:     "content without directives is unchanged",
			content:  "package main\n\nfunc main() {}\n",
			expected: "package main\n\nfunc main() {}\n",
		},
		{
			name:    "missing end reports an error",
			content: "a\n# sombra:skip\nb\n",
			wantErr: true,
		},
		{
			name:    "unexpected end reports an error",
			content: "a\n# sombra:end\nb\n",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stripBlockDirectives([]byte(tt.content))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("stripBlockDirectives() expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("stripBlockDirectives() unexpected error: %v", err)
			}
			if string(got) != tt.expected {
				t.Fatalf("stripBlockDirectives() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProcessorProcessFileBlocks(t *testing.T) {
	processor := NewProcessor()
	content := []byte("start\n# sombra:skip\nrun sombra\n# sombra:end\nend\n")

	got, err := processor.ProcessFile(content, entities.MapList{{Key: "word:sombra", Value: "app"}}, entities.Mappings{}, true)
	if err != nil {
		t.Fatalf("ProcessFile() unexpected error: %v", err)
	}
	if want := "start\nend\n"; string(got) != want {
		t.Fatalf("ProcessFile() = %q, want %q", got, want)
	}
}

func TestProcessorProcessFileBlockDirectivesOptOut(t *testing.T) {
	processor := NewProcessor()
	content := []byte("start\n# sombra:skip\nrun sombra\n# sombra:end\nend\n")

	got, err := processor.ProcessFile(content, entities.MapList{{Key: "word:sombra", Value: "app"}}, entities.Mappings{}, false)
	if err != nil {
		t.Fatalf("ProcessFile() unexpected error: %v", err)
	}
	// With block directives disabled the marker lines are ordinary content and
	// only content mappings apply, so the block is not evaluated.
	if want := "start\n# app:skip\nrun app\n# app:end\nend\n"; string(got) != want {
		t.Fatalf("ProcessFile() = %q, want %q", got, want)
	}
}
