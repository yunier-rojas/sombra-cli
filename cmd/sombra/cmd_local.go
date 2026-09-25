package main

import "github.com/alexflint/go-arg"

type LocalSubcommand struct {
	LocalInit *LocalInitArgs `arg:"subcommand:init"`
	// sombra:skip
	LocalUpdate *LocalUpdateArgs `arg:"subcommand:update"`
	// sombra:end
}

func (args *LocalSubcommand) Run(parser *arg.Parser) {
	switch {
	case args.LocalInit != nil:
		args.LocalInit.Run()
	// sombra:skip
	case args.LocalUpdate != nil:
		args.LocalUpdate.Run()
	// sombra:end
	default:
		_ = parser.FailSubcommand("no command specified", parser.SubcommandNames()...)
	}
}
