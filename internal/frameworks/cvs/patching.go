package cvs

import (
	"bytes"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"os"
	"os/exec"
)

type PatchService struct {
}

func NewPatchService() *PatchService {
	return &PatchService{}
}

func (t *PatchService) Apply(dir string, patch []byte) error {
	cmd := exec.Command(
		"patch",
		"-p1", "--force", "--fuzz=5",
		"--no-backup-if-mismatch", "--remove-empty-files",
	)
	cmd.Stdin = bytes.NewReader(patch)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to apply patch", err)
		return err
	}
	logger.Info("Applied patch successfully")
	return nil
}

var _ usecases.PatchPort = (*PatchService)(nil)
