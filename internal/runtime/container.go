package runtime

import (
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	// sombra:skip
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/cvs"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/files"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/lock"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/sombra"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/templates"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/vars"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/versions"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/workspace"
	// sombra:end
	"go.uber.org/dig"
)

type provider struct {
	constructor any
	options     []dig.ProvideOption
}

func newContainer() (*dig.Container, error) {
	container := dig.New()

	providers := []provider{
		{constructor: usecases.NewCliVersionInteractor, options: []dig.ProvideOption{dig.As(new(usecases.CliVersionCase))}},

		// sombra:skip
		{constructor: usecases.NewLocalInitInteractor, options: []dig.ProvideOption{dig.As(new(usecases.LocalInitCase))}},
		{constructor: usecases.NewCliLocalInitInteractor, options: []dig.ProvideOption{dig.As(new(usecases.CliLocalInitCase))}},
		{constructor: func() usecases.RepositoryFactory { return cvs.For }},
		{constructor: usecases.NewRepositoryPrepareInteractor, options: []dig.ProvideOption{dig.As(new(usecases.RepositoryPrepareCase))}},
		{constructor: usecases.NewLocalUpdateInteractor, options: []dig.ProvideOption{dig.As(new(usecases.LocalUpdateCase))}},
		{constructor: usecases.NewDirectoryTemplateInitInteractor, options: []dig.ProvideOption{dig.As(new(usecases.TemplateInitCase))}},
		{constructor: usecases.NewCliTemplateInitInteractor, options: []dig.ProvideOption{dig.As(new(usecases.CliTemplateInitCase))}},

		{constructor: files.NewService, options: []dig.ProvideOption{dig.As(
			new(usecases.FileListerPort),
			new(usecases.FileStorePort),
			new(usecases.PathMatcherPort),
			new(usecases.HasherPort),
		)}},
		{constructor: lock.NewService, options: []dig.ProvideOption{dig.As(new(usecases.LockStorePort))}},
		{constructor: sombra.NewDefService, options: []dig.ProvideOption{dig.As(new(usecases.SombraDefManagerPort))}},
		{constructor: sombra.NewTransformer, options: []dig.ProvideOption{dig.As(new(usecases.TransformerPort))}},
		{constructor: templates.NewDefinitionService, options: []dig.ProvideOption{dig.As(new(usecases.DefinitionPort))}},
		{constructor: workspace.NewService, options: []dig.ProvideOption{dig.As(new(usecases.WorkspacePort))}},
		{constructor: vars.NewReader, options: []dig.ProvideOption{dig.As(new(usecases.VariableReaderPort))}},
		{constructor: versions.NewTemplateTagManagerService, options: []dig.ProvideOption{dig.As(new(usecases.VersionManagerPort))}},
		// sombra:end
	}

	for _, p := range providers {
		if err := container.Provide(p.constructor, p.options...); err != nil {
			return nil, err
		}
	}

	return container, nil
}

func resolve[T any](container *dig.Container) (T, error) {
	var value T
	err := container.Invoke(func(resolved T) {
		value = resolved
	})
	return value, err
}
