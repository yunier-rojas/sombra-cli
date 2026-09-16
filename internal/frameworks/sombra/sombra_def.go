package sombra

import (
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type DefService struct{}

func (r *DefService) GetFile(dir string) entities.File {
	return entities.File(filepath.Join(dir, "sombra.yaml"))
}

func (r *DefService) Load(fn entities.File) (*entities.SombraDef, error) {
	var conf entities.SombraDef

	data, err := os.ReadFile(string(fn))
	if err != nil {
		return &conf, nil
	}
	_ = yaml.Unmarshal(data, &conf)
	return &conf, nil
}

func (r *DefService) Save(fn entities.File, sombraDef *entities.SombraDef) error {
	out, err := yaml.Marshal(sombraDef)

	if err != nil {
		logger.Error("Error marshalling yaml", err)
		return err
	}
	logger.Info("Sombra definition yaml marshalled successfully")

	if writeErr := os.WriteFile(string(fn), out, 0644); writeErr != nil {
		logger.Error("Error writing file", writeErr)
		return err
	}
	logger.Info("Sombra definition file written successfully")
	return nil
}

func NewDefService() *DefService {
	return &DefService{}
}

var _ usecases.SombraDefManagerPort = (*DefService)(nil)
