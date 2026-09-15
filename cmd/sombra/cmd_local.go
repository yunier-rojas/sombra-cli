package main

import (
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
)

type LocalSubcommand struct {
	LocalInit *LocalInitArgs `arg:"subcommand:init"`
	// sombra:skip
	LocalUpdate *LocalUpdateArgs `arg:"subcommand:update"`
	// sombra:end
}

func (args *LocalSubcommand) Run() {
	switch {
	case args.LocalInit != nil:
		args.LocalInit.Run()
	// sombra:skip
	case args.LocalUpdate != nil:
		args.LocalUpdate.Run()
	// sombra:end
	default:
		logger.Panic("command not supported")
	}

}
