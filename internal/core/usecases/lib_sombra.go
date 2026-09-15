package usecases

import "github.com/yunier-rojas/sombra-cli/internal/core/entities"

type SombraDefManagerPort interface {
	GetFile(dir string) entities.File
	Load(fn entities.File) (*entities.SombraDef, error)
	Save(fn entities.File, def *entities.SombraDef) error
}
