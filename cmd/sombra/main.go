package main

import (
	"errors"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
)

/***********
COMMANDS
************/

var args struct {
	// sombra:skip
	Local    *LocalSubcommand    `arg:"subcommand:local"`
	Template *TemplateSubcommand `arg:"subcommand:template"`
	// sombra:end
	Version *VersionSubcommand `arg:"subcommand:version"`
}

/***********
CONFIG
************/

func main() {
	logger.Init()

	parser, err := arg.NewParser(arg.Config{Program: "sombra", Out: os.Stderr}, &args)
	if err != nil {
		logger.Error("Failed to create command parser", err)
		os.Exit(1)
	}

	if err := parser.Parse(os.Args[1:]); err != nil {
		if errors.Is(err, arg.ErrHelp) {
			parser.WriteHelp(os.Stdout)
			os.Exit(0)
		}

		_ = parser.FailSubcommand(err.Error(), parser.SubcommandNames()...)
		os.Exit(2)
	}

	switch {
	// sombra:skip
	case args.Local != nil:
		args.Local.Run(parser)
	case args.Template != nil:
		args.Template.Run(parser)
	// sombra:end
	case args.Version != nil:
		args.Version.Run()
	default:
		_ = parser.FailSubcommand("no command specified")
	}
}
