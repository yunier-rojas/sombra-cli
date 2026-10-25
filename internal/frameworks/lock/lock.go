package lock

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"gopkg.in/yaml.v3"
)

const currentVersion = 1

// Service loads and saves .sombra/lock.yaml.
type Service struct{}

var _ usecases.LockStorePort = (*Service)(nil)

func (s *Service) Path(target string) string {
	return filepath.Join(target, ".sombra", "lock.yaml")
}

func (s *Service) Load(target string) (*entities.Lock, error) {
	data, err := os.ReadFile(s.Path(target))
	if err != nil {
		if os.IsNotExist(err) {
			return emptyLock(), nil
		}
		if os.IsPermission(err) {
			return nil, entities.Permission("read lock file", err)
		}
		return nil, entities.Storage("read lock file", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var lock entities.Lock
	if err := decoder.Decode(&lock); err != nil {
		return nil, entities.Invalid("parse lock file "+s.Path(target), err)
	}
	if lock.Version != 0 && lock.Version != currentVersion {
		return nil, entities.Invalid(fmt.Sprintf("unsupported lock version %d in %s", lock.Version, s.Path(target)), nil)
	}
	lock.Version = currentVersion
	if lock.Templates == nil {
		lock.Templates = make(map[string]entities.LockedTemplate)
	}
	for name, template := range lock.Templates {
		if template.Files == nil {
			template.Files = make(map[string]entities.LockedFile)
			lock.Templates[name] = template
		}
	}
	return &lock, nil
}

func (s *Service) Save(target string, lock *entities.Lock) error {
	if lock == nil {
		lock = emptyLock()
	}
	lock.Version = currentVersion

	data, err := yaml.Marshal(lock)
	if err != nil {
		return entities.Invalid("marshal lock file", err)
	}

	filename := s.Path(target)
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return writeFailure("create lock directory", err)
	}
	return writeAtomic(filename, data, 0o644)
}

func emptyLock() *entities.Lock {
	return &entities.Lock{
		Version:   currentVersion,
		Templates: make(map[string]entities.LockedTemplate),
	}
}

func writeAtomic(filename string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(filename), ".sombra-*")
	if err != nil {
		return writeFailure("create temporary lock file", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return writeFailure("write temporary lock file", err)
	}
	if err := tmp.Close(); err != nil {
		return writeFailure("close temporary lock file", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return writeFailure("set lock file mode", err)
	}
	if err := os.Rename(tmpName, filename); err != nil {
		return writeFailure("replace lock file", err)
	}
	return nil
}

func writeFailure(op string, err error) error {
	if os.IsPermission(err) {
		return entities.Permission(op, err)
	}
	return entities.Storage(op, err)
}

// NewService returns a lock service.
func NewService() *Service {
	return &Service{}
}
