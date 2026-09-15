package sombra

import (
	"bytes"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestProcessorProcessString(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		mapping  entities.MapList
		expected string
	}{
		{
			name:     "literal replacement",
			content:  "hello sombra-cli",
			mapping:  entities.MapList{{Key: "sombra-cli", Value: "my-app"}},
			expected: "hello my-app",
		},
		{
			name:     "unknown prefix stays literal",
			content:  "see https://example.com now",
			mapping:  entities.MapList{{Key: "https://example.com", Value: "https://new"}},
			expected: "see https://new now",
		},
		{
			name:     "regex replacement keeps captures",
			content:  "cmd/sombra@latest",
			mapping:  entities.MapList{{Key: `re:cmd/(\w+)@latest`, Value: "cmd/$1"}},
			expected: "cmd/sombra",
		},
		{
			name:     "word replaces a standalone token",
			content:  "run sombra now",
			mapping:  entities.MapList{{Key: "word:sombra", Value: "my-app"}},
			expected: "run my-app now",
		},
		{
			name:     "word keeps derived names",
			content:  "sombra-cli sombra.yaml .sombra sombrahq sombra_cli",
			mapping:  entities.MapList{{Key: "word:sombra", Value: "app"}},
			expected: "sombra-cli sombra.yaml .sombra sombrahq sombra_cli",
		},
		{
			name:     "word replaces adjacent tokens",
			content:  "sombra sombra",
			mapping:  entities.MapList{{Key: "word:sombra", Value: "app"}},
			expected: "app app",
		},
		{
			name:     "word matches around punctuation and edges",
			content:  "`sombra` and (sombra)\nsombra",
			mapping:  entities.MapList{{Key: "word:sombra", Value: "app"}},
			expected: "`app` and (app)\napp",
		},
		{
			name:     "word ignores empty payload",
			content:  "unchanged",
			mapping:  entities.MapList{{Key: "word:", Value: "app"}},
			expected: "unchanged",
		},
		{
			name:     "word replacement is literal",
			content:  "run sombra now",
			mapping:  entities.MapList{{Key: "word:sombra", Value: "cost $5"}},
			expected: "run cost $5 now",
		},
		{
			name:     "ci replaces regardless of case",
			content:  "Sombra sombra SOMBRA",
			mapping:  entities.MapList{{Key: "ci:sombra", Value: "app"}},
			expected: "app app app",
		},
		{
			name:     "ci is not token aware",
			content:  "Sombra-cli",
			mapping:  entities.MapList{{Key: "ci:sombra", Value: "app"}},
			expected: "app-cli",
		},
		{
			name:     "word-ci combines both behaviors",
			content:  "Sombra sombra-cli .Sombra",
			mapping:  entities.MapList{{Key: "word-ci:sombra", Value: "app"}},
			expected: "app sombra-cli .Sombra",
		},
		{
			name:     "mappings are applied in order",
			content:  "sombra-cli sombra",
			mapping:  entities.MapList{{Key: "sombra-cli", Value: "x"}, {Key: "word:sombra", Value: "y"}},
			expected: "x y",
		},
		{
			name:     "word covers installer and makefile assignments",
			content:  "go install github.com/org/repo/cmd/sombra@latest\nmake build WHAT=sombra\n",
			mapping:  entities.MapList{{Key: "word:sombra", Value: "my-app"}},
			expected: "go install github.com/org/repo/cmd/my-app@latest\nmake build WHAT=my-app\n",
		},
	}

	processor := NewProcessor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processor.ProcessString(tt.content, tt.mapping); got != tt.expected {
				t.Fatalf("ProcessString(%q) = %q, want %q", tt.content, got, tt.expected)
			}
		})
	}
}

func TestProcessorProcessContent(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		mapping  entities.MapList
		expected []byte
	}{
		{
			name:     "literal replacement",
			content:  []byte("hello sombra-cli"),
			mapping:  entities.MapList{{Key: "sombra-cli", Value: "my-app"}},
			expected: []byte("hello my-app"),
		},
		{
			name:     "regex replacement keeps captures",
			content:  []byte("cmd/sombra@latest"),
			mapping:  entities.MapList{{Key: `re:cmd/(\w+)@latest`, Value: "cmd/$1"}},
			expected: []byte("cmd/sombra"),
		},
		{
			name:     "word replaces a standalone token",
			content:  []byte("run sombra now"),
			mapping:  entities.MapList{{Key: "word:sombra", Value: "my-app"}},
			expected: []byte("run my-app now"),
		},
		{
			name:     "word replacement is literal",
			content:  []byte("run sombra now"),
			mapping:  entities.MapList{{Key: "word:sombra", Value: "cost $5"}},
			expected: []byte("run cost $5 now"),
		},
		{
			name:     "ci replaces regardless of case",
			content:  []byte("Sombra SOMBRA"),
			mapping:  entities.MapList{{Key: "ci:sombra", Value: "app"}},
			expected: []byte("app app"),
		},
	}

	processor := NewProcessor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := processor.ProcessContent(tt.content, tt.mapping); !bytes.Equal(got, tt.expected) {
				t.Fatalf("ProcessContent(%q) = %q, want %q", tt.content, got, tt.expected)
			}
		})
	}
}
