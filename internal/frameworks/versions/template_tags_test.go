package versions

import (
	"errors"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestGetLatest(t *testing.T) {
	version, err := NewTemplateTagManagerService().GetLatest([]string{"v1.0.0", "v2.0.0"}, "*")
	if err != nil {
		t.Fatal(err)
	}
	if version != entities.Version("v2.0.0") {
		t.Fatalf("version = %q", version)
	}
}

func TestGetLatestNoMatchIsNotFound(t *testing.T) {
	_, err := NewTemplateTagManagerService().GetLatest(nil, "*")
	if !errors.Is(err, entities.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestGetLatestInvalidConstraintIsInvalid(t *testing.T) {
	_, err := NewTemplateTagManagerService().GetLatest([]string{"v1.0.0"}, "[")
	if !errors.Is(err, entities.ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}
