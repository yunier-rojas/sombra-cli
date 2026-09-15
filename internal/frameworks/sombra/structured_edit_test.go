package sombra

import (
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestPatchJSON(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		path        string
		replacement string
		expected    string
		wantErr     bool
	}{
		{
			name:        "replaces nested value preserving formatting",
			content:     "{\n  \"name\": \"app\",\n  \"scripts\": {\n    \"test\": \"old\"\n  }\n}\n",
			path:        "scripts.test",
			replacement: "go test ./...",
			expected:    "{\n  \"name\": \"app\",\n  \"scripts\": {\n    \"test\": \"go test ./...\"\n  }\n}\n",
		},
		{
			name:        "keeps json types",
			content:     `{"private": false}`,
			path:        "private",
			replacement: "true",
			expected:    `{"private": true}`,
		},
		{
			name:        "replaces an array index",
			content:     `{"jobs":[{"name":"a"},{"name":"b"}]}`,
			path:        "jobs.1.name",
			replacement: "c",
			expected:    `{"jobs":[{"name":"a"},{"name":"c"}]}`,
		},
		{
			name:        "missing path errors",
			content:     `{"a":1}`,
			path:        "b",
			replacement: "2",
			wantErr:     true,
		},
		{
			name:        "invalid array index errors",
			content:     `{"jobs":[1,2]}`,
			path:        "jobs.x",
			replacement: "3",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := patchJSON([]byte(tt.content), tt.path, tt.replacement)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("patchJSON() expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("patchJSON() unexpected error: %v", err)
			}
			if string(got) != tt.expected {
				t.Fatalf("patchJSON() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPatchYAML(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		path        string
		replacement string
		expected    string
		wantErr     bool
	}{
		{
			name:        "replaces a value and keeps comments",
			content:     "# top comment\nname: app\nversion: 1.0.0 # inline\n",
			path:        "version",
			replacement: "2.0.0",
			expected:    "# top comment\nname: app\nversion: 2.0.0 # inline\n",
		},
		{
			name:        "replaces a nested sequence value",
			content:     "jobs:\n  - name: a\n  - name: b\n",
			path:        "jobs.1.name",
			replacement: "c",
			expected:    "jobs:\n  - name: a\n  - name: c\n",
		},
		{
			name:        "missing path errors",
			content:     "name: app\n",
			path:        "missing",
			replacement: "x",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := patchYAML([]byte(tt.content), tt.path, tt.replacement)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("patchYAML() expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("patchYAML() unexpected error: %v", err)
			}
			if string(got) != tt.expected {
				t.Fatalf("patchYAML() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPatchINI(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		path        string
		replacement string
		expected    string
		wantErr     bool
	}{
		{
			name:        "replaces a key in a section",
			content:     "[core]\n\teditor = vim\n\tautocrlf = true\n\n[user]\n\tname = dev\n",
			path:        "core.editor",
			replacement: "nano",
			expected:    "[core]\n\teditor = nano\n\tautocrlf = true\n\n[user]\n\tname = dev\n",
		},
		{
			name:        "supports colon separators",
			content:     "[core]\neditor: vim\n",
			path:        "core.editor",
			replacement: "nano",
			expected:    "[core]\neditor: nano\n",
		},
		{
			name:        "supports global keys",
			content:     "editor = vim\n",
			path:        "editor",
			replacement: "nano",
			expected:    "editor = nano\n",
		},
		{
			name:        "missing path errors",
			content:     "[core]\neditor = vim\n",
			path:        "core.missing",
			replacement: "x",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := patchINI([]byte(tt.content), tt.path, tt.replacement)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("patchINI() expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("patchINI() unexpected error: %v", err)
			}
			if string(got) != tt.expected {
				t.Fatalf("patchINI() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestProcessorProcessFileStructured(t *testing.T) {
	processor := NewProcessor()

	content := []byte("{\n  \"scripts\": {\n    \"test\": \"old\"\n  }\n}\n")
	got, err := processor.ProcessFile(content, entities.MapList{{Key: "json:scripts.test", Value: "new"}}, nil, true)
	if err != nil {
		t.Fatalf("ProcessFile() unexpected error: %v", err)
	}
	if want := "{\n  \"scripts\": {\n    \"test\": \"new\"\n  }\n}\n"; string(got) != want {
		t.Fatalf("ProcessFile() = %q, want %q", got, want)
	}
}

func TestProcessContentSkipsStructuredDirectives(t *testing.T) {
	processor := NewProcessor()

	// Diff fragments cannot be parsed as documents, so string processing must
	// leave structured directives untouched.
	content := []byte("+  \"test\": \"old\"")
	got := processor.ProcessContent(content, entities.MapList{{Key: "json:scripts.test", Value: "new"}})
	if string(got) != string(content) {
		t.Fatalf("ProcessContent() = %q, want %q", got, content)
	}
}
