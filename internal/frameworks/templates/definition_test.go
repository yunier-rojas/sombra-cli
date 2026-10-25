package templates

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

func loadDefinition(t *testing.T, body string) (*entities.TemplateDef, error) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".sombra"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, ".sombra", "template.yaml")
	if err := os.WriteFile(file, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return NewDefinitionService().Load(entities.File(file))
}

func TestLoadDefinitionShorthands(t *testing.T) {
	def, err := loadDefinition(t, `
schema: 2
vars:
  project: app
  author:
    description: Author name
parts:
  base:
    include:
      - /Makefile
      - path: /cmd/main.go
        mode: seed
        directives: true
  ci: [/.github/]
  docs:
    description: docs
    include: [/docs/]
exclude: ["**/*_test.go"]
rewrites:
  - find: foo
    replace: bar
    in: [path, name, content]
  - find: x+
    as: regex
    replace: y
  - find: z
    replace: w
    in: all
    files: /Makefile
`)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if def.Schema != 2 {
		t.Fatalf("schema = %d", def.Schema)
	}
	if def.Vars["project"].Default != "app" {
		t.Fatalf("project default = %q", def.Vars["project"].Default)
	}
	if def.Vars["author"].Description != "Author name" {
		t.Fatalf("author description = %q", def.Vars["author"].Description)
	}
	base := def.Parts[entities.BasePart]
	if len(base.Include) != 2 || base.Include[1].Mode != entities.ModeSeed || !base.Include[1].Directives {
		t.Fatalf("base include = %#v", base.Include)
	}
	if len(def.Parts["ci"].Include) != 1 || def.Parts["docs"].Description != "docs" {
		t.Fatalf("parts = %#v", def.Parts)
	}
	if len(def.Rewrites) != 3 || len(def.Rewrites[0].In) != 3 {
		t.Fatalf("rewrites = %#v", def.Rewrites)
	}
	if len(def.Rewrites[2].In) != 1 || def.Rewrites[2].In[0] != "all" {
		t.Fatalf("scalar in = %#v", def.Rewrites[2].In)
	}
	if len(def.Rewrites[2].Files) != 1 || def.Rewrites[2].Files[0] != "/Makefile" {
		t.Fatalf("scalar files = %#v", def.Rewrites[2].Files)
	}
	if def.Rewrites[1].As != "regex" {
		t.Fatalf("as = %q", def.Rewrites[1].As)
	}
}

func TestLoadDefinitionRejectsUnknownFields(t *testing.T) {
	cases := map[string]string{
		"top level":       "schema: 2\nparts:\n  base: [/a]\nbogus: true\n",
		"removed include": "schema: 2\ninclude: [/a]\n",
		"entry":           "schema: 2\nparts:\n  base:\n    - path: /a\n      bogus: true\n",
		"part":            "schema: 2\nparts:\n  base: [/a]\n  ci:\n    bogus: true\n",
		"var":             "schema: 2\nparts:\n  base: [/a]\nvars:\n  project:\n    bogus: true\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := loadDefinition(t, body); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestLoadDefinitionValidation(t *testing.T) {
	cases := map[string]string{
		"schema":          "schema: 1\nparts:\n  base: [/a]\n",
		"missing base":    "schema: 2\nparts:\n  ci: [/a]\n",
		"relative path":   "schema: 2\nparts:\n  base: [a]\n",
		"parent path":     "schema: 2\nparts:\n  base: [/../a]\n",
		"invalid mode":    "schema: 2\nparts:\n  base:\n    - path: /a\n      mode: never\n",
		"empty part":      "schema: 2\nparts:\n  base: [/a]\n  ci: []\n",
		"missing find":    "schema: 2\nparts:\n  base: [/a]\nrewrites:\n  - replace: x\n",
		"missing replace": "schema: 2\nparts:\n  base: [/a]\nrewrites:\n  - find: x\n",
		"invalid regex":   "schema: 2\nparts:\n  base: [/a]\nrewrites:\n  - find: '[a'\n    as: regex\n    replace: x\n",
		"structured path": "schema: 2\nparts:\n  base: [/a]\nrewrites:\n  - find: x\n    as: json\n    replace: y\n    in: [path]\n",
		"invalid scope":   "schema: 2\nparts:\n  base: [/a]\nrewrites:\n  - find: x\n    replace: y\n    in: [everywhere]\n",
		"invalid exclude": "schema: 2\nparts:\n  base: [/a]\nexclude: ['[a']\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := loadDefinition(t, body); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestLoadDefinitionAllowsEmptyBase(t *testing.T) {
	def, err := loadDefinition(t, "schema: 2\nparts:\n  base: []\n  ci: [/.github/]\n")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(def.Parts[entities.BasePart].Include) != 0 {
		t.Fatalf("base = %#v", def.Parts[entities.BasePart])
	}
}

func TestLoadDefinitionReportsLegacyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".sombra"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".sombra", "default.yaml"), []byte("vars: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := NewDefinitionService().Load(NewDefinitionService().GetFile(dir))
	if err == nil || !strings.Contains(err.Error(), "legacy") {
		t.Fatalf("error = %v, want a legacy migration error", err)
	}
	if !errors.Is(err, entities.ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}

func TestLoadDefinitionMissingIsNotFound(t *testing.T) {
	_, err := NewDefinitionService().Load(entities.File(filepath.Join(t.TempDir(), ".sombra", "template.yaml")))
	if !errors.Is(err, entities.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound", err)
	}
}

func TestLoadDefinitionMalformedIsInvalid(t *testing.T) {
	_, err := loadDefinition(t, "schema: [\n")
	if !errors.Is(err, entities.ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
}

func TestRenderValue(t *testing.T) {
	service := NewDefinitionService()

	got, err := service.RenderValue("hello {{ .project | upper }}", entities.Mappings{"project": "app"})
	if err != nil {
		t.Fatalf("RenderValue() error: %v", err)
	}
	if got != "hello APP" {
		t.Fatalf("RenderValue() = %q", got)
	}

	if _, err := service.RenderValue("{{ .missing }}", entities.Mappings{}); err == nil {
		t.Fatal("expected a missing key error")
	}

	got, err = service.RenderValue(`{{ "{{" }} .project }}`, entities.Mappings{})
	if err != nil {
		t.Fatalf("RenderValue() escape error: %v", err)
	}
	if got != "{{ .project }}" {
		t.Fatalf("escape = %q", got)
	}
}

func TestRenderOverlay(t *testing.T) {
	dir := t.TempDir()
	overlay := filepath.Join(dir, ".sombra", "overlay", "docs", "page.md")
	if err := os.MkdirAll(filepath.Dir(overlay), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(overlay, []byte("# {{ .project }}"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := NewDefinitionService()
	content, found, err := service.RenderOverlay(dir, entities.File("/docs/page.md"), entities.Mappings{"project": "app"})
	if err != nil {
		t.Fatalf("RenderOverlay() error: %v", err)
	}
	if !found || string(content) != "# app" {
		t.Fatalf("found = %v, content = %q", found, content)
	}

	_, found, err = service.RenderOverlay(dir, entities.File("/docs/missing.md"), entities.Mappings{})
	if err != nil || found {
		t.Fatalf("missing overlay: found = %v, err = %v", found, err)
	}

	if _, _, err := service.RenderOverlay(dir, entities.File("/../escape.md"), entities.Mappings{}); err == nil {
		t.Fatal("expected a path traversal error")
	}
}
