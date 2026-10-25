package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"github.com/yunier-rojas/sombra-cli/internal/runtime"
)

type LocalInitArgs struct {
	Template string `arg:"positional,required" help:"Git repository to use as template"`

	ID   string   `arg:"--id" help:"Unique id for the template (generated from the URI when omitted)"`
	With []string `arg:"--with,separate" help:"Template parts to enable (default: base; repeatable or comma separated)"`
}

func (args *LocalInitArgs) Run() {
	rt, err := runtime.NewLocalInitRuntime()
	if err != nil {
		logger.Error("Failed to create local init runtime", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("Failed to read working directory", err)
		os.Exit(1)
	}

	parts := splitParts(args.With)
	noteMissingBase(parts)
	if err := rt.UseCase.DoLocalInit(cwd, args.Template, args.ID, parts); err != nil {
		logger.Error("Failed to init local project", err)
		os.Exit(1)
	}
}

func splitParts(values []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part == "" || seen[part] {
				continue
			}
			seen[part] = true
			result = append(result, part)
		}
	}
	return result
}

func noteMissingBase(parts []string) {
	if len(parts) == 0 {
		return
	}
	for _, part := range parts {
		if part == entities.BasePart {
			return
		}
	}
	fmt.Fprintln(os.Stderr, "note: base is not selected; pass --with base,... to include it")
}
