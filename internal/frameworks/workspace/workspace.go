package workspace

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
)

// Service inspects a source repository for template init.
type Service struct{}

var _ usecases.WorkspacePort = (*Service)(nil)

// ListFiles returns the tracked files of a git repository. When git is not
// available or the repository has no commits, it walks the directory and skips
// well-known build and cache directories.
func (s *Service) ListFiles(dir string, includeUntracked bool) ([]entities.File, error) {
	if files, ok := gitFiles(dir, includeUntracked); ok {
		return files, nil
	}
	return walkFiles(dir)
}

// Identity detects the values used to seed template vars and rewrites.
func (s *Service) Identity(dir string) entities.Identity {
	identity := entities.Identity{
		Module:     modulePath(dir),
		Repository: normalizeRemote(gitOutput(dir, "remote", "get-url", "origin")),
		Author:     strings.TrimSpace(gitOutput(dir, "config", "user.name")),
	}

	identity.Project = filepath.Base(identity.Module)
	if identity.Module == "" || identity.Project == "." {
		identity.Project = filepath.Base(filepath.Clean(dir))
	}
	return identity
}

func gitFiles(dir string, includeUntracked bool) ([]entities.File, bool) {
	if strings.TrimSpace(gitOutput(dir, "rev-parse", "--verify", "HEAD")) == "" {
		return nil, false
	}

	args := []string{"-C", dir, "ls-files", "-z"}
	if includeUntracked {
		args = append(args, "--others", "--exclude-standard")
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, false
	}

	var result []entities.File
	for _, raw := range strings.Split(string(output), "\x00") {
		if raw == "" {
			continue
		}
		full := filepath.Join(dir, filepath.FromSlash(raw))
		info, err := os.Lstat(full)
		if err != nil || info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		result = append(result, entities.File("/"+filepath.ToSlash(raw)))
	}

	result = filterSpecial(result)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, true
}

func walkFiles(dir string) ([]entities.File, error) {
	var result []entities.File

	err := filepath.WalkDir(dir, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if current != dir && ignoredDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(dir, current)
		if err != nil {
			return err
		}
		result = append(result, entities.File("/"+filepath.ToSlash(rel)))
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, entities.NotFound("list "+dir, err)
		}
		if os.IsPermission(err) {
			return nil, entities.Permission("list "+dir, err)
		}
		return nil, entities.Unavailable("list "+dir, err)
	}

	result = filterSpecial(result)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func filterSpecial(files []entities.File) []entities.File {
	result := make([]entities.File, 0, len(files))
	for _, file := range files {
		value := string(file)
		if value == "/sombra.yaml" {
			continue
		}
		if value == "/.sombra" || strings.HasPrefix(value, "/.sombra/") {
			continue
		}
		result = append(result, file)
	}
	return result
}

func ignoredDir(name string) bool {
	for _, ignored := range entities.InitIgnoreDirs {
		if name == ignored {
			return true
		}
	}
	return false
}

func modulePath(dir string) string {
	file, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func gitOutput(dir string, args ...string) string {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func normalizeRemote(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimSuffix(raw, ".git")
	raw = strings.TrimRight(raw, "/")
	if raw == "" {
		return ""
	}

	if index := strings.Index(raw, "://"); index >= 0 {
		raw = raw[index+3:]
		if at := strings.Index(raw, "@"); at >= 0 {
			raw = raw[at+1:]
		}
		if slash := strings.Index(raw, "/"); slash >= 0 {
			raw = raw[slash+1:]
		}
		return raw
	}

	if colon := strings.Index(raw, ":"); colon >= 0 {
		raw = raw[colon+1:]
	}
	return strings.TrimPrefix(raw, "/")
}

// NewService returns a workspace service.
func NewService() *Service {
	return &Service{}
}
