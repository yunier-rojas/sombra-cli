package usecases

import "github.com/yunier-rojas/sombra-cli/internal/core/entities"

type VersionManagerPort interface {
	GetLatest(tags []string, constraint string) (entities.Version, error)
}
