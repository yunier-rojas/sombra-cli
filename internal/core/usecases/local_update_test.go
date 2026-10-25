package usecases_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/files"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/lock"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/sombra"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/templates"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/versions"
)

type fakeRepo struct{ dir string }

func (r *fakeRepo) Clone() error               { return nil }
func (r *fakeRepo) Clean() error               { return nil }
func (r *fakeRepo) Dir() string                { return r.dir }
func (r *fakeRepo) Use(string) (string, error) { return "", nil }
func (r *fakeRepo) GetTags() ([]string, error) { return []string{"v1.0.0"}, nil }

type fakePrepare struct {
	repo  *fakeRepo
	repos map[string]*fakeRepo
}

func (p *fakePrepare) Prepare(uri, _ string) (usecases.RepositoryPort, error) {
	if repo, ok := p.repos[uri]; ok {
		return repo, nil
	}
	return p.repo, nil
}

func newInteractor(t *testing.T, templateDir string) *usecases.LocalUpdateInteractor {
	t.Helper()
	return newInteractorWithPrepare(t, &fakePrepare{repo: &fakeRepo{dir: templateDir}})
}

func newInteractorWithPrepare(t *testing.T, prepare *fakePrepare) *usecases.LocalUpdateInteractor {
	t.Helper()
	fileService := files.NewService()
	return usecases.NewLocalUpdateInteractor(
		prepare,
		templates.NewDefinitionService(),
		sombra.NewDefService(),
		versions.NewTemplateTagManagerService(),
		fileService,
		fileService,
		fileService,
		lock.NewService(),
		fileService,
		sombra.NewTransformer(),
	)
}

