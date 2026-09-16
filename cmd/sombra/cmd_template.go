package main

import (
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
)

type TemplateSubcommand struct {
	TemplateInit *TemplateInitArgs `arg:"subcommand:init"`
}

func (args *TemplateSubcommand) Run() {
	switch {
	case args.TemplateInit != nil:
		args.TemplateInit.Run()

	default:
		logger.Panic("command not supported")
	}

}
