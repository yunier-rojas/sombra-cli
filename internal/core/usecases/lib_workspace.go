package usecases

import "github.com/yunier-rojas/sombra-cli/internal/core/entities"

// WorkspacePort inspects the source repository during template init.
type WorkspacePort interface {
	ListFiles(dir string, includeUntracked bool) ([]entities.File, error)
	Identity(dir string) entities.Identity
}
