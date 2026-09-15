package runtime

import "testing"

func TestContainerResolvesAllRuntimes(t *testing.T) {
	if _, err := NewLocalInitRuntime(); err != nil {
		t.Fatalf("local init: %v", err)
	}
	if _, err := NewLocalUpdateRuntime(); err != nil {
		t.Fatalf("local update: %v", err)
	}
	if _, err := NewTemplateRuntime(); err != nil {
		t.Fatalf("template init: %v", err)
	}
}
