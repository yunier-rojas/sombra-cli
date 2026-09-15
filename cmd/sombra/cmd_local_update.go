package main

import (
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"github.com/yunier-rojas/sombra-cli/internal/runtime"
	"os"
)

type LocalUpdateArgs struct {
	Template string `arg:"positional,required" help:"Git template to update"`
	Tag      string `arg:"--tag" help:"Git tag to use as template"`
	Method   string `arg:"--method" help:"Method to use for updating the project. (copy|diff)" default:"copy"`
	Prune    bool   `arg:"--prune" help:"Remove files matching a delete pattern (tombstones)"`
}

func (args *LocalUpdateArgs) Run() {
	rt, err := runtime.NewLocalUpdateRuntime()
	if err != nil {
		logger.Panic("Failed to create local update runtime")
		return
	}

	cwd, err := os.Getwd()
	if err != nil {
		logger.Panic("What local directory")
		return
	}

	removed, err := rt.UseCase.DoLocalUpdate(cwd, args.Template, args.Tag, args.Method, args.Prune)
	if err != nil {
		logger.Panic("Failed to do dry copy")
		return
	}

	for _, path := range removed {
		if args.Prune {
			logger.Info("Removed " + path)
		} else {
			logger.Info("Would remove " + path + " (run with --prune to apply)")
		}
	}
}
