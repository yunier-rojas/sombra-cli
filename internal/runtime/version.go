package runtime

import (
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
)

type VersionRuntime struct {
	UseCase usecases.CliVersionCase
}

func NewVersionRuntime(version string) (*VersionRuntime, error) {
	container, err := newContainer()
	if err != nil {
		return nil, err
	}

	if err := container.Provide(func() string { return version }); err != nil {
		return nil, err
	}

	useCase, err := resolve[usecases.CliVersionCase](container)
	if err != nil {
		return nil, err
	}

	return &VersionRuntime{
		UseCase: useCase,
	}, nil
}
