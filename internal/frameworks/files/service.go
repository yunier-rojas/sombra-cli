package files

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
)

// Service implements file listing, storage, matching and hashing.
type Service struct{}

var (
	_ usecases.FileListerPort  = (*Service)(nil)
	_ usecases.FileStorePort   = (*Service)(nil)
	_ usecases.PathMatcherPort = (*Service)(nil)
	_ usecases.HasherPort      = (*Service)(nil)
)

// List returns the files selected by an entry. Directory entries include every
// regular file below them. Symlinks are skipped.
func (s *Service) List(root string, entry entities.Entry) ([]entities.SourceFile, error) {
	relative := strings.TrimPrefix(entry.Path, "/")
	full := filepath.Join(root, filepath.FromSlash(relative))

	info, err := os.Lstat(full)
	if err != nil {
		return nil, readFailure("template path "+entry.Path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, nil
	}

	if !info.IsDir() {
		if isIgnored(string(entry.Path)) {
			return nil, nil
		}
		return []entities.SourceFile{{Path: entities.File(entry.Path), Mode: uint32(info.Mode().Perm())}}, nil
	}

	var result []entities.SourceFile
	err = filepath.WalkDir(full, func(current string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if item.IsDir() {
			if current != full && isIgnoredDir(item.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if item.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, current)
		if err != nil {
			return fmt.Errorf("resolve template path %s: %w", current, err)
		}
		result = append(result, entities.SourceFile{
			Path: entities.File("/" + filepath.ToSlash(rel)),
			Mode: uint32(info.Mode().Perm()),
		})
		return nil
	})
	if err != nil {
		return nil, readFailure("list template path "+entry.Path, err)
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result, nil
}

// Match reports whether a slash separated path matches a glob. Patterns
// without a slash match at any depth; patterns without a leading slash are
// anchored at the root.
func (s *Service) Match(value, pattern string) (bool, error) {
	value = normalizePath(value)
	pattern = normalizeGlob(pattern)
	if pattern == "" {
		return false, nil
	}
	match, err := doublestar.PathMatch(pattern, value)
	if err != nil {
		return false, entities.Invalid(fmt.Sprintf("match %q against %q", value, pattern), err)
	}
	return match, nil
}

func (s *Service) Read(root, name string) ([]byte, error) {
	filename := resolve(root, name)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, readFailure("read "+filename, err)
	}
	return data, nil
}

func (s *Service) ReadIfExists(root, name string) ([]byte, bool, error) {
	filename := resolve(root, name)
	data, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, readFailure("read "+filename, err)
	}
	return data, true, nil
}

func (s *Service) WriteAtomic(root, name string, content []byte, mode uint32) error {
	filename := resolve(root, name)
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return writeFailure("create directory "+dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".sombra-*")
	if err != nil {
		return writeFailure("create temporary file in "+dir, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return writeFailure("write "+filename, err)
	}
	if err := tmp.Close(); err != nil {
		return writeFailure("close temporary file for "+filename, err)
	}

	perm := os.FileMode(mode)
	if perm == 0 {
		perm = 0o644
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return writeFailure("set mode on "+filename, err)
	}
	if err := os.Rename(tmpName, filename); err != nil {
		return writeFailure("replace "+filename, err)
	}
	return nil
}

func (s *Service) Remove(root, name string) error {
	filename := resolve(root, name)
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return writeFailure("remove "+filename, err)
	}

	rootDir := filepath.Clean(root)
	dir := filepath.Dir(filename)
	for dir != rootDir && dir != "." && dir != string(filepath.Separator) {
		if err := os.Remove(dir); err != nil {
			break
		}
		dir = filepath.Dir(dir)
	}
	return nil
}

func resolve(root, name string) string {
	return filepath.Join(root, filepath.FromSlash(strings.TrimPrefix(name, "/")))
}

func readFailure(op string, err error) error {
	if os.IsNotExist(err) {
		return entities.NotFound(op, err)
	}
	if os.IsPermission(err) {
		return entities.Permission(op, err)
	}
	return entities.Unavailable(op, err)
}

func writeFailure(op string, err error) error {
	if os.IsPermission(err) {
		return entities.Permission(op, err)
	}
	return entities.Storage(op, err)
}

func (s *Service) Hash(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func normalizePath(value string) string {
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	return path.Clean(value)
}

func normalizeGlob(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return ""
	}
	if strings.HasSuffix(pattern, "/") {
		pattern += "**"
	}
	if strings.HasPrefix(pattern, "/") {
		return pattern
	}
	if strings.Contains(pattern, "/") {
		return "/" + pattern
	}
	return "/**/" + pattern
}

func isIgnored(name string) bool {
	for _, ignored := range entities.AlwaysIgnoreDirs {
		if name == ignored || strings.HasPrefix(name, "/"+ignored+"/") {
			return true
		}
	}
	return false
}

func isIgnoredDir(name string) bool {
	for _, ignored := range entities.AlwaysIgnoreDirs {
		if name == ignored {
			return true
		}
	}
	return false
}

// NewService returns a file service.
func NewService() *Service {
	return &Service{}
}
