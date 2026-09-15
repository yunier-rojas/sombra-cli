package usecases

import (
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

type CliTemplateInitCase interface {
	DoTemplateInit(templateDir string, only []string, exclude []string) error
}

type CliTemplateInitInteractor struct {
	templateCase TemplateInitCase
}

func (l *CliTemplateInitInteractor) DoTemplateInit(templateDir string, only []string, exclude []string) error {
	err := l.templateCase.TemplateInit(templateDir, entities.ConvertToWildcards(only), entities.ConvertToWildcards(exclude))
	if err != nil {
		return err
	}
	return nil
}

func NewCliTemplateInitInteractor(templateEngine TemplateInitCase) *CliTemplateInitInteractor {
	return &CliTemplateInitInteractor{templateCase: templateEngine}
}

var _ CliTemplateInitCase = (*CliTemplateInitInteractor)(nil)
