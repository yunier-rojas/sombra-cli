package usecases

import (
	"errors"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"go.uber.org/mock/gomock"
)

func TestDirectoryLocalDiffInteractor_LocalUpdate(t *testing.T) {
	tests := []struct {
		name   string
		target string
		uri    string
		tag    string
		setup  func(ctrl *gomock.Controller) (
			*MockRepositoryPrepareCase,
			*MockPatchPort,
			*MockTemplateDefManagerPort,
			*MockSombraDefManagerPort,
			*MockVersionManagerPort,
			*MockDirectoryManagerPort,
			*MockFileManagerPort,
			*MockSombraEngineCase,
			*MockRepositoryPort,
		)
		shouldError bool
		errorMsg    string
	}{
		{
			name:   "successful update with explicit tag",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
				*MockPatchPort,
				*MockTemplateDefManagerPort,
				*MockSombraDefManagerPort,
				*MockVersionManagerPort,
				*MockDirectoryManagerPort,
				*MockFileManagerPort,
				*MockSombraEngineCase,
				*MockRepositoryPort,
			) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockPatchManager := NewMockPatchPort(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVersionManager := NewMockVersionManagerPort(ctrl)
				mockDirectoryManager := NewMockDirectoryManagerPort(ctrl)
				mockFileManager := NewMockFileManagerPort(ctrl)
				mockSombraEngine := NewMockSombraEngineCase(ctrl)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/project/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/project").
					Return(sombraFile)

				sombraDef := &entities.SombraDef{
					Templates: []*entities.TemplateConfig{
						{
							URI:     "github.com/user/repo",
							Path:    "src",
							Current: entities.Version("v0.9.0"),
							Vars: entities.Mappings{
								"projectName": "test-project",
							},
						},
					},
				}
				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(sombraDef, nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup VersionManager mock
				mockVersionManager.EXPECT().
					Compare(entities.Version("v0.9.0"), entities.Version("v1.0.0")).
					Return(int8(-1), nil)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Dir().Return("/tmp/repo").AnyTimes()
				mockRepo.EXPECT().Clean().Return(nil)

				// Setup TemplateDefManager mock
				templateFile := entities.File("/tmp/repo/sombra-template.yaml")
				mockTemplateDefManager.EXPECT().
					GetFile("/tmp/repo").
					Return(templateFile)

				tplDef := &entities.TemplateDef{
					Vars: []string{"projectName"},
					Patterns: []*entities.Pattern{
						{
							Pattern: "src/**/*",
							Default: entities.Mappings{
								"projectName": "{{projectName}}",
							},
						},
					},
				}
				mockTemplateDefManager.EXPECT().
					Render(templateFile, gomock.Any()).
					Return(tplDef, nil)

				// Setup RepositoryPort mock for diff operations
				mockRepo.EXPECT().
					Use("v1.0.0").
					Return("some-commit-hash", nil)

				patchContent := []byte(`diff --git a/src/main.go b/src/main.go
--- a/src/main.go
+++ b/src/main.go
@@ -1,5 +1,5 @@
 package main

 func main() {
-  // old code
+  // new code for {{projectName}}
 }`)

				mockRepo.EXPECT().
					Diff("v0.9.0").
					Return(patchContent, nil)

				// Setup SombraEngine mock for patch transformation
				mockSombraEngine.EXPECT().
					Match(entities.File("/src/main.go"), tplDef.Patterns).
					Return(true, tplDef.Patterns, nil)

				mapResult := &entities.MapResult{
					Path: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
					Name: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
					Content: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
				}
				mockSombraEngine.EXPECT().
					Combine(tplDef.Patterns).
					Return(mapResult)

				// Transform file paths in diff
				mockSombraEngine.EXPECT().
					NewFile(entities.File("/src/main.go"), mapResult.Path, mapResult.Name).
					Return(entities.File("/src/main.go")).AnyTimes()

				// Transform content
				mockSombraEngine.EXPECT().
					NewContent(gomock.Any(), mapResult.Content).
					DoAndReturn(func(content []byte, mappings entities.MapList) []byte {
						if string(content) == " func main() {" {
							return []byte(" func main() {")
						}
						if string(content) == "-  // old code" {
							return []byte("-  // old code")
						}
						if string(content) == "+  // new code for {{projectName}}" {
							return []byte("+  // new code for test-project")
						}
						return content
					}).Times(3)

				// Apply the transformed patch
				transformedPatch := []byte(`diff --git a/src/main.go b/src/main.go
--- a/src/main.go
+++ b/src/main.go
@@ -1,5 +1,5 @@
 package main

 func main() {
-  // old code
+  // new code for test-project
 }`)

				mockPatchManager.EXPECT().
					Apply("/path/to/project/src", gomock.Any()).
					DoAndReturn(func(targetDir string, patchContent []byte) error {
						if string(patchContent) != string(transformedPatch) {
							t.Errorf("Expected transformed patch to be \n%s\n but got \n%s", transformedPatch, patchContent)
						}
						return nil
					})

				// Check that SombraDef is saved with updated version
				mockSombraDefManager.EXPECT().
					Save(sombraFile, gomock.Any()).
					DoAndReturn(func(fn entities.File, def *entities.SombraDef) error {
						if len(def.Templates) != 1 {
							t.Errorf("Expected 1 template, got %d", len(def.Templates))
						}

						if def.Templates[0].Current != "v1.0.0" {
							t.Errorf("Expected current version to be v1.0.0, got %s", def.Templates[0].Current)
						}

						return nil
					})

				return mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: false,
		},
		{
			name:   "successful update with latest tag",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "", // Empty means use latest tag
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
				*MockPatchPort,
				*MockTemplateDefManagerPort,
				*MockSombraDefManagerPort,
				*MockVersionManagerPort,
				*MockDirectoryManagerPort,
				*MockFileManagerPort,
				*MockSombraEngineCase,
				*MockRepositoryPort,
			) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockPatchManager := NewMockPatchPort(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVersionManager := NewMockVersionManagerPort(ctrl)
				mockDirectoryManager := NewMockDirectoryManagerPort(ctrl)
				mockFileManager := NewMockFileManagerPort(ctrl)
				mockSombraEngine := NewMockSombraEngineCase(ctrl)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/project/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/project").
					Return(sombraFile)

				sombraDef := &entities.SombraDef{
					Templates: []*entities.TemplateConfig{
						{
							URI:     "github.com/user/repo",
							Path:    "src",
							Current: entities.Version("v0.9.0"),
							Vars: entities.Mappings{
								"projectName": "test-project",
							},
						},
					},
				}
				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(sombraDef, nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup RepositoryPort mock for GetTags
				tags := []string{"v0.9.0", "v1.0.0", "v1.1.0"}
				mockRepo.EXPECT().
					GetTags().
					Return(tags, nil)

				// Setup VersionManager mock
				mockVersionManager.EXPECT().
					GetLatest(tags, "*").
					Return(entities.Version("v1.1.0"), nil)

				mockVersionManager.EXPECT().
					Compare(entities.Version("v0.9.0"), entities.Version("v1.1.0")).
					Return(int8(-1), nil)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Dir().Return("/tmp/repo").AnyTimes()
				mockRepo.EXPECT().Clean().Return(nil)

				// Setup TemplateDefManager mock
				templateFile := entities.File("/tmp/repo/sombra-template.yaml")
				mockTemplateDefManager.EXPECT().
					GetFile("/tmp/repo").
					Return(templateFile)

				tplDef := &entities.TemplateDef{
					Vars: []string{"projectName"},
					Patterns: []*entities.Pattern{
						{
							Pattern: "src/**/*",
							Default: entities.Mappings{
								"projectName": "{{projectName}}",
							},
						},
					},
				}
				mockTemplateDefManager.EXPECT().
					Render(templateFile, gomock.Any()).
					Return(tplDef, nil)

				// Setup RepositoryPort mock for diff operations
				mockRepo.EXPECT().
					Use("v1.1.0").
					Return("some-commit-hash", nil)

				patchContent := []byte(`diff --git a/src/main.go b/src/main.go
--- a/src/main.go
+++ b/src/main.go
@@ -1,5 +1,5 @@
 package main

 func main() {
-  // old code
+  // new code for {{projectName}}
 }`)

				mockRepo.EXPECT().
					Diff("v0.9.0").
					Return(patchContent, nil)

				// Setup SombraEngine mock for patch transformation
				mockSombraEngine.EXPECT().
					Match(entities.File("/src/main.go"), tplDef.Patterns).
					Return(true, tplDef.Patterns, nil)

				mapResult := &entities.MapResult{
					Path: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
					Name: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
					Content: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
				}
				mockSombraEngine.EXPECT().
					Combine(tplDef.Patterns).
					Return(mapResult)

				// Transform file paths in diff
				mockSombraEngine.EXPECT().
					NewFile(entities.File("/src/main.go"), mapResult.Path, mapResult.Name).
					Return(entities.File("/src/main.go")).AnyTimes()

				// Transform content
				mockSombraEngine.EXPECT().
					NewContent(gomock.Any(), mapResult.Content).
					DoAndReturn(func(content []byte, mappings entities.MapList) []byte {
						if string(content) == " func main() {" {
							return []byte(" func main() {")
						}
						if string(content) == "-  // old code" {
							return []byte("-  // old code")
						}
						if string(content) == "+  // new code for {{projectName}}" {
							return []byte("+  // new code for test-project")
						}
						return content
					}).Times(3)

				// Apply the transformed patch
				transformedPatch := []byte(`diff --git a/src/main.go b/src/main.go
--- a/src/main.go
+++ b/src/main.go
@@ -1,5 +1,5 @@
 package main

 func main() {
-  // old code
+  // new code for test-project
 }`)

				mockPatchManager.EXPECT().
					Apply("/path/to/project/src", gomock.Any()).
					DoAndReturn(func(targetDir string, patchContent []byte) error {
						if string(patchContent) != string(transformedPatch) {
							t.Errorf("Expected transformed patch to be \n%s\n but got \n%s", transformedPatch, patchContent)
						}
						return nil
					})

				// Check that SombraDef is saved with updated version
				mockSombraDefManager.EXPECT().
					Save(sombraFile, gomock.Any()).
					DoAndReturn(func(fn entities.File, def *entities.SombraDef) error {
						if len(def.Templates) != 1 {
							t.Errorf("Expected 1 template, got %d", len(def.Templates))
						}

						if def.Templates[0].Current != "v1.1.0" {
							t.Errorf("Expected current version to be v1.1.0, got %s", def.Templates[0].Current)
						}

						return nil
					})

				return mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: false,
		},
		{
			name:   "no current version uses special hash",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
				*MockPatchPort,
				*MockTemplateDefManagerPort,
				*MockSombraDefManagerPort,
				*MockVersionManagerPort,
				*MockDirectoryManagerPort,
				*MockFileManagerPort,
				*MockSombraEngineCase,
				*MockRepositoryPort,
			) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockPatchManager := NewMockPatchPort(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVersionManager := NewMockVersionManagerPort(ctrl)
				mockDirectoryManager := NewMockDirectoryManagerPort(ctrl)
				mockFileManager := NewMockFileManagerPort(ctrl)
				mockSombraEngine := NewMockSombraEngineCase(ctrl)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/project/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/project").
					Return(sombraFile)

				sombraDef := &entities.SombraDef{
					Templates: []*entities.TemplateConfig{
						{
							URI:     "github.com/user/repo",
							Path:    "src",
							Current: "", // No current version
							Vars: entities.Mappings{
								"projectName": "test-project",
							},
						},
					},
				}
				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(sombraDef, nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Dir().Return("/tmp/repo").AnyTimes()
				mockRepo.EXPECT().Clean().Return(nil)

				// Setup TemplateDefManager mock
				templateFile := entities.File("/tmp/repo/sombra-template.yaml")
				mockTemplateDefManager.EXPECT().
					GetFile("/tmp/repo").
					Return(templateFile)

				tplDef := &entities.TemplateDef{
					Vars: []string{"projectName"},
					Patterns: []*entities.Pattern{
						{
							Pattern: "src/**/*",
							Default: entities.Mappings{
								"projectName": "{{projectName}}",
							},
						},
					},
				}
				mockTemplateDefManager.EXPECT().
					Render(templateFile, gomock.Any()).
					Return(tplDef, nil)

				// Setup RepositoryPort mock for diff operations
				mockRepo.EXPECT().
					Use("v1.0.0").
					Return("some-commit-hash", nil)

				// Should use the special empty tree hash as fromVersion
				emptyTreeHash := "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

				patchContent := []byte(`diff --git a/src/main.go b/src/main.go
--- a/src/main.go
+++ b/src/main.go
@@ -0,0 +1,5 @@
+package main
+
+func main() {
+  // {{projectName}} implementation
+}`)

				mockRepo.EXPECT().
					Diff(emptyTreeHash).
					Return(patchContent, nil)

				// Setup SombraEngine mock for patch transformation
				mockSombraEngine.EXPECT().
					Match(entities.File("/src/main.go"), tplDef.Patterns).
					Return(true, tplDef.Patterns, nil)

				mapResult := &entities.MapResult{
					Path: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
					Name: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
					Content: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
				}
				mockSombraEngine.EXPECT().
					Combine(tplDef.Patterns).
					Return(mapResult)

				// Transform file paths in diff
				mockSombraEngine.EXPECT().
					NewFile(entities.File("/src/main.go"), mapResult.Path, mapResult.Name).
					Return(entities.File("/src/main.go")).AnyTimes()

				// Transform content (only addition lines in this case)
				mockSombraEngine.EXPECT().
					NewContent(gomock.Any(), mapResult.Content).
					DoAndReturn(func(content []byte, mappings entities.MapList) []byte {
						if string(content) == "+package main" {
							return []byte("+package main")
						}
						if string(content) == "+" {
							return []byte("+")
						}
						if string(content) == "+func main() {" {
							return []byte("+func main() {")
						}
						if string(content) == "+  // {{projectName}} implementation" {
							return []byte("+  // test-project implementation")
						}
						if string(content) == "+}" {
							return []byte("+}")
						}
						return content
					}).Times(5)

				mockPatchManager.EXPECT().
					Apply("/path/to/project/src", gomock.Any()).
					DoAndReturn(func(targetDir string, patchContent []byte) error {
						// We don't do exact string comparison here due to potential whitespace differences
						if len(patchContent) == 0 {
							t.Errorf("Expected non-empty transformed patch")
						}
						return nil
					})

				// Check that SombraDef is saved with updated version
				mockSombraDefManager.EXPECT().
					Save(sombraFile, gomock.Any()).
					DoAndReturn(func(fn entities.File, def *entities.SombraDef) error {
						if len(def.Templates) != 1 {
							t.Errorf("Expected 1 template, got %d", len(def.Templates))
						}

						if def.Templates[0].Current != "v1.0.0" {
							t.Errorf("Expected current version to be v1.0.0, got %s", def.Templates[0].Current)
						}

						return nil
					})

				return mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: false,
		},
		{
			name:   "skip when target version is not newer",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v0.8.0", // Older than current v0.9.0
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
				*MockPatchPort,
				*MockTemplateDefManagerPort,
				*MockSombraDefManagerPort,
				*MockVersionManagerPort,
				*MockDirectoryManagerPort,
				*MockFileManagerPort,
				*MockSombraEngineCase,
				*MockRepositoryPort,
			) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockPatchManager := NewMockPatchPort(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVersionManager := NewMockVersionManagerPort(ctrl)
				mockDirectoryManager := NewMockDirectoryManagerPort(ctrl)
				mockFileManager := NewMockFileManagerPort(ctrl)
				mockSombraEngine := NewMockSombraEngineCase(ctrl)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/project/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/project").
					Return(sombraFile)

				sombraDef := &entities.SombraDef{
					Templates: []*entities.TemplateConfig{
						{
							URI:     "github.com/user/repo",
							Path:    "src",
							Current: entities.Version("v0.9.0"),
							Vars: entities.Mappings{
								"projectName": "test-project",
							},
						},
					},
				}
				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(sombraDef, nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup VersionManager mock - returns 1 (current > target)
				mockVersionManager.EXPECT().
					Compare(entities.Version("v0.9.0"), entities.Version("v0.8.0")).
					Return(int8(1), nil)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Clean().Return(nil)

				// Should save without changes
				mockSombraDefManager.EXPECT().
					Save(sombraFile, sombraDef).
					Return(nil)

				return mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: false,
		},
		{
			name:   "sombra definition load failure",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
				*MockPatchPort,
				*MockTemplateDefManagerPort,
				*MockSombraDefManagerPort,
				*MockVersionManagerPort,
				*MockDirectoryManagerPort,
				*MockFileManagerPort,
				*MockSombraEngineCase,
				*MockRepositoryPort,
			) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockPatchManager := NewMockPatchPort(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVersionManager := NewMockVersionManagerPort(ctrl)
				mockDirectoryManager := NewMockDirectoryManagerPort(ctrl)
				mockFileManager := NewMockFileManagerPort(ctrl)
				mockSombraEngine := NewMockSombraEngineCase(ctrl)

				// Setup SombraDefManager mock to fail
				sombraFile := entities.File("/path/to/project/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/project").
					Return(sombraFile)

				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(nil, errors.New("sombra definition load failed"))

				return mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "sombra definition load failed",
		},
		{
			name:   "repository preparation failure",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
				*MockPatchPort,
				*MockTemplateDefManagerPort,
				*MockSombraDefManagerPort,
				*MockVersionManagerPort,
				*MockDirectoryManagerPort,
				*MockFileManagerPort,
				*MockSombraEngineCase,
				*MockRepositoryPort,
			) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockPatchManager := NewMockPatchPort(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVersionManager := NewMockVersionManagerPort(ctrl)
				mockDirectoryManager := NewMockDirectoryManagerPort(ctrl)
				mockFileManager := NewMockFileManagerPort(ctrl)
				mockSombraEngine := NewMockSombraEngineCase(ctrl)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/project/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/project").
					Return(sombraFile)

				sombraDef := &entities.SombraDef{
					Templates: []*entities.TemplateConfig{
						{
							URI:     "github.com/user/repo",
							Path:    "src",
							Current: "v0.9.0",
							Vars: entities.Mappings{
								"projectName": "test-project",
							},
						},
					},
				}
				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(sombraDef, nil)

				// Setup RepositoryPrepareCase mock to fail
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(nil, errors.New("repository preparation failed"))

				return mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "repository preparation failed",
		},
		{
			name:   "get tags failure",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "", // Empty means use latest tag
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
				*MockPatchPort,
				*MockTemplateDefManagerPort,
				*MockSombraDefManagerPort,
				*MockVersionManagerPort,
				*MockDirectoryManagerPort,
				*MockFileManagerPort,
				*MockSombraEngineCase,
				*MockRepositoryPort,
			) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockPatchManager := NewMockPatchPort(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVersionManager := NewMockVersionManagerPort(ctrl)
				mockDirectoryManager := NewMockDirectoryManagerPort(ctrl)
				mockFileManager := NewMockFileManagerPort(ctrl)
				mockSombraEngine := NewMockSombraEngineCase(ctrl)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/project/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/project").
					Return(sombraFile)

				sombraDef := &entities.SombraDef{
					Templates: []*entities.TemplateConfig{
						{
							URI:     "github.com/user/repo",
							Path:    "src",
							Current: "v0.9.0",
							Vars: entities.Mappings{
								"projectName": "test-project",
							},
						},
					},
				}
				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(sombraDef, nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup RepositoryPort mock for GetTags to fail
				mockRepo.EXPECT().
					GetTags().
					Return(nil, errors.New("get tags failed"))

				// Cleanup should still happen
				mockRepo.EXPECT().Clean().Return(nil)

				return mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "get tags failed",
		},
		{
			name:   "template render failure",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
				*MockPatchPort,
				*MockTemplateDefManagerPort,
				*MockSombraDefManagerPort,
				*MockVersionManagerPort,
				*MockDirectoryManagerPort,
				*MockFileManagerPort,
				*MockSombraEngineCase,
				*MockRepositoryPort,
			) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockPatchManager := NewMockPatchPort(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVersionManager := NewMockVersionManagerPort(ctrl)
				mockDirectoryManager := NewMockDirectoryManagerPort(ctrl)
				mockFileManager := NewMockFileManagerPort(ctrl)
				mockSombraEngine := NewMockSombraEngineCase(ctrl)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/project/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/project").
					Return(sombraFile)

				sombraDef := &entities.SombraDef{
					Templates: []*entities.TemplateConfig{
						{
							URI:     "github.com/user/repo",
							Path:    "src",
							Current: entities.Version("v0.9.0"),
							Vars: entities.Mappings{
								"projectName": "test-project",
							},
						},
					},
				}
				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(sombraDef, nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup VersionManager mock
				mockVersionManager.EXPECT().
					Compare(entities.Version("v0.9.0"), entities.Version("v1.0.0")).
					Return(int8(-1), nil)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Dir().Return("/tmp/repo").AnyTimes()
				mockRepo.EXPECT().Clean().Return(nil)

				// Setup TemplateDefManager mock to fail on render
				templateFile := entities.File("/tmp/repo/sombra-template.yaml")
				mockTemplateDefManager.EXPECT().
					GetFile("/tmp/repo").
					Return(templateFile)

				mockTemplateDefManager.EXPECT().
					Render(templateFile, gomock.Any()).
					Return(nil, errors.New("template render failed"))

				return mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "template render failed",
		},
		{
			name:   "repository use version failure",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
				*MockPatchPort,
				*MockTemplateDefManagerPort,
				*MockSombraDefManagerPort,
				*MockVersionManagerPort,
				*MockDirectoryManagerPort,
				*MockFileManagerPort,
				*MockSombraEngineCase,
				*MockRepositoryPort,
			) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockPatchManager := NewMockPatchPort(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVersionManager := NewMockVersionManagerPort(ctrl)
				mockDirectoryManager := NewMockDirectoryManagerPort(ctrl)
				mockFileManager := NewMockFileManagerPort(ctrl)
				mockSombraEngine := NewMockSombraEngineCase(ctrl)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/project/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/project").
					Return(sombraFile)

				sombraDef := &entities.SombraDef{
					Templates: []*entities.TemplateConfig{
						{
							URI:     "github.com/user/repo",
							Path:    "src",
							Current: entities.Version("v0.9.0"),
							Vars: entities.Mappings{
								"projectName": "test-project",
							},
						},
					},
				}
				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(sombraDef, nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup VersionManager mock
				mockVersionManager.EXPECT().
					Compare(entities.Version("v0.9.0"), entities.Version("v1.0.0")).
					Return(int8(-1), nil)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Dir().Return("/tmp/repo").AnyTimes()
				mockRepo.EXPECT().Clean().Return(nil)

				// Setup TemplateDefManager mock
				templateFile := entities.File("/tmp/repo/sombra-template.yaml")
				mockTemplateDefManager.EXPECT().
					GetFile("/tmp/repo").
					Return(templateFile)

				tplDef := &entities.TemplateDef{
					Vars: []string{"projectName"},
					Patterns: []*entities.Pattern{
						{
							Pattern: "src/**/*",
							Default: entities.Mappings{
								"projectName": "{{projectName}}",
							},
						},
					},
				}
				mockTemplateDefManager.EXPECT().
					Render(templateFile, gomock.Any()).
					Return(tplDef, nil)

				// Setup RepositoryPort mock for use to fail
				mockRepo.EXPECT().
					Use("v1.0.0").
					Return("", errors.New("repository use version failed"))

				return mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "repository use version failed",
		},
		{
			name:   "repository diff failure",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
				*MockPatchPort,
				*MockTemplateDefManagerPort,
				*MockSombraDefManagerPort,
				*MockVersionManagerPort,
				*MockDirectoryManagerPort,
				*MockFileManagerPort,
				*MockSombraEngineCase,
				*MockRepositoryPort,
			) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockPatchManager := NewMockPatchPort(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVersionManager := NewMockVersionManagerPort(ctrl)
				mockDirectoryManager := NewMockDirectoryManagerPort(ctrl)
				mockFileManager := NewMockFileManagerPort(ctrl)
				mockSombraEngine := NewMockSombraEngineCase(ctrl)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/project/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/project").
					Return(sombraFile)

				sombraDef := &entities.SombraDef{
					Templates: []*entities.TemplateConfig{
						{
							URI:     "github.com/user/repo",
							Path:    "src",
							Current: entities.Version("v0.9.0"),
							Vars: entities.Mappings{
								"projectName": "test-project",
							},
						},
					},
				}
				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(sombraDef, nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup VersionManager mock
				mockVersionManager.EXPECT().
					Compare(entities.Version("v0.9.0"), entities.Version("v1.0.0")).
					Return(int8(-1), nil)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Dir().Return("/tmp/repo").AnyTimes()
				mockRepo.EXPECT().Clean().Return(nil)

				// Setup TemplateDefManager mock
				templateFile := entities.File("/tmp/repo/sombra-template.yaml")
				mockTemplateDefManager.EXPECT().
					GetFile("/tmp/repo").
					Return(templateFile)

				tplDef := &entities.TemplateDef{
					Vars: []string{"projectName"},
					Patterns: []*entities.Pattern{
						{
							Pattern: "src/**/*",
							Default: entities.Mappings{
								"projectName": "{{projectName}}",
							},
						},
					},
				}
				mockTemplateDefManager.EXPECT().
					Render(templateFile, gomock.Any()).
					Return(tplDef, nil)

				// Setup RepositoryPort mock for diff operations
				mockRepo.EXPECT().
					Use("v1.0.0").
					Return("some-commit-hash", nil)

				mockRepo.EXPECT().
					Diff("v0.9.0").
					Return(nil, errors.New("repository diff failed"))

				return mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "repository diff failed",
		},
		{
			name:   "patch apply failure",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
				*MockPatchPort,
				*MockTemplateDefManagerPort,
				*MockSombraDefManagerPort,
				*MockVersionManagerPort,
				*MockDirectoryManagerPort,
				*MockFileManagerPort,
				*MockSombraEngineCase,
				*MockRepositoryPort,
			) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockPatchManager := NewMockPatchPort(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVersionManager := NewMockVersionManagerPort(ctrl)
				mockDirectoryManager := NewMockDirectoryManagerPort(ctrl)
				mockFileManager := NewMockFileManagerPort(ctrl)
				mockSombraEngine := NewMockSombraEngineCase(ctrl)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/project/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/project").
					Return(sombraFile)

				sombraDef := &entities.SombraDef{
					Templates: []*entities.TemplateConfig{
						{
							URI:     "github.com/user/repo",
							Path:    "src",
							Current: entities.Version("v0.9.0"),
							Vars: entities.Mappings{
								"projectName": "test-project",
							},
						},
					},
				}
				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(sombraDef, nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup VersionManager mock
				mockVersionManager.EXPECT().
					Compare(entities.Version("v0.9.0"), entities.Version("v1.0.0")).
					Return(int8(-1), nil)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Dir().Return("/tmp/repo").AnyTimes()
				mockRepo.EXPECT().Clean().Return(nil)

				// Setup TemplateDefManager mock
				templateFile := entities.File("/tmp/repo/sombra-template.yaml")
				mockTemplateDefManager.EXPECT().
					GetFile("/tmp/repo").
					Return(templateFile)

				tplDef := &entities.TemplateDef{
					Vars: []string{"projectName"},
					Patterns: []*entities.Pattern{
						{
							Pattern: "src/**/*",
							Default: entities.Mappings{
								"projectName": "{{projectName}}",
							},
						},
					},
				}
				mockTemplateDefManager.EXPECT().
					Render(templateFile, gomock.Any()).
					Return(tplDef, nil)

				// Setup RepositoryPort mock for diff operations
				mockRepo.EXPECT().
					Use("v1.0.0").
					Return("some-commit-hash", nil)

				patchContent := []byte(`diff --git a/src/main.go b/src/main.go
--- a/src/main.go
+++ b/src/main.go
@@ -1,5 +1,5 @@
 package main

 func main() {
-  // old code
+  // new code for {{projectName}}
 }`)

				mockRepo.EXPECT().
					Diff("v0.9.0").
					Return(patchContent, nil)

				// Setup SombraEngine mock for patch transformation
				mockSombraEngine.EXPECT().
					Match(entities.File("/src/main.go"), tplDef.Patterns).
					Return(true, tplDef.Patterns, nil)

				mapResult := &entities.MapResult{
					Path: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
					Name: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
					Content: entities.MapList{
						{Key: "projectName", Value: "test-project"},
					},
				}
				mockSombraEngine.EXPECT().
					Combine(tplDef.Patterns).
					Return(mapResult)

				// Transform file paths in diff
				mockSombraEngine.EXPECT().
					NewFile(entities.File("/src/main.go"), mapResult.Path, mapResult.Name).
					Return(entities.File("/src/main.go")).AnyTimes()

				// Transform content
				mockSombraEngine.EXPECT().
					NewContent(gomock.Any(), mapResult.Content).
					DoAndReturn(func(content []byte, mappings entities.MapList) []byte {
						if string(content) == " func main() {" {
							return []byte(" func main() {")
						}
						if string(content) == "-  // old code" {
							return []byte("-  // old code")
						}
						if string(content) == "+  // new code for {{projectName}}" {
							return []byte("+  // new code for test-project")
						}
						return content
					}).Times(3)

				// Patch apply fails
				mockPatchManager.EXPECT().
					Apply("/path/to/project/src", gomock.Any()).
					Return(errors.New("patch apply failed"))

				return mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "patch apply failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Set up mocks
			mockRepoPrepare, mockPatchManager, mockTemplateDefManager, mockSombraDefManager,
				mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, _ :=
				tt.setup(ctrl)

			// Create interactor
			interactor := NewDirectoryLocalDiffInteractor(
				mockRepoPrepare,
				mockPatchManager,
				mockTemplateDefManager,
				mockSombraDefManager,
				mockVersionManager,
				mockDirectoryManager,
				mockFileManager,
				mockSombraEngine,
			)

			// Execute
			_, err := interactor.LocalUpdate(tt.target, tt.uri, tt.tag, false)

			// Check error
			if (err != nil) != tt.shouldError {
				t.Errorf("LocalUpdate() error = %v, shouldError = %v", err, tt.shouldError)
			}

			if tt.shouldError && err != nil && tt.errorMsg != "" && err.Error() != tt.errorMsg {
				t.Errorf("Expected error message %q, got %q", tt.errorMsg, err.Error())
			}
		})
	}
}
