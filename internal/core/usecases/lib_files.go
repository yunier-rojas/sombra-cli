package usecases

import (
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

type DirectoryManagerPort interface {
	ScanTree(dir string, only []entities.Wildcard, exclude []entities.Wildcard) <-chan entities.FileScanResult
	IsAllowed(fn entities.File, only []entities.Wildcard, exclude []entities.Wildcard) (bool, error)
	PathMatch(fn entities.File, pattern entities.Wildcard) (bool, error)
}

type FileManagerPort interface {
	EnsureDir(dir string, fn entities.File) error
	Read(dir string, fn entities.File) ([]byte, error)
	Write(dir string, fn entities.File, content []byte) error
	Remove(dir string, fn entities.File) error
}
