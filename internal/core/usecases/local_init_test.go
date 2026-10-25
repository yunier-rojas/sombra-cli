package usecases_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/sombra"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/templates"
)

type fakeVars struct {
	seen []entities.VarSpec
}

func (v *fakeVars) GetValues(specs []entities.VarSpec) (entities.Mappings, error) {
	v.seen = specs
	values := make(entities.Mappings, len(specs))
	for _, spec := range specs {
		values[spec.Name] = "value-" + spec.Name
	}
	return values, nil
}

func TestLocalInitStoresPartsAndVars(t *testing.T) {
	templateDir := t.TempDir()
	writeFile(t, templateDir, ".sombra/template.yaml", `
schema: 2
vars:
  project: app
  author:
    description: Author
parts:
  base:
    include:
      - /README.md
  ci:
    include: [/.github/]
`)
	target := t.TempDir()
	reader := &fakeVars{}
	interactor := usecases.NewLocalInitInteractor(
		&fakePrepare{repo: &fakeRepo{dir: templateDir}},
		templates.NewDefinitionService(),
		sombra.NewDefService(),
		reader,
	)

	if err := interactor.LocalInit(target, "https://example.com/template.git", "", []string{"ci"}); err != nil {
		t.Fatal(err)
	}
	if len(reader.seen) != 2 || reader.seen[0].Name != "author" || reader.seen[1].Name != "project" {
		t.Fatalf("vars = %#v", reader.seen)
	}

	data, err := os.ReadFile(filepath.Join(target, "sombra.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "parts:") || !strings.Contains(content, "ci") {
		t.Fatalf("sombra.yaml = %s", content)
	}
	if !strings.Contains(content, "value-project") {
		t.Fatalf("vars missing: %s", content)
	}
}

func TestLocalInitDefaultsToBase(t *testing.T) {
	templateDir := t.TempDir()
	writeFile(t, templateDir, ".sombra/template.yaml", `
schema: 2
parts:
  base: [/README.md]
  ci: [/.github/]
`)
	target := t.TempDir()
	interactor := usecases.NewLocalInitInteractor(
		&fakePrepare{repo: &fakeRepo{dir: templateDir}},
		templates.NewDefinitionService(),
		sombra.NewDefService(),
		&fakeVars{},
	)

	if err := interactor.LocalInit(target, "https://example.com/template.git", "", nil); err != nil {
		t.Fatal(err)
	}

	def, err := sombra.NewDefService().Load(entities.File(filepath.Join(target, "sombra.yaml")))
	if err != nil {
		t.Fatal(err)
	}
	if len(def.Templates) != 1 || len(def.Templates[0].Parts) != 1 || def.Templates[0].Parts[0] != entities.BasePart {
		t.Fatalf("parts = %#v", def.Templates)
	}
}

func TestLocalInitRejectsUnknownPart(t *testing.T) {
	templateDir := t.TempDir()
	writeFile(t, templateDir, ".sombra/template.yaml", `
schema: 2
parts:
  base: [/README.md]
`)
	target := t.TempDir()
	interactor := usecases.NewLocalInitInteractor(
		&fakePrepare{repo: &fakeRepo{dir: templateDir}},
		templates.NewDefinitionService(),
		sombra.NewDefService(),
		&fakeVars{},
	)
	if err := interactor.LocalInit(target, "https://example.com/template.git", "", []string{"nope"}); err == nil {
		t.Fatal("expected an unknown part error")
	}
}
