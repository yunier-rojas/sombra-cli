package usecases

import (
	"errors"
	"testing"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"go.uber.org/mock/gomock"
)

func TestSombraEngineCombineCarriesReplace(t *testing.T) {
	engine := NewSombraEngineInteractor(nil, nil, nil)

	tests := []struct {
		name     string
		patterns []*entities.Pattern
		want     *string
	}{
		{
			name: "pattern without replace leaves it unset",
			patterns: []*entities.Pattern{
				{Pattern: "/README.md", Content: entities.Mappings{"a": "b"}},
			},
			want: nil,
		},
		{
			name: "single pattern replace is carried",
			patterns: []*entities.Pattern{
				{Pattern: "/README.md", Replace: "snippets/README.md"},
			},
			want: stringPtr("snippets/README.md"),
		},
		{
			name: "abstract replace is carried as a default",
			patterns: []*entities.Pattern{
				{Pattern: "/**/*", Abstract: true, Replace: "snippets/base"},
			},
			want: stringPtr("snippets/base"),
		},
		{
			name: "specific replace overrides abstract",
			patterns: []*entities.Pattern{
				{Pattern: "/**/*", Abstract: true, Replace: "snippets/base"},
				{Pattern: "/README.md", Replace: "snippets/README.md"},
			},
			want: stringPtr("snippets/README.md"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.Combine(tt.patterns)
			switch {
			case tt.want == nil && got.Replace != nil:
				t.Fatalf("Combine() Replace = %q, want nil", *got.Replace)
			case tt.want != nil && got.Replace == nil:
				t.Fatalf("Combine() Replace = nil, want %q", *tt.want)
			case tt.want != nil && *got.Replace != *tt.want:
				t.Fatalf("Combine() Replace = %q, want %q", *got.Replace, *tt.want)
			}
		})
	}
}

func TestLocalCopyProcessFileUsesReplace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEngine := NewMockSombraEngineCase(ctrl)
	mockFiles := NewMockFileManagerPort(ctrl)
	mockTemplateDef := NewMockTemplateDefManagerPort(ctrl)
	copyObj := &LocalCopyInteractor{engine: mockEngine, localFiles: mockFiles, templateDefManager: mockTemplateDef}

	file := entities.File("/README.md")
	replacement := []byte("# Example\n")
	res := &entities.MapResult{
		Path:    entities.MapList{},
		Name:    entities.MapList{},
		Content: entities.MapList{{Key: "word:sombra", Value: "my-app"}},
		Replace: stringPtr("snippets/README.md"),
	}

	mockEngine.EXPECT().
		NewFile(file, res.Path, res.Name).
		Return(file)

	// The matched source file is ignored: the .sombra replacement is rendered.
	mockTemplateDef.EXPECT().
		RenderReplace("/template", "snippets/README.md", gomock.Any()).
		Return(replacement, nil)

	mockEngine.EXPECT().
		TransformFile(replacement, res.Content, gomock.Any(), false).
		Return([]byte("my-app"), nil)

	mockFiles.EXPECT().
		Write("/target", file, []byte("my-app")).
		Return(nil)

	if err := copyObj.processFile("/template", "/target", file, res, entities.Mappings{}); err != nil {
		t.Fatalf("processFile() unexpected error: %v", err)
	}
}

func TestLocalCopyProcessFileReplaceError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEngine := NewMockSombraEngineCase(ctrl)
	mockFiles := NewMockFileManagerPort(ctrl)
	mockTemplateDef := NewMockTemplateDefManagerPort(ctrl)
	copyObj := &LocalCopyInteractor{engine: mockEngine, localFiles: mockFiles, templateDefManager: mockTemplateDef}

	file := entities.File("/README.md")
	res := &entities.MapResult{
		Path:    entities.MapList{},
		Name:    entities.MapList{},
		Replace: stringPtr("missing.md"),
	}

	mockEngine.EXPECT().
		NewFile(file, res.Path, res.Name).
		Return(file)

	mockTemplateDef.EXPECT().
		RenderReplace("/template", "missing.md", gomock.Any()).
		Return(nil, errors.New("replace file not found"))

	err := copyObj.processFile("/template", "/target", file, res, entities.Mappings{})
	if err == nil || err.Error() != "replace file not found" {
		t.Fatalf("processFile() error = %v, want replace file not found", err)
	}
}

func stringPtr(s string) *string {
	return &s
}