func writeFile(t *testing.T, root, name, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeTarget(t *testing.T, target, uri string, parts []string) {
	t.Helper()
	body := "templates:\n  - id: app\n    uri: " + uri + "\n    vars:\n      project: app\n"
	if len(parts) > 0 {
		body = "templates:\n  - id: app\n    uri: " + uri + "\n    parts: [" + strings.Join(parts, ", ") + "]\n    vars:\n      project: app\n"
	}
	writeFile(t, target, "sombra.yaml", body)
}

func setupTemplate(t *testing.T) string {
	t.Helper()
	templateDir := t.TempDir()
	writeFile(t, templateDir, ".sombra/template.yaml", `
schema: 2
parts:
  base:
    include:
      - /Makefile
      - path: /cmd/sombra/main.go
        mode: seed
        directives: true
  docs:
    include: [/docs/]
rewrites:
  - find: sombra-cli
    replace: "{{ .project }}"
    in: [path, name, content]
  - find: sombra
    as: word
    replace: "{{ .project }}"
    in: [path, name, content]
`)
	writeFile(t, templateDir, "Makefile", "BINARY=sombra-cli\n")
	writeFile(t, templateDir, "cmd/sombra/main.go", "package main\n// sombra:skip\nsecret\n// sombra:end\n// sombra-cli\n")
	writeFile(t, templateDir, "docs/readme.md", "# sombra-cli docs\n")
	return templateDir
}

func TestLocalUpdateCreatesAndRenames(t *testing.T) {
	templateDir := setupTemplate(t)
	target := t.TempDir()
	writeTarget(t, target, "https://example.com/template.git", nil)

	interactor := newInteractor(t, templateDir)
	result, err := interactor.LocalUpdate(target, "app", entities.LocalUpdateOptions{Tag: "v1.0.0", PartsSet: true})
	if err != nil {
		t.Fatalf("LocalUpdate() error: %v", err)
	}
	if len(result.Created) != 2 {
		t.Fatalf("created = %#v", result.Created)
	}

	makefile, err := os.ReadFile(filepath.Join(target, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if string(makefile) != "BINARY=app\n" {
		t.Fatalf("Makefile = %q", makefile)
	}

	mainFile, err := os.ReadFile(filepath.Join(target, "cmd/app/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(mainFile) != "package main\n// app\n" {
		t.Fatalf("main.go = %q", mainFile)
	}

	if _, err := os.Stat(filepath.Join(target, ".sombra/lock.yaml")); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}
}

func TestLocalUpdateConflictAndForce(t *testing.T) {
	templateDir := setupTemplate(t)
	target := t.TempDir()
	writeTarget(t, target, "https://example.com/template.git", nil)

	interactor := newInteractor(t, templateDir)
	opts := entities.LocalUpdateOptions{Tag: "v1.0.0", PartsSet: true}
	if _, err := interactor.LocalUpdate(target, "app", opts); err != nil {
		t.Fatal(err)
	}

	writeFile(t, target, "Makefile", "BINARY=local\n")
	result, err := interactor.LocalUpdate(target, "app", opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) != 1 || result.Conflicts[0].Path != "Makefile" {
		t.Fatalf("conflicts = %#v", result.Conflicts)
	}

	forced := opts
	forced.Force = true
	if _, err := interactor.LocalUpdate(target, "app", forced); err != nil {
		t.Fatal(err)
	}
	makefile, err := os.ReadFile(filepath.Join(target, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	if string(makefile) != "BINARY=app\n" {
		t.Fatalf("Makefile after force = %q", makefile)
	}
}

func TestLocalUpdateSeedKeepsEdits(t *testing.T) {
	templateDir := setupTemplate(t)
	target := t.TempDir()
	writeTarget(t, target, "https://example.com/template.git", nil)

	interactor := newInteractor(t, templateDir)
	opts := entities.LocalUpdateOptions{Tag: "v1.0.0", PartsSet: true}
	if _, err := interactor.LocalUpdate(target, "app", opts); err != nil {
		t.Fatal(err)
	}

	writeFile(t, target, "cmd/app/main.go", "package main\n// local edit\n")
	result, err := interactor.LocalUpdate(target, "app", opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) != 0 {
		t.Fatalf("conflicts = %#v", result.Conflicts)
	}
	if len(result.Kept) != 1 || result.Kept[0] != "cmd/app/main.go" {
		t.Fatalf("kept = %#v", result.Kept)
	}
	content, err := os.ReadFile(filepath.Join(target, "cmd/app/main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "package main\n// local edit\n" {
		t.Fatalf("seed file was overwritten: %q", content)
	}
}

func TestLocalUpdatePartsAndPrune(t *testing.T) {
	templateDir := setupTemplate(t)
	target := t.TempDir()
	writeTarget(t, target, "https://example.com/template.git", nil)

	interactor := newInteractor(t, templateDir)
	opts := entities.LocalUpdateOptions{Tag: "v1.0.0", PartsSet: true, Parts: []string{"base", "docs"}}
	result, err := interactor.LocalUpdate(target, "app", opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 3 {
		t.Fatalf("created = %#v", result.Created)
	}
	if _, err := os.Stat(filepath.Join(target, "docs/readme.md")); err != nil {
		t.Fatal(err)
	}

	withoutDocs := entities.LocalUpdateOptions{Tag: "v1.0.0", PartsSet: true}
	result, err = interactor.LocalUpdate(target, "app", withoutDocs)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Prunable) != 1 || result.Prunable[0] != "docs/readme.md" {
		t.Fatalf("prunable = %#v", result.Prunable)
	}

	pruned := withoutDocs
	pruned.Prune = true
	result, err = interactor.LocalUpdate(target, "app", pruned)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Removed) != 1 || result.Removed[0] != "docs/readme.md" {
		t.Fatalf("removed = %#v", result.Removed)
	}
	if _, err := os.Stat(filepath.Join(target, "docs")); !os.IsNotExist(err) {
		t.Fatalf("docs directory was not removed: %v", err)
	}
}

func TestLocalUpdatePartOmitsBase(t *testing.T) {
	templateDir := setupTemplate(t)
	target := t.TempDir()
	writeTarget(t, target, "https://example.com/template.git", nil)

	interactor := newInteractor(t, templateDir)
	result, err := interactor.LocalUpdate(target, "app", entities.LocalUpdateOptions{
		Tag:      "v1.0.0",
		PartsSet: true,
		Parts:    []string{"docs"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 1 || result.Created[0] != "docs/readme.md" {
		t.Fatalf("created = %#v", result.Created)
	}
	if _, err := os.Stat(filepath.Join(target, "Makefile")); !os.IsNotExist(err) {
		t.Fatalf("base file was copied: %v", err)
	}
}

func TestLocalUpdateUsesStoredSelectionOrDefaultBase(t *testing.T) {
	templateDir := setupTemplate(t)

	t.Run("missing parts defaults to base", func(t *testing.T) {
		target := t.TempDir()
		writeTarget(t, target, "https://example.com/template.git", nil)

		interactor := newInteractor(t, templateDir)
		result, err := interactor.LocalUpdate(target, "app", entities.LocalUpdateOptions{Tag: "v1.0.0"})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Created) != 2 {
			t.Fatalf("created = %#v", result.Created)
		}
	})

	t.Run("stored parts replace the default", func(t *testing.T) {
		target := t.TempDir()
		writeTarget(t, target, "https://example.com/template.git", []string{"docs"})

		interactor := newInteractor(t, templateDir)
		result, err := interactor.LocalUpdate(target, "app", entities.LocalUpdateOptions{Tag: "v1.0.0"})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Created) != 1 || result.Created[0] != "docs/readme.md" {
			t.Fatalf("created = %#v", result.Created)
		}
	})
}

func TestLocalUpdateAdopt(t *testing.T) {
	templateDir := setupTemplate(t)
	target := t.TempDir()
	writeTarget(t, target, "https://example.com/template.git", nil)

	interactor := newInteractor(t, templateDir)
	opts := entities.LocalUpdateOptions{Tag: "v1.0.0", PartsSet: true}
	if _, err := interactor.LocalUpdate(target, "app", opts); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(target, ".sombra/lock.yaml")); err != nil {
		t.Fatal(err)
	}

	result, err := interactor.LocalUpdate(target, "app", opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) == 0 {
		t.Fatal("expected conflicts without a lock")
	}

	adopted := opts
	adopted.Adopt = true
	result, err = interactor.LocalUpdate(target, "app", adopted)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Conflicts) != 0 {
		t.Fatalf("conflicts = %#v", result.Conflicts)
	}
	if len(result.Adopted) == 0 {
		t.Fatalf("adopted = %#v", result.Adopted)
	}
}

func TestLocalUpdateRejectsPathEscape(t *testing.T) {
	templateDir := t.TempDir()
	writeFile(t, templateDir, ".sombra/template.yaml", `
schema: 2
parts:
  base: [/main.go]
rewrites:
  - find: main.go
    replace: "../escape.go"
    in: [name]
`)
	writeFile(t, templateDir, "main.go", "package main\n")

	target := t.TempDir()
	writeTarget(t, target, "https://example.com/template.git", nil)

	interactor := newInteractor(t, templateDir)
	_, err := interactor.LocalUpdate(target, "app", entities.LocalUpdateOptions{Tag: "v1.0.0", PartsSet: true})
	if err == nil {
		t.Fatal("expected a path escape error")
	}
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(target), "escape.go")); !os.IsNotExist(statErr) {
		t.Fatalf("escape file was written: %v", statErr)
	}
}

func TestLocalUpdateDeduplicatesIdenticalEntries(t *testing.T) {
	templateDir := t.TempDir()
	writeFile(t, templateDir, ".sombra/template.yaml", `
schema: 2
parts:
  base: [/Makefile]
  extra: [/Makefile]
`)
	writeFile(t, templateDir, "Makefile", "BINARY=app\n")

	target := t.TempDir()
	writeTarget(t, target, "https://example.com/template.git", []string{"base", "extra"})

	result, err := newInteractor(t, templateDir).LocalUpdate(target, "app", entities.LocalUpdateOptions{Tag: "v1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Created) != 1 || result.Created[0] != "Makefile" {
		t.Fatalf("created = %#v", result.Created)
	}
}

func TestLocalUpdateComposesTemplatesByPart(t *testing.T) {
	appDir := t.TempDir()
	writeFile(t, appDir, ".sombra/template.yaml", `
schema: 2
parts:
  base: [/Makefile]
  ci: [/ci/app.txt]
`)
	writeFile(t, appDir, "Makefile", "BINARY=app\n")
	writeFile(t, appDir, "ci/app.txt", "app\n")

	commonDir := t.TempDir()
	writeFile(t, commonDir, ".sombra/template.yaml", `
schema: 2
parts:
  base: [/common.txt]
  ci: [/ci/common.txt]
`)
	writeFile(t, commonDir, "common.txt", "common base\n")
	writeFile(t, commonDir, "ci/common.txt", "common ci\n")

	target := t.TempDir()
	writeFile(t, target, "sombra.yaml", `templates:
  - id: app
    uri: https://example.com/app.git
    parts: [base]
    vars: {}
  - id: common
    uri: https://example.com/common.git
    parts: [ci]
    vars: {}
`)

	prepare := &fakePrepare{repos: map[string]*fakeRepo{
		"https://example.com/app.git":    {dir: appDir},
		"https://example.com/common.git": {dir: commonDir},
	}}
	interactor := newInteractorWithPrepare(t, prepare)

	if _, err := interactor.LocalUpdate(target, "app", entities.LocalUpdateOptions{Tag: "v1.0.0", PartsSet: true, Parts: []string{"base"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := interactor.LocalUpdate(target, "common", entities.LocalUpdateOptions{Tag: "v1.0.0", PartsSet: true, Parts: []string{"ci"}}); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"Makefile", "ci/common.txt"} {
		if _, err := os.Stat(filepath.Join(target, name)); err != nil {
			t.Fatalf("%s was not written: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(target, "common.txt")); !os.IsNotExist(err) {
		t.Fatalf("common base was copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "ci/app.txt")); !os.IsNotExist(err) {
		t.Fatalf("unselected app part was copied: %v", err)
	}
}
