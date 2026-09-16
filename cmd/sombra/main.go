package main

import (
	"github.com/alexflint/go-arg"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
)

/***********
COMMANDS
************/

var args struct {
	Local *LocalSubcommand `arg:"subcommand:local"`
	// sombra:skip
	Template *TemplateSubcommand `arg:"subcommand:template"`
	// sombra:end
}

/***********
CONFIG
************/

func main() {
	logger.Init()
	arg.MustParse(&args)

	switch {
	case args.Local != nil:
		args.Local.Run()
	// sombra:skip
	case args.Template != nil:
		args.Template.Run()
	// sombra:end
	default:
		logger.Panic("No command specified")
	}
}
