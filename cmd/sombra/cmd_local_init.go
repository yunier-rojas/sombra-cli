package main

import (
	"os"

	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"github.com/yunier-rojas/sombra-cli/internal/runtime"
)

type LocalInitArgs struct {
	Template string `arg:"positional,required" help:"Git Repository to use as template"`
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

	err = rt.UseCase.DoLocalInit(cwd, args.Template)
	if err != nil {
		logger.Panic("Failed to init local project")
		return
	}
}
