package files

import (
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"os"
	"path/filepath"
)

type FileManagerService struct{}

func (f *FileManagerService) EnsureDir(dir string, fn entities.File) error {
	var err error
	destDir := filepath.Join(dir, string(fn))
	if _, err = os.Stat(destDir); os.IsNotExist(err) {
		if err = os.MkdirAll(destDir, os.FileMode(0777)); err != nil {
			logger.Error("failed to create directory", err)
			return err
		}
		logger.Info("Directory created: " + destDir)
	} else if err != nil {
		logger.Error("unexpected error when checking directory existence", err)
		return err
	} else {
		logger.Info("Directory already exists: " + destDir)
	}

	return nil
}

func (f *FileManagerService) Read(dir string, fn entities.File) ([]byte, error) {
	file := filepath.Join(dir, string(fn))
	data, err := os.ReadFile(file)
	if err != nil {
		logger.Error("error while reading file", err)
		return nil, err
	}
	logger.Info("File read successfully: " + file)
	return data, nil
}

func (f *FileManagerService) Write(dir string, fn entities.File, content []byte) error {
	file := filepath.Join(dir, string(fn))
	path := filepath.Dir(file)
	err := f.EnsureDir(path, "")
	if err != nil {
		logger.Error("failed to ensure directory", err)
		return err
	}

	err = os.WriteFile(file, content, 0777)
	if err != nil {
		logger.Error("error while writing to file", err)
		return err
	}
	logger.Info("File written successfully: " + file)
	return nil
}

func (f *FileManagerService) Remove(dir string, fn entities.File) error {
	file := filepath.Join(dir, string(fn))
	if err := os.Remove(file); err != nil {
		logger.Error("error while removing file", err)
		return err
	}
	logger.Info("File removed successfully: " + file)
	return nil
}

func NewFileManagerService() *FileManagerService {
	return &FileManagerService{}
}

var _ usecases.FileManagerPort = (*FileManagerService)(nil)
