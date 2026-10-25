package usecases

import "github.com/yunier-rojas/sombra-cli/internal/core/entities"

type CliTemplateInitCase interface {
	DoTemplateInit(dir string, opts entities.TemplateInitOptions) (*entities.TemplateInitResult, error)
}

type CliTemplateInitInteractor struct {
	templateCase TemplateInitCase
}

func (l *CliTemplateInitInteractor) DoTemplateInit(dir string, opts entities.TemplateInitOptions) (*entities.TemplateInitResult, error) {
	return l.templateCase.TemplateInit(dir, opts)
}

func NewCliTemplateInitInteractor(templateCase TemplateInitCase) *CliTemplateInitInteractor {
	return &CliTemplateInitInteractor{templateCase: templateCase}
}

var _ CliTemplateInitCase = (*CliTemplateInitInteractor)(nil)
