package sombra

import (
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestReplaceRex(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		pattern     string
		replacement string
		expected    string
		wantErr     bool
	}{
		{
			name:        "named group with sprig function",
			content:     "hello world",
			pattern:     `(?P<word>\w+)`,
			replacement: "[[ upper .Match.word ]]",
			expected:    "HELLO WORLD",
		},
		{
			name:        "numbered groups can be reordered",
			content:     "a1b2",
			pattern:     `([a-z])(\d)`,
			replacement: "[[ index .Groups 2 ]][[ index .Groups 1 ]]",
			expected:    "1a2b",
		},
		{
			name:        "whole match is available",
			content:     "x=1",
			pattern:     `\d+`,
			replacement: "[[ .Value ]]0",
			expected:    "x=10",
		},
		{
			name:        "plain replacement without delimiters",
			content:     "a1b2",
			pattern:     `\d`,
			replacement: "#",
			expected:    "a#b#",
		},
		{
			name:        "adjacent matches are rewritten",
			content:     "a1b2",
			pattern:     `([a-z])(\d)`,
			replacement: "[[ index .Groups 2 ]][[ index .Groups 1 ]]",
			expected:    "1a2b",
		},
		{
			name:        "no match returns content unchanged",
			content:     "abc",
			pattern:     `\d`,
			replacement: "#",
			expected:    "abc",
		},
		{
			name:        "invalid pattern errors",
			content:     "abc",
			pattern:     `(`,
			replacement: "#",
			wantErr:     true,
		},
		{
			name:        "invalid template errors",
			content:     "abc1",
			pattern:     `\d`,
			replacement: "[[ nosuchfunc .Value ]]",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replaceRex([]byte(tt.content), tt.pattern, tt.replacement)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("replaceRex() expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("replaceRex() unexpected error: %v", err)
			}
			if string(got) != tt.expected {
				t.Fatalf("replaceRex() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProcessorProcessFileRex(t *testing.T) {
	processor := NewProcessor()

	got, err := processor.ProcessFile(
		[]byte("hi there"),
		entities.MapList{{Key: `rex:(?P<word>\w+)`, Value: "[[ upper .Match.word ]]"}},
		nil,
		true,
	)
	if err != nil {
		t.Fatalf("ProcessFile() unexpected error: %v", err)
	}
	if want := "HI THERE"; string(got) != want {
		t.Fatalf("ProcessFile() = %q, want %q", got, want)
	}
}

func TestProcessorProcessStringRex(t *testing.T) {
	processor := NewProcessor()

	got := processor.ProcessString(
		"src/app",
		entities.MapList{{Key: `rex:(?P<name>\w+)$`, Value: "[[ .Match.name ]]-api"}},
	)
	if want := "src/app-api"; got != want {
		t.Fatalf("ProcessString() = %q, want %q", got, want)
	}
}

func TestRegexNamedGroupsRemainSupported(t *testing.T) {
	processor := NewProcessor()

	// `re:` uses Go's regexp expansion, which already understands ${name}.
	got := processor.ProcessString("hi", entities.MapList{{Key: `re:(?P<word>\w+)`, Value: "${word}!"}})
	if want := "hi!"; got != want {
		t.Fatalf("ProcessString() = %q, want %q", got, want)
	}
}
