package runtime

import (
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
)

type LocalInitRuntime struct {
	UseCase usecases.CliLocalInitCase
}

func NewLocalInitRuntime() (*LocalInitRuntime, error) {
	container, err := newContainer()
	if err != nil {
		return nil, err
	}

	useCase, err := resolve[usecases.CliLocalInitCase](container)
	if err != nil {
		return nil, err
	}

	return &LocalInitRuntime{
		UseCase: useCase,
	}, nil
}
