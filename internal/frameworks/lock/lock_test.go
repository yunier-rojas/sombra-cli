package lock

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestLoadMissingReturnsEmptyLock(t *testing.T) {
	lock, err := NewService().Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if lock == nil || lock.Templates == nil {
		t.Fatalf("lock = %#v", lock)
	}
}

func TestSaveAndLoad(t *testing.T) {
	target := t.TempDir()
	service := NewService()
	original := &entities.Lock{
		Templates: map[string]entities.LockedTemplate{
			"app": {
				Version: "v1.0.0",
				Parts:   []string{"ci"},
				Files: map[string]entities.LockedFile{
					"/Makefile": {Hash: "sha256:abc", Mode: entities.ModeSync, Source: "/Makefile"},
				},
			},
		},
	}
	if err := service.Save(target, original); err != nil {
		t.Fatal(err)
	}

	loaded, err := service.Load(target)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != 1 {
		t.Fatalf("version = %d", loaded.Version)
	}
	if loaded.Templates["app"].Files["/Makefile"].Hash != "sha256:abc" {
		t.Fatalf("lock = %#v", loaded)
	}
}

func TestLoadRejectsUnknownFields(t *testing.T) {
	target := t.TempDir()
	path := filepath.Join(target, ".sombra", "lock.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("lock: 1\nbogus: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService().Load(target); !errors.Is(err, entities.ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}

func TestLoadRejectsCorruptYAML(t *testing.T) {
	target := t.TempDir()
	path := filepath.Join(target, ".sombra", "lock.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("lock: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewService().Load(target); !errors.Is(err, entities.ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}
