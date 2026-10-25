package usecases_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/files"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/templates"
)

type fakeWorkspace struct {
	files    []entities.File
	identity entities.Identity
}

func (w *fakeWorkspace) ListFiles(string, bool) ([]entities.File, error) { return w.files, nil }
func (w *fakeWorkspace) Identity(string) entities.Identity               { return w.identity }

func newTemplateInit(paths []entities.File, identity entities.Identity) *usecases.DirectoryTemplateInitInteractor {
	return usecases.NewDirectoryTemplateInitInteractor(
		&fakeWorkspace{files: paths, identity: identity},
		templates.NewDefinitionService(),
		files.NewService(),
	)
}

func TestTemplateInitBuildsDefinition(t *testing.T) {
	dir := t.TempDir()
	interactor := newTemplateInit([]entities.File{
		"/README.md",
		"/Makefile",
		"/cmd/main.go",
		"/cmd/util.go",
	}, entities.Identity{
		Module:     "github.com/acme/sombra-cli",
		Repository: "acme/sombra-cli",
		Project:    "sombra-cli",
		Author:     "Acme",
	})

	result, err := interactor.TemplateInit(dir, entities.TemplateInitOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if result.Definition.Schema != 2 {
		t.Fatalf("schema = %d", result.Definition.Schema)
	}
	base := result.Definition.Parts[entities.BasePart]
	if len(base.Include) != 3 {
		t.Fatalf("base = %#v", base.Include)
	}
	paths := map[string]bool{}
	for _, entry := range base.Include {
		paths[entry.Path] = true
	}
	for _, want := range []string{"/README.md", "/Makefile", "/cmd/"} {
		if !paths[want] {
			t.Fatalf("base missing %s: %#v", want, base.Include)
		}
	}

	if result.Definition.Vars["module"].Default != "github.com/acme/sombra-cli" {
		t.Fatalf("module var = %#v", result.Definition.Vars["module"])
	}
	if len(result.Definition.Rewrites) != 4 {
		t.Fatalf("rewrites = %#v", result.Definition.Rewrites)
	}
	if result.Definition.Rewrites[0].Find != "github.com/acme/sombra-cli" {
		t.Fatalf("first rewrite should be the module: %#v", result.Definition.Rewrites[0])
	}
	if *result.Definition.Rewrites[0].Replace != "{{ .module }}" {
		t.Fatalf("module replacement = %q", *result.Definition.Rewrites[0].Replace)
	}
	if result.Definition.Rewrites[2].As != "word" {
		t.Fatalf("project rewrite should be word based: %#v", result.Definition.Rewrites[2])
	}

	if _, err := os.Stat(filepath.Join(dir, ".sombra", "template.yaml")); err != nil {
		t.Fatalf("definition not written: %v", err)
	}

	loaded, err := templates.NewDefinitionService().Load(result.Path)
	if err != nil {
		t.Fatalf("generated definition does not load: %v", err)
	}
	if len(loaded.Parts[entities.BasePart].Include) != 3 {
		t.Fatalf("loaded base = %#v", loaded.Parts[entities.BasePart])
	}
}

func TestTemplateInitOnlyAndExclude(t *testing.T) {
	dir := t.TempDir()
	interactor := newTemplateInit([]entities.File{
		"/README.md",
		"/Makefile",
		"/cmd/main.go",
		"/cmd/util.go",
	}, entities.Identity{Project: "app"})

	result, err := interactor.TemplateInit(dir, entities.TemplateInitOptions{
		Only:    []string{"/cmd/"},
		Exclude: []string{"*.md"},
	})
	if err != nil {
		t.Fatal(err)
	}
	base := result.Definition.Parts[entities.BasePart]
	if len(base.Include) != 1 || base.Include[0].Path != "/cmd/" {
		t.Fatalf("base = %#v", base.Include)
	}
	if len(result.Definition.Exclude) != 1 || result.Definition.Exclude[0] != "*.md" {
		t.Fatalf("exclude = %#v", result.Definition.Exclude)
	}
}

func TestTemplateInitExactOnly(t *testing.T) {
	dir := t.TempDir()
	interactor := newTemplateInit([]entities.File{
		"/cmd/main.go",
		"/cmd/util.go",
	}, entities.Identity{Project: "app"})

	result, err := interactor.TemplateInit(dir, entities.TemplateInitOptions{Only: []string{"/cmd/main.go"}})
	if err != nil {
		t.Fatal(err)
	}
	base := result.Definition.Parts[entities.BasePart]
	if len(base.Include) != 1 || base.Include[0].Path != "/cmd/main.go" {
		t.Fatalf("base = %#v", base.Include)
	}
}

func TestTemplateInitForceAndDryRun(t *testing.T) {
	dir := t.TempDir()
	interactor := newTemplateInit([]entities.File{"/README.md"}, entities.Identity{Project: "app"})

	first, err := interactor.TemplateInit(dir, entities.TemplateInitOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(first.YAML) == 0 {
		t.Fatal("dry run produced no YAML")
	}
	if _, err := os.Stat(filepath.Join(dir, ".sombra", "template.yaml")); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote a file: %v", err)
	}

	if _, err := interactor.TemplateInit(dir, entities.TemplateInitOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := interactor.TemplateInit(dir, entities.TemplateInitOptions{}); err == nil {
		t.Fatal("expected an overwrite error")
	}
	if _, err := interactor.TemplateInit(dir, entities.TemplateInitOptions{Force: true}); err != nil {
		t.Fatalf("force overwrite: %v", err)
	}
}

func TestTemplateInitWarnsWhenIdentityIsMissing(t *testing.T) {
	dir := t.TempDir()
	interactor := newTemplateInit([]entities.File{"/README.md"}, entities.Identity{Project: "app"})

	result, err := interactor.TemplateInit(dir, entities.TemplateInitOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected identity warnings")
	}
	if len(result.Definition.Rewrites) != 1 {
		t.Fatalf("rewrites = %#v", result.Definition.Rewrites)
	}
}
