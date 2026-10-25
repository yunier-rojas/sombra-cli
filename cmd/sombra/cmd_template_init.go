package main

import (
	"fmt"
	"os"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"github.com/yunier-rojas/sombra-cli/internal/runtime"
)

type TemplateInitArgs struct {
	Dir              string   `arg:"positional" default:"." help:"Directory of the project to initialize as template"`
	Exclude          []string `arg:"-e,--exclude,separate" help:"Glob of files to exclude"`
	Only             []string `arg:"-o,--only,separate" help:"Glob of files to include"`
	Force            bool     `arg:"--force" help:"Overwrite an existing template definition"`
	DryRun           bool     `arg:"--dry-run" help:"Print the definition without writing it"`
	IncludeUntracked bool     `arg:"--include-untracked" help:"Include untracked files (git sources only)"`
}

func (args *TemplateInitArgs) Run() {
	rt, err := runtime.NewTemplateRuntime()
	if err != nil {
		logger.Error("Failed to create template init runtime", err)
		os.Exit(1)
	}

	result, err := rt.UseCase.DoTemplateInit(args.Dir, entities.TemplateInitOptions{
		Only:             args.Only,
		Exclude:          args.Exclude,
		Force:            args.Force,
		DryRun:           args.DryRun,
		IncludeUntracked: args.IncludeUntracked,
	})
	if err != nil {
		logger.Error("Failed to init template", err)
		os.Exit(1)
	}

	for _, warning := range result.Warnings {
		fmt.Fprintln(os.Stderr, "warning: "+warning)
	}

	if args.DryRun {
		fmt.Print(string(result.YAML))
		return
	}

	fmt.Printf("Wrote %s\n", result.Path)
	fmt.Printf("  base:      %d entries\n", len(result.Definition.Parts[entities.BasePart].Include))
	fmt.Printf("  parts:     %d\n", len(result.Definition.Parts))
	fmt.Printf("  vars:      %d\n", len(result.Definition.Vars))
	fmt.Printf("  rewrites:  %d\n", len(result.Definition.Rewrites))
	fmt.Println("Next:")
	fmt.Println("  - remove business code from base or exclude it")
	fmt.Println("  - move optional trees into parts")
	fmt.Println("  - commit .sombra/template.yaml and tag a release")
}
