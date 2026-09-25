package main

import (
	"os"

	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"github.com/yunier-rojas/sombra-cli/internal/runtime"
)

type LocalInitArgs struct {
	Template string `arg:"positional,required" help:"Git Repository to use as template"`

	// sombra:skip
	ID string `arg:"--id" help:"Unique id for the template (generated from the URI when omitted)"`

	// sombra:end
}

func (args *LocalInitArgs) Run() {
	rt, err := runtime.NewLocalInitRuntime()
	if err != nil {
		logger.Panic("Failed to create local init runtime")
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		logger.Panic("What local directory")
		return
	}

	err = rt.UseCase.DoLocalInit(cwd, args.Template, args.ID)
	if err != nil {
		logger.Panic("Failed to init local project")
		return
	}
}
