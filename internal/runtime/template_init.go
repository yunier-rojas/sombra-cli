package runtime

import (
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
)

type TemplateInitRuntime struct {
	UseCase usecases.CliTemplateInitCase
}

func NewTemplateRuntime() (*TemplateInitRuntime, error) {
	container, err := newContainer()
	if err != nil {
		return nil, err
	}

	useCase, err := resolve[usecases.CliTemplateInitCase](container)
	if err != nil {
		return nil, err
	}

	return &TemplateInitRuntime{
		UseCase: useCase,
	}, nil
}
