package main

import (
	"fmt"
	"os"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"github.com/yunier-rojas/sombra-cli/internal/runtime"
)

type LocalUpdateArgs struct {
	Template string   `arg:"positional,required" help:"Template id or repo URL to update"`
	Tag      string   `arg:"--tag" help:"Git tag to use as template"`
	Only     []string `arg:"--only,separate" help:"Parts to enable, replacing the stored selection (repeatable or comma separated)"`
	DryRun   bool     `arg:"--dry-run" help:"Print the plan without writing files"`
	Force    bool     `arg:"--force" help:"Overwrite files even when they were modified"`
	Adopt    bool     `arg:"--adopt" help:"Record existing files whose content already matches"`
	Prune    bool     `arg:"--prune" help:"Remove template files that are no longer produced"`
}

func (args *LocalUpdateArgs) Run() {
	rt, err := runtime.NewLocalUpdateRuntime()
	if err != nil {
		logger.Error("Failed to create local update runtime", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to read working directory", err)
		os.Exit(1)
	}

	parts := splitParts(args.Only)
	noteMissingBase(parts)
	result, err := rt.UseCase.LocalUpdate(cwd, args.Template, entities.LocalUpdateOptions{
		Tag:      args.Tag,
		Parts:    parts,
		PartsSet: len(args.Only) > 0,
		Prune:    args.Prune,
		Force:    args.Force,
		Adopt:    args.Adopt,
		DryRun:   args.DryRun,
	})
	if err != nil {
		logger.Error("Failed to update project", err)
		os.Exit(1)
	}

	if args.DryRun {
		fmt.Println("dry run: no files were written")
	}
	createdLabel := "created"
	updatedLabel := "updated"
	adoptedLabel := "adopted"
	removedLabel := "removed"
	if args.DryRun {
		createdLabel = "would create"
		updatedLabel = "would update"
		adoptedLabel = "would adopt"
		removedLabel = "would remove"
	}
	printPaths(createdLabel, result.Created)
	printPaths(updatedLabel, result.Updated)
	printPaths("unchanged", result.Unchanged)
	printPaths("kept", result.Kept)
	printPaths(adoptedLabel, result.Adopted)
	printPaths(removedLabel, result.Removed)
	if !args.Prune {
		for _, path := range result.Prunable {
			fmt.Printf("prunable  %s (run with --prune to remove)\n", path)
		}
	}

	for _, conflict := range result.Conflicts {
		fmt.Fprintf(os.Stderr, "conflict  %s: %s\n", conflict.Path, conflict.Reason)
	}
	if len(result.Conflicts) > 0 {
		os.Exit(1)
	}
}

func printPaths(label string, paths []entities.File) {
	for _, path := range paths {
		fmt.Printf("%-14s%s\n", label, path)
	}
}
