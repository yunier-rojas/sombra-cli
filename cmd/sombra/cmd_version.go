package main

import (
	"os"

	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"github.com/yunier-rojas/sombra-cli/internal/runtime"
)

var Version = "dev"

type VersionSubcommand struct{}

func (args *VersionSubcommand) Run() {
	rt, err := runtime.NewVersionRuntime(Version)
	if err != nil {
		logger.Panic("Failed to create version runtime")
		return
	}

	_, _ = os.Stdout.WriteString("sombra " + rt.UseCase.GetVersion() + "\n")
}
