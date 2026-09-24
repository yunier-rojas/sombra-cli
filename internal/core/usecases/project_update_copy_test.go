// local_copy_test.go
package usecases

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"go.uber.org/mock/gomock"
)

// Mock for creating channels for ScanTree
func createScanResultChannel(results []entities.FileScanResult) <-chan entities.FileScanResult {
	ch := make(chan entities.FileScanResult, len(results))
	for _, r := range results {
		ch <- r
	}
	close(ch)
	return ch
}

func TestLocalCopyInteractor_LocalUpdate(t *testing.T) {
	tests := []struct {
		name   string
		target string
		uri    string
		tag    string
		setup  func(ctrl *gomock.Controller) (
			*MockRepositoryPrepareCase,
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
						{
							URI:     "github.com/other/repo",
							Path:    "lib",
							Current: "v1.0.0",
							Vars: entities.Mappings{
								"libName": "test-lib",
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

				// Setup DirectoryManager mock for file scan
				scanResults := []entities.FileScanResult{
					{
						File:  "src/main.go",
						IsDir: false,
						Err:   nil,
					},
					{
						File:  "src/utils",
						IsDir: true,
						Err:   nil,
					},
					{
						File:  "src/utils/helper.go",
						IsDir: false,
						Err:   nil,
					},
				}
				mockDirectoryManager.EXPECT().
					ScanTree("/tmp/repo", []entities.Wildcard{"**/*"}, nil).
					Return(createScanResultChannel(scanResults))

				// Setup SombraEngine mock for matching and transforming
				for _, result := range scanResults {
					patterns := []*entities.Pattern{
						{
							Pattern: "src/**/*",
							Default: entities.Mappings{
								"projectName": "test-project",
							},
						},
					}

					// Match for each file
					mockSombraEngine.EXPECT().
						Match(result.File, tplDef.Patterns).
						Return(true, patterns, nil)

					// Combine patterns for each file
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
						Combine(patterns).
						Return(mapResult)

					if result.IsDir {
						// Directory processing
						newDir := entities.File(result.File)
						mockSombraEngine.EXPECT().
							NewFile(result.File, mapResult.Path, mapResult.Name).
							Return(newDir)
						mockFileManager.EXPECT().
							EnsureDir(filepath.Join("/path/to/project", "src"), newDir).
							Return(nil)
					} else {
						// File processing
						newFile := entities.File(result.File)
						mockSombraEngine.EXPECT().
							NewFile(result.File, mapResult.Path, mapResult.Name).
							Return(newFile)

						fileContent := []byte("package main\n\nfunc main() {\n  // {{projectName}}\n}\n")
						mockFileManager.EXPECT().
							Read("/tmp/repo", result.File).
							Return(fileContent, nil)

						newContent := []byte("package main\n\nfunc main() {\n  // test-project\n}\n")
						mockSombraEngine.EXPECT().
							TransformFile(fileContent, mapResult.Content, gomock.Any(), false).
							Return(newContent, nil)

						mockFileManager.EXPECT().
							Write(filepath.Join("/path/to/project", "src"), newFile, newContent).
							Return(nil)
					}
				}

				// Check that SombraDef is saved with updated version
				mockSombraDefManager.EXPECT().
					Save(sombraFile, gomock.Any()).
					DoAndReturn(func(fn entities.File, def *entities.SombraDef) error {
						if len(def.Templates) != 2 {
							t.Errorf("Expected 2 templates, got %d", len(def.Templates))
						}

						if def.Templates[0].URI != "github.com/user/repo" {
							t.Errorf("Expected first template URI github.com/user/repo, got %s", def.Templates[0].URI)
						}

						if def.Templates[0].Current != "v1.0.0" {
							t.Errorf("Expected current version to be v1.0.0, got %s", def.Templates[0].Current)
						}

						return nil
					})

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
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

				// Setup RepositoryPort mock for GetTags
				tags := []string{"v0.9.0", "v1.0.0", "v1.1.0"}
				mockRepo.EXPECT().
					GetTags().
					Return(tags, nil)

				// Setup VersionManager mock
				mockVersionManager.EXPECT().
					GetLatest(tags, "*").
					Return(entities.Version("v1.1.0"), nil)

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

				// Setup DirectoryManager mock for file scan
				scanResults := []entities.FileScanResult{
					{
						File:  "src/main.go",
						IsDir: false,
						Err:   nil,
					},
				}
				mockDirectoryManager.EXPECT().
					ScanTree("/tmp/repo", []entities.Wildcard{"**/*"}, nil).
					Return(createScanResultChannel(scanResults))

				// Setup SombraEngine mock for matching and transforming
				patterns := []*entities.Pattern{
					{
						Pattern: "src/**/*",
						Default: entities.Mappings{
							"projectName": "test-project",
						},
					},
				}

				// Match for the file
				mockSombraEngine.EXPECT().
					Match(entities.File("src/main.go"), tplDef.Patterns).
					Return(true, patterns, nil)

				// Combine patterns for the file
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
					Combine(patterns).
					Return(mapResult)

				// File processing
				newFile := entities.File("src/main.go")
				mockSombraEngine.EXPECT().
					NewFile(entities.File("src/main.go"), mapResult.Path, mapResult.Name).
					Return(newFile)

				fileContent := []byte("package main\n\nfunc main() {\n  // {{projectName}}\n}\n")
				mockFileManager.EXPECT().
					Read("/tmp/repo", entities.File("src/main.go")).
					Return(fileContent, nil)

				newContent := []byte("package main\n\nfunc main() {\n  // test-project\n}\n")
				mockSombraEngine.EXPECT().
					TransformFile(fileContent, mapResult.Content, gomock.Any(), false).
					Return(newContent, nil)

				mockFileManager.EXPECT().
					Write(filepath.Join("/path/to/project", "src"), newFile, newContent).
					Return(nil)

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

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
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

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
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

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
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

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "get tags failed",
		},
		{
			name:   "get latest version failure",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "", // Empty means use latest tag
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
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

				// Setup RepositoryPort mock for GetTags
				tags := []string{"v0.9.0", "v1.0.0", "invalid-version"}
				mockRepo.EXPECT().
					GetTags().
					Return(tags, nil)

				// Setup VersionManager mock to fail
				mockVersionManager.EXPECT().
					GetLatest(tags, "*").
					Return(entities.Version(""), errors.New("get latest version failed"))

				// Cleanup should still happen
				mockRepo.EXPECT().Clean().Return(nil)

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "get latest version failed",
		},
		{
			name:   "template render failure",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
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

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "template render failed",
		},
		{
			name:   "file scan error",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
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

				// Setup DirectoryManager mock for file scan with error
				scanResults := []entities.FileScanResult{
					{
						File:  "",
						IsDir: false,
						Err:   errors.New("file scan error"),
					},
				}
				mockDirectoryManager.EXPECT().
					ScanTree("/tmp/repo", []entities.Wildcard{"**/*"}, nil).
					Return(createScanResultChannel(scanResults))

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "file scan error",
		},
		{
			name:   "engine match error",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
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

				// Setup DirectoryManager mock for file scan
				scanResults := []entities.FileScanResult{
					{
						File:  "src/main.go",
						IsDir: false,
						Err:   nil,
					},
				}
				mockDirectoryManager.EXPECT().
					ScanTree("/tmp/repo", []entities.Wildcard{"**/*"}, nil).
					Return(createScanResultChannel(scanResults))

				// Setup SombraEngine mock to fail on match
				mockSombraEngine.EXPECT().
					Match(entities.File("src/main.go"), tplDef.Patterns).
					Return(false, nil, errors.New("engine match error"))

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "engine match error",
		},
		{
			name:   "file read error",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
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

				// Setup DirectoryManager mock for file scan
				scanResults := []entities.FileScanResult{
					{
						File:  "src/main.go",
						IsDir: false,
						Err:   nil,
					},
				}
				mockDirectoryManager.EXPECT().
					ScanTree("/tmp/repo", []entities.Wildcard{"**/*"}, nil).
					Return(createScanResultChannel(scanResults))

				// Setup SombraEngine mock for matching
				patterns := []*entities.Pattern{
					{
						Pattern: "src/**/*",
						Default: entities.Mappings{
							"projectName": "test-project",
						},
					},
				}
				mockSombraEngine.EXPECT().
					Match(entities.File("src/main.go"), tplDef.Patterns).
					Return(true, patterns, nil)

				// Combine patterns
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
					Combine(patterns).
					Return(mapResult)

				// File processing
				newFile := entities.File("src/main.go")
				mockSombraEngine.EXPECT().
					NewFile(entities.File("src/main.go"), mapResult.Path, mapResult.Name).
					Return(newFile)

				// File read error
				mockFileManager.EXPECT().
					Read("/tmp/repo", entities.File("src/main.go")).
					Return(nil, errors.New("file read error"))

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "file read error",
		},
		{
			name:   "file write error",
			target: "/path/to/project",
			uri:    "github.com/user/repo",
			tag:    "v1.0.0",
			setup: func(ctrl *gomock.Controller) (
				*MockRepositoryPrepareCase,
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

				// Setup DirectoryManager mock for file scan
				scanResults := []entities.FileScanResult{
					{
						File:  "src/main.go",
						IsDir: false,
						Err:   nil,
					},
				}
				mockDirectoryManager.EXPECT().
					ScanTree("/tmp/repo", []entities.Wildcard{"**/*"}, nil).
					Return(createScanResultChannel(scanResults))

				// Setup SombraEngine mock for matching
				patterns := []*entities.Pattern{
					{
						Pattern: "src/**/*",
						Default: entities.Mappings{
							"projectName": "test-project",
						},
					},
				}
				mockSombraEngine.EXPECT().
					Match(entities.File("src/main.go"), tplDef.Patterns).
					Return(true, patterns, nil)

				// Combine patterns
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
					Combine(patterns).
					Return(mapResult)

				// File processing
				newFile := entities.File("src/main.go")
				mockSombraEngine.EXPECT().
					NewFile(entities.File("src/main.go"), mapResult.Path, mapResult.Name).
					Return(newFile)

				fileContent := []byte("package main\n\nfunc main() {\n  // {{projectName}}\n}\n")
				mockFileManager.EXPECT().
					Read("/tmp/repo", entities.File("src/main.go")).
					Return(fileContent, nil)

				newContent := []byte("package main\n\nfunc main() {\n  // test-project\n}\n")
				mockSombraEngine.EXPECT().
					TransformFile(fileContent, mapResult.Content, gomock.Any(), false).
					Return(newContent, nil)

				// File write error
				mockFileManager.EXPECT().
					Write(filepath.Join("/path/to/project", "src"), newFile, newContent).
					Return(errors.New("file write error"))

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, mockRepo
			},
			shouldError: true,
			errorMsg:    "file write error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Set up mocks
			mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVersionManager, mockDirectoryManager, mockFileManager, mockSombraEngine, _ := tt.setup(ctrl)

			// Create interactor
			interactor := NewLocalCopyInteractor(
				mockRepoPrepare,
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

func TestLocalCopyInteractor_LocalUpdate_UnknownRef(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
	sombraFile := entities.File("/path/to/project/sombra.yaml")
	mockSombraDefManager.EXPECT().GetFile("/path/to/project").Return(sombraFile)
	mockSombraDefManager.EXPECT().Load(sombraFile).Return(&entities.SombraDef{
		Templates: []*entities.TemplateConfig{{ID: "api", URI: "github.com/user/repo"}},
	}, nil)

	interactor := NewLocalCopyInteractor(nil, nil, mockSombraDefManager, nil, nil, nil, nil)

	_, err := interactor.LocalUpdate("/path/to/project", "missing", "", false)
	if err == nil || err.Error() != `no template registered with id or uri "missing"` {
		t.Fatalf("LocalUpdate() error = %v, want unknown reference error", err)
	}
}

func TestLocalCopyInteractor_LocalUpdate_ResolvesIDToURI(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
	sombraFile := entities.File("/path/to/project/sombra.yaml")
	mockSombraDefManager.EXPECT().GetFile("/path/to/project").Return(sombraFile)
	mockSombraDefManager.EXPECT().Load(sombraFile).Return(&entities.SombraDef{
		Templates: []*entities.TemplateConfig{{ID: "api", URI: "github.com/user/repo"}},
	}, nil)

	mockRepo := NewMockRepositoryPort(ctrl)
	mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
	mockRepoPrepare.EXPECT().Prepare("github.com/user/repo", "").Return(mockRepo, nil)
	mockRepo.EXPECT().GetTags().Return(nil, errors.New("tags failed"))
	mockRepo.EXPECT().Clean().Return(nil)

	interactor := NewLocalCopyInteractor(mockRepoPrepare, nil, mockSombraDefManager, nil, nil, nil, nil)

	_, err := interactor.LocalUpdate("/path/to/project", "api", "", false)
	if err == nil || err.Error() != "tags failed" {
		t.Fatalf("LocalUpdate() error = %v, want tags failed", err)
	}
}
