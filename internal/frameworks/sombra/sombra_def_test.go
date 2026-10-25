package sombra

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestDefServiceRoundTrip(t *testing.T) {
	target := t.TempDir()
	service := NewDefService()

	original := &entities.SombraDef{Templates: []*entities.TemplateConfig{{
		ID:    "app",
		URI:   "https://example.com/app.git",
		Parts: []string{"ci"},
		Vars:  entities.Mappings{"project": "app"},
	}}}
	if err := service.Save(service.GetFile(target), original); err != nil {
		t.Fatal(err)
	}

	loaded, err := service.Load(service.GetFile(target))
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Templates) != 1 || loaded.Templates[0].ID != "app" {
		t.Fatalf("loaded = %#v", loaded)
	}
	if len(loaded.Templates[0].Parts) != 1 || loaded.Templates[0].Parts[0] != "ci" {
		t.Fatalf("parts = %#v", loaded.Templates[0].Parts)
	}
}

func TestDefServiceMissingFile(t *testing.T) {
	loaded, err := NewDefService().Load(entities.File(filepath.Join(t.TempDir(), "sombra.yaml")))
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.Templates == nil {
		t.Fatalf("loaded = %#v", loaded)
	}
}

func TestDefServiceRejectsUnknownFields(t *testing.T) {
	target := t.TempDir()
	path := filepath.Join(target, "sombra.yaml")
	if err := os.WriteFile(path, []byte("templates:\n  - uri: x\n    current: v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewDefService().Load(entities.File(path)); err == nil {
		t.Fatal("expected an error for the removed current field")
	}
}
