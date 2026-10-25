package usecases

import "github.com/yunier-rojas/sombra-cli/internal/core/entities"

// DefinitionLoaderPort loads template definitions.
type DefinitionLoaderPort interface {
	GetFile(dir string) entities.File
	Load(file entities.File) (*entities.TemplateDef, error)
	Exists(file entities.File) (bool, error)
}

// DefinitionSaverPort saves template definitions.
type DefinitionSaverPort interface {
	Save(file entities.File, def *entities.TemplateDef) error
	Marshal(def *entities.TemplateDef) ([]byte, error)
}

// ValueRendererPort renders Go template values.
type ValueRendererPort interface {
	RenderValue(text string, vars entities.Mappings) (string, error)
}

// OverlayRendererPort renders overlay files.
type OverlayRendererPort interface {
	RenderOverlay(dir string, source entities.File, vars entities.Mappings) ([]byte, bool, error)
}

// DefinitionPort loads and saves template definitions and renders values.
type DefinitionPort interface {
	DefinitionLoaderPort
	DefinitionSaverPort
	ValueRendererPort
	OverlayRendererPort
}
