package runtime

import (
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
)

type LocalUpdateRuntime struct {
	UseCase usecases.CliUpdateCase
}

func NewLocalUpdateRuntime() (*LocalUpdateRuntime, error) {
	container, err := newContainer()
	if err != nil {
		return nil, err
	}

	useCase, err := resolve[usecases.CliUpdateCase](container)
	if err != nil {
		return nil, err
	}

	return &LocalUpdateRuntime{
		UseCase: useCase,
	}, nil
}
