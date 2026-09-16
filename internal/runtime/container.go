package runtime

import (
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	// sombra:skip
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/analysers"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/cvs"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/files"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/sombra"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/templates"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/vars"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/versions"
	// sombra:end
	"go.uber.org/dig"
)

// sombra:skip

type updateCasesParams struct {
	dig.In
	Copy usecases.LocalUpdateCase `name:"copy"`
	Diff usecases.LocalUpdateCase `name:"diff"`
}

// sombra:end

type provider struct {
	constructor interface{}
	options     []dig.ProvideOption
}

func newContainer() (*dig.Container, error) {
	container := dig.New()

	providers := []provider{
		{constructor: usecases.NewLocalInitInteractor, options: []dig.ProvideOption{dig.As(new(usecases.LocalInitCase))}},
		{constructor: usecases.NewCliLocalInitInteractor, options: []dig.ProvideOption{dig.As(new(usecases.CliLocalInitCase))}},

		// sombra:skip
		{constructor: func() usecases.RepositoryFactory { return cvs.For }},
		{constructor: usecases.NewRepositoryPrepareInteractor, options: []dig.ProvideOption{dig.As(new(usecases.RepositoryPrepareCase))}},
		{constructor: files.NewDirectoryScannerService, options: []dig.ProvideOption{dig.As(new(usecases.DirectoryManagerPort))}},
		{constructor: files.NewFileManagerService, options: []dig.ProvideOption{dig.As(new(usecases.FileManagerPort))}},
		{constructor: sombra.NewProcessor, options: []dig.ProvideOption{dig.As(new(usecases.SombraStringsPort))}},
		{constructor: sombra.NewDefService, options: []dig.ProvideOption{dig.As(new(usecases.SombraDefManagerPort))}},
		{constructor: templates.NewDefService, options: []dig.ProvideOption{dig.As(new(usecases.TemplateDefManagerPort))}},
		{constructor: vars.NewReader, options: []dig.ProvideOption{dig.As(new(usecases.VariableReaderPort))}},
		{constructor: versions.NewTemplateTagManagerService, options: []dig.ProvideOption{dig.As(new(usecases.VersionManagerPort))}},
		{constructor: cvs.NewPatchService, options: []dig.ProvideOption{dig.As(new(usecases.PatchPort))}},
		{constructor: analysers.GetRegistry},
		{constructor: usecases.NewSombraEngineInteractor, options: []dig.ProvideOption{dig.As(new(usecases.SombraEngineCase))}},
		{constructor: usecases.NewLocalCopyInteractor, options: []dig.ProvideOption{dig.As(new(usecases.LocalUpdateCase)), dig.Name("copy")}},
		{constructor: usecases.NewDirectoryLocalDiffInteractor, options: []dig.ProvideOption{dig.As(new(usecases.LocalUpdateCase)), dig.Name("diff")}},
		{constructor: newCliUpdateInteractor},
		{constructor: usecases.NewDirectoryTemplateInitInteractor, options: []dig.ProvideOption{dig.As(new(usecases.TemplateInitCase))}},
		{constructor: usecases.NewCliTemplateInitInteractor, options: []dig.ProvideOption{dig.As(new(usecases.CliTemplateInitCase))}},
		// sombra:end

	}

	for _, p := range providers {
		if err := container.Provide(p.constructor, p.options...); err != nil {
			return nil, err
		}
	}

	return container, nil
}

// sombra:skip

func newCliUpdateInteractor(params updateCasesParams) usecases.CliUpdateCase {
	return usecases.NewCliUpdateInteractor(params.Copy, params.Diff)
}

// sombra:end

func resolve[T any](container *dig.Container) (T, error) {
	var value T
	err := container.Invoke(func(resolved T) {
		value = resolved
	})
	return value, err
}
