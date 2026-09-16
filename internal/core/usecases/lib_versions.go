package usecases

import "github.com/yunier-rojas/sombra-cli/internal/core/entities"

type VersionManagerPort interface {
	GetLatest(tags []string, constraint string) (entities.Version, error)
	GetNext(tags []string, constraint string, current entities.Version) (entities.Version, error)
	Compare(v1, v2 entities.Version) (int8, error)
}
