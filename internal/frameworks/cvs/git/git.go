package git

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Service struct {
	uri  string
	path string
	name string
}

func Factory(uri string) (usecases.RepositoryPort, error) {
	tmp, err := os.MkdirTemp("/tmp", "*")
	if err != nil {
		logger.Error("Failed to create temp directory", err)
		return nil, err
	}
	name := uuid.New().String()
	logger.Info(fmt.Sprintf("Created repository service for URL: %s", uri))
	return &Service{
		uri:  uri,
		path: tmp,
		name: name,
	}, nil
}

func (t *Service) Clone() error {
	cmd := exec.Command("git", "clone", t.uri, t.name)
	cmd.Dir = t.path
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		logger.Error("Failed to clone git repo", err)
		return err
	}
	logger.Info(fmt.Sprintf("Cloned repository from URL: %s to path: %s", t.uri, t.Dir()))
	return nil
}

func (t *Service) Clean() error {
	err := os.RemoveAll(t.path) // UseTag RemoveAll to delete the directory and its contents
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to remove directory %s", t.path), err)
		return err
	}
	logger.Info(fmt.Sprintf("Removed temporary directory: %s", t.path))
	return nil
}

func (t *Service) Dir() string {
	return filepath.Join(t.path, t.name)
}

func (t *Service) Use(version string) (string, error) {
	// version can be a branch, a tag or a commit

	// Check if version is a tag before checkout
	isTag := false
	cmd := exec.Command("git", "tag", "-l", version)
	cmd.Dir = t.Dir()
	var tagBuffer bytes.Buffer
	cmd.Stdout = &tagBuffer
	cmd.Stderr = nil
	err := cmd.Run()

	if err == nil && strings.TrimSpace(tagBuffer.String()) == version {
		isTag = true
	}

	// Use git checkout to switch to the specified version
	cmd = exec.Command("git", "checkout", version)
	cmd.Dir = t.Dir()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()

	if err != nil {
		logger.Error(fmt.Sprintf("Failed to checkout version %s", version), err)
		return "", err
	}

	// If it's a tag, return the tag name
	if isTag {
		logger.Info(fmt.Sprintf("Checked out tag: %s", version))
		return version, nil
	}

	// Get the current commit hash
	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = t.Dir()
	var commitBuffer bytes.Buffer
	cmd.Stdout = &commitBuffer
	cmd.Stderr = os.Stderr
	err = cmd.Run()

	if err != nil {
		logger.Error("Failed to get current commit hash", err)
		return "", err
	}

	commitID := strings.TrimSpace(commitBuffer.String())
	logger.Info(fmt.Sprintf("Checked out commit: %s", commitID))
	return commitID, nil
}

func (t *Service) Diff(commit string) ([]byte, error) {
	cmd := exec.Command("git", "diff", "--diff-algorithm=histogram", "--patch", "--unified=10", fmt.Sprintf("%s..HEAD", commit))
	cmd.Dir = t.Dir()
	cmd.Stderr = os.Stderr
	diff, err := cmd.Output()
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get diff for commit %s", commit), err)
		return []byte{}, err
	}
	logger.Info(fmt.Sprintf("Generated diff for commit: %s", commit))
	return diff, nil
}

func (t *Service) GetTags() ([]string, error) {
	cmd := exec.Command("git", "tag", "--sort=v:refname")
	cmd.Dir = t.Dir()
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		logger.Error("Failed to get tags", err)
		return nil, err
	}
	tags := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	logger.Info(fmt.Sprintf("Retrieved tags: %v", tags))
	return tags, nil
}

var _ usecases.RepositoryPort = (*Service)(nil)
