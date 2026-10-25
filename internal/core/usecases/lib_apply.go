package usecases

import "github.com/yunier-rojas/sombra-cli/internal/core/entities"

// FileListerPort lists template files for one entry.
type FileListerPort interface {
	List(root string, entry entities.Entry) ([]entities.SourceFile, error)
}

// PathMatcherPort matches slash separated paths against glob patterns.
type PathMatcherPort interface {
	Match(path, pattern string) (bool, error)
}

// FileReaderPort reads target files.
type FileReaderPort interface {
	Read(root, path string) ([]byte, error)
	// ReadIfExists returns the content and whether the file exists.
	ReadIfExists(root, path string) ([]byte, bool, error)
}

// FileWriterPort writes and removes target files.
type FileWriterPort interface {
	WriteAtomic(root, path string, content []byte, mode uint32) error
	Remove(root, path string) error
}

// FileStorePort reads, writes and removes target files. Root is an opaque
// directory identifier; path is slash separated and relative to root.
type FileStorePort interface {
	FileReaderPort
	FileWriterPort
}

// LockStorePort loads and saves the target lock file.
type LockStorePort interface {
	Load(target string) (*entities.Lock, error)
	Save(target string, lock *entities.Lock) error
}

// HasherPort produces a stable content hash.
type HasherPort interface {
	Hash(content []byte) string
}
