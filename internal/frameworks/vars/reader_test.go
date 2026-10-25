package vars

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func TestGetValuesUsesDefaults(t *testing.T) {
	out := &bytes.Buffer{}
	reader := &FileReader{input: strings.NewReader("\nAcme\n"), out: out}

	values, err := reader.GetValues([]entities.VarSpec{
		{Name: "project", Description: "Project name", Default: "app"},
		{Name: "author"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if values["project"] != "app" || values["author"] != "Acme" {
		t.Fatalf("values = %#v", values)
	}
	if !strings.Contains(out.String(), "Project name") || !strings.Contains(out.String(), "[app]") {
		t.Fatalf("prompt = %q", out.String())
	}
}

func TestGetValuesRequiresValue(t *testing.T) {
	reader := &FileReader{input: strings.NewReader("\n"), out: &bytes.Buffer{}}
	if _, err := reader.GetValues([]entities.VarSpec{{Name: "author"}}); err == nil {
		t.Fatal("expected an error")
	}
}
