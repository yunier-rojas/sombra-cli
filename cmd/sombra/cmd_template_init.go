package main

import (
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"github.com/yunier-rojas/sombra-cli/internal/runtime"
)

type TemplateInitArgs struct {
	Dir     string   `arg:"positional" default:"." help:"Directory of the project to initialize as template"`
	Exclude []string `arg:"-e,--exclude,separate" help:"Wildcard of the files to exclude"`
	Only    []string `arg:"-o,--only,separate"  help:"Wildcard of the files to include"`
}

func (args *TemplateInitArgs) Run() {
	rt, err := runtime.NewTemplateRuntime()
	if err != nil {
		logger.Panic("Failed to create template init runtime")
		return
	}

	if args.Exclude == nil {
		args.Exclude = []string{}
	}

	if args.Only == nil {
		args.Only = []string{"/**/*"}
	}

	err = rt.UseCase.DoTemplateInit(args.Dir, args.Only, args.Exclude)
	if err != nil {
		logger.Panic("Failed to init template")
		return
	}
}
