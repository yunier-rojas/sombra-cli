// project_init_test.go
package usecases

import (
	"errors"
	"reflect"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"go.uber.org/mock/gomock"
)

// MockVariableReaderPort is a mock for VariableReaderPort interface
type MockVariableReaderPort struct {
	ctrl     *gomock.Controller
	recorder *MockVariableReaderPortMockRecorder
}

type MockVariableReaderPortMockRecorder struct {
	mock *MockVariableReaderPort
}

func NewMockVariableReaderPort(ctrl *gomock.Controller) *MockVariableReaderPort {
	mock := &MockVariableReaderPort{ctrl: ctrl}
	mock.recorder = &MockVariableReaderPortMockRecorder{mock}
	return mock
}

func (m *MockVariableReaderPort) EXPECT() *MockVariableReaderPortMockRecorder {
	return m.recorder
}

func (m *MockVariableReaderPort) GetValues(vars []string) *entities.Mappings {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValues", vars)
	ret0, _ := ret[0].(*entities.Mappings)
	return ret0
}

func (mr *MockVariableReaderPortMockRecorder) GetValues(vars interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValues", reflect.TypeOf((*MockVariableReaderPort)(nil).GetValues), vars)
}

func TestLocalInitInteractor_LocalInit(t *testing.T) {
	tests := []struct {
		name   string
		target string
		uri    string
		setUp  func(ctrl *gomock.Controller) (
			*MockRepositoryPrepareCase,
			*MockTemplateDefManagerPort,
			*MockSombraDefManagerPort,
			*MockVariableReaderPort,
			*MockRepositoryPort,
		)
		shouldError bool
		errorMsg    string
	}{
		{
			name:   "successful initialization",
			target: "/path/to/target",
			uri:    "github.com/user/repo",
			setUp: func(ctrl *gomock.Controller) (*MockRepositoryPrepareCase, *MockTemplateDefManagerPort, *MockSombraDefManagerPort, *MockVariableReaderPort, *MockRepositoryPort) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVarReader := NewMockVariableReaderPort(ctrl)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Dir().Return("/tmp/repo").AnyTimes()
				mockRepo.EXPECT().Clean().Return(nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup TemplateDefManager mock
				templateFile := entities.File("/tmp/repo/sombra-template.yaml")
				mockTemplateDefManager.EXPECT().
					GetFile("/tmp/repo").
					Return(templateFile)

				mockTemplateDefManager.EXPECT().
					Load(templateFile).
					Return(&entities.TemplateDef{
						Vars: []string{"projectName", "projectVersion"},
					}, nil)

				// Setup VariableReader mock
				expectedMappings := &entities.Mappings{
					"projectName":    "test-project",
					"projectVersion": "1.0.0",
				}
				mockVarReader.EXPECT().
					GetValues([]string{"projectName", "projectVersion"}).
					Return(expectedMappings)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/target/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/target").
					Return(sombraFile)

				existingDef := &entities.SombraDef{
					Templates: []*entities.TemplateConfig{
						{
							URI: "github.com/other/repo",
							Vars: entities.Mappings{
								"name": "other-project",
							},
						},
					},
				}
				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(existingDef, nil)

				// Check that save is called with updated definition
				mockSombraDefManager.EXPECT().
					Save(sombraFile, gomock.Any()).
					DoAndReturn(func(fn entities.File, def *entities.SombraDef) error {
						if len(def.Templates) != 2 {
							t.Errorf("Expected 2 templates, got %d", len(def.Templates))
						}
						if def.Templates[0].URI != "github.com/other/repo" {
							t.Errorf("Expected first template URI github.com/other/repo, got %s", def.Templates[0].URI)
						}
						if def.Templates[1].URI != "github.com/user/repo" {
							t.Errorf("Expected second template URI github.com/user/repo, got %s", def.Templates[1].URI)
						}

						// Check that the variables were properly set
						mappings := def.Templates[1].Vars
						if mappings["projectName"] != "test-project" {
							t.Errorf("Expected projectName to be test-project, got %s", mappings["projectName"])
						}
						if mappings["projectVersion"] != "1.0.0" {
							t.Errorf("Expected projectVersion to be 1.0.0, got %s", mappings["projectVersion"])
						}

						return nil
					})

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVarReader, mockRepo
			},
			shouldError: false,
		},
		{
			name:   "repository preparation failure",
			target: "/path/to/target",
			uri:    "invalid-uri",
			setUp: func(ctrl *gomock.Controller) (*MockRepositoryPrepareCase, *MockTemplateDefManagerPort, *MockSombraDefManagerPort, *MockVariableReaderPort, *MockRepositoryPort) {
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVarReader := NewMockVariableReaderPort(ctrl)

				// Setup RepositoryPrepareCase mock to fail
				mockRepoPrepare.EXPECT().
					Prepare("invalid-uri", "").
					Return(nil, errors.New("repository preparation failed"))

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVarReader, nil
			},
			shouldError: true,
			errorMsg:    "repository preparation failed",
		},
		{
			name:   "template definition load failure",
			target: "/path/to/target",
			uri:    "github.com/user/repo",
			setUp: func(ctrl *gomock.Controller) (*MockRepositoryPrepareCase, *MockTemplateDefManagerPort, *MockSombraDefManagerPort, *MockVariableReaderPort, *MockRepositoryPort) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVarReader := NewMockVariableReaderPort(ctrl)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Dir().Return("/tmp/repo").AnyTimes()
				mockRepo.EXPECT().Clean().Return(nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup TemplateDefManager mock to fail on load
				templateFile := entities.File("/tmp/repo/sombra-template.yaml")
				mockTemplateDefManager.EXPECT().
					GetFile("/tmp/repo").
					Return(templateFile)

				mockTemplateDefManager.EXPECT().
					Load(templateFile).
					Return(nil, errors.New("template definition load failed"))

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVarReader, mockRepo
			},
			shouldError: true,
			errorMsg:    "template definition load failed",
		},
		{
			name:   "sombra definition load failure",
			target: "/path/to/target",
			uri:    "github.com/user/repo",
			setUp: func(ctrl *gomock.Controller) (*MockRepositoryPrepareCase, *MockTemplateDefManagerPort, *MockSombraDefManagerPort, *MockVariableReaderPort, *MockRepositoryPort) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVarReader := NewMockVariableReaderPort(ctrl)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Dir().Return("/tmp/repo").AnyTimes()
				mockRepo.EXPECT().Clean().Return(nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup TemplateDefManager mock
				templateFile := entities.File("/tmp/repo/sombra-template.yaml")
				mockTemplateDefManager.EXPECT().
					GetFile("/tmp/repo").
					Return(templateFile)

				mockTemplateDefManager.EXPECT().
					Load(templateFile).
					Return(&entities.TemplateDef{
						Vars: []string{"projectName"},
					}, nil)

				// Setup VariableReader mock
				expectedMappings := &entities.Mappings{
					"projectName": "test-project",
				}
				mockVarReader.EXPECT().
					GetValues([]string{"projectName"}).
					Return(expectedMappings)

				// Setup SombraDefManager mock to fail on load
				sombraFile := entities.File("/path/to/target/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/target").
					Return(sombraFile)

				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(nil, errors.New("sombra definition load failed"))

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVarReader, mockRepo
			},
			shouldError: true,
			errorMsg:    "sombra definition load failed",
		},
		{
			name:   "sombra definition save failure",
			target: "/path/to/target",
			uri:    "github.com/user/repo",
			setUp: func(ctrl *gomock.Controller) (*MockRepositoryPrepareCase, *MockTemplateDefManagerPort, *MockSombraDefManagerPort, *MockVariableReaderPort, *MockRepositoryPort) {
				mockRepo := NewMockRepositoryPort(ctrl)
				mockRepoPrepare := NewMockRepositoryPrepareCase(ctrl)
				mockTemplateDefManager := NewMockTemplateDefManagerPort(ctrl)
				mockSombraDefManager := NewMockSombraDefManagerPort(ctrl)
				mockVarReader := NewMockVariableReaderPort(ctrl)

				// Setup RepositoryPort mock
				mockRepo.EXPECT().Dir().Return("/tmp/repo").AnyTimes()
				mockRepo.EXPECT().Clean().Return(nil)

				// Setup RepositoryPrepareCase mock
				mockRepoPrepare.EXPECT().
					Prepare("github.com/user/repo", "").
					Return(mockRepo, nil)

				// Setup TemplateDefManager mock
				templateFile := entities.File("/tmp/repo/sombra-template.yaml")
				mockTemplateDefManager.EXPECT().
					GetFile("/tmp/repo").
					Return(templateFile)

				mockTemplateDefManager.EXPECT().
					Load(templateFile).
					Return(&entities.TemplateDef{
						Vars: []string{"projectName"},
					}, nil)

				// Setup VariableReader mock
				expectedMappings := &entities.Mappings{
					"projectName": "test-project",
				}
				mockVarReader.EXPECT().
					GetValues([]string{"projectName"}).
					Return(expectedMappings)

				// Setup SombraDefManager mock
				sombraFile := entities.File("/path/to/target/sombra.yaml")
				mockSombraDefManager.EXPECT().
					GetFile("/path/to/target").
					Return(sombraFile)

				mockSombraDefManager.EXPECT().
					Load(sombraFile).
					Return(&entities.SombraDef{
						Templates: []*entities.TemplateConfig{},
					}, nil)

				// Setup SombraDefManager to fail on save
				mockSombraDefManager.EXPECT().
					Save(sombraFile, gomock.Any()).
					Return(errors.New("sombra definition save failed"))

				return mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVarReader, mockRepo
			},
			shouldError: true,
			errorMsg:    "sombra definition save failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Set up mocks
			mockRepoPrepare, mockTemplateDefManager, mockSombraDefManager, mockVarReader, _ := tt.setUp(ctrl)

			// Create interactor
			interactor := NewLocalInitInteractor(
				mockRepoPrepare,
				mockTemplateDefManager,
				mockSombraDefManager,
				mockVarReader,
			)

			// Execute
			err := interactor.LocalInit(tt.target, tt.uri)

			// Check error
			if (err != nil) != tt.shouldError {
				t.Errorf("LocalInit() error = %v, shouldError = %v", err, tt.shouldError)
			}

			if tt.shouldError && err != nil && tt.errorMsg != "" && err.Error() != tt.errorMsg {
				t.Errorf("Expected error message %q, got %q", tt.errorMsg, err.Error())
			}
		})
	}
}
