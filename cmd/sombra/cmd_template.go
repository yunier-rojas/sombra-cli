package main

import "github.com/alexflint/go-arg"

type TemplateSubcommand struct {
	TemplateInit *TemplateInitArgs `arg:"subcommand:init"`
}

func (args *TemplateSubcommand) Run(parser *arg.Parser) {
	switch {
	case args.TemplateInit != nil:
		args.TemplateInit.Run()

	default:
		_ = parser.FailSubcommand("no command specified", parser.SubcommandNames()...)
	}
}
