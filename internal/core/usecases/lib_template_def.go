package usecases

import "github.com/yunier-rojas/sombra-cli/internal/core/entities"

type TemplateDefManagerPort interface {
	GetFile(dir string) entities.File
	Load(def entities.File) (*entities.TemplateDef, error)
	Save(def entities.File, templateDef *entities.TemplateDef) error
	Render(def entities.File, vars entities.Mappings) (*entities.TemplateDef, error)
	// RenderReplace loads a file from the template's .sombra directory and
	// renders it with the project variables. The name is relative to .sombra.
	RenderReplace(dir, name string, vars entities.Mappings) ([]byte, error)
}
