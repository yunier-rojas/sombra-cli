package files

import (
	"github.com/bmatcuk/doublestar/v4"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"os"
	"path/filepath"
	"strings"
)

type DirectoryScannerService struct {
	alwaysIgnore []entities.Wildcard
}

func (f *DirectoryScannerService) IsAllowed(fn entities.File, only []entities.Wildcard, exclude []entities.Wildcard) (bool, error) {
	var match bool
	var err error

	exceptions := append(exclude, f.alwaysIgnore...)
	for _, except := range exceptions {
		match, err = f.PathMatch(fn, except)
		if err != nil {
			logger.Error("Failed to match exception pattern", err)
			return false, err
		}
		if match {
			return false, nil
		}
	}

	for _, value := range only {
		match, err = f.PathMatch(fn, value)
		if err != nil {
			logger.Error("Failed to match pattern", err)
			return false, err
		}

		if match {
			return true, nil
		}
	}

	return false, nil
}

func (f *DirectoryScannerService) PathMatch(fn entities.File, pattern entities.Wildcard) (bool, error) {
	if !strings.HasPrefix(string(pattern), "/") {
		pattern = "**/" + pattern
	}
	match, err := doublestar.PathMatch(string(pattern), string(fn))
	if err != nil {
		logger.Error("Error while matching path pattern", err)
		return false, err
	}
	return match, nil
}

func (f *DirectoryScannerService) ScanTree(baseDir string, only []entities.Wildcard, exclude []entities.Wildcard) <-chan entities.FileScanResult {
	results := make(chan entities.FileScanResult)

	go func() {
		defer close(results)

		err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				results <- entities.FileScanResult{Err: err}
				return err
			}

			if info.IsDir() {
				for _, suffix := range f.alwaysIgnore {
					if strings.HasSuffix(info.Name(), string(suffix)) {
						return filepath.SkipDir
					}
				}
				return nil
			}

			relativeName, err := filepath.Rel(baseDir, path)
			if err != nil {
				logger.Error("error getting relative path name", err)
				results <- entities.FileScanResult{
					Err: err,
				}
				return nil
			}

			if !strings.HasPrefix(relativeName, "/") {
				relativeName = "/" + relativeName
			}
			file := entities.File(relativeName)

			allowed, err := f.IsAllowed(file, only, append(exclude, f.alwaysIgnore...))
			if err != nil {
				results <- entities.FileScanResult{Err: err}
				return nil
			}

			if allowed {
				results <- entities.FileScanResult{File: file, IsDir: info.IsDir()}
			}
			return nil
		})

		if err != nil {
			results <- entities.FileScanResult{Err: err}
		}
	}()

	return results
}

func NewDirectoryScannerService() *DirectoryScannerService {
	return &DirectoryScannerService{
		alwaysIgnore: []entities.Wildcard{
			".git", ".idea", "node_modules", "__pycache__",
		},
	}
}

var _ usecases.DirectoryManagerPort = (*DirectoryScannerService)(nil)
