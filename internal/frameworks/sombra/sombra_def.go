package sombra

import (
	"bytes"
	"os"
	"path/filepath"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"gopkg.in/yaml.v3"
)

// DefService loads and saves the target project's sombra.yaml.
type DefService struct{}

var _ usecases.SombraDefManagerPort = (*DefService)(nil)

func (r *DefService) GetFile(dir string) entities.File {
	return entities.File(filepath.Join(dir, "sombra.yaml"))
}

func (r *DefService) Load(fn entities.File) (*entities.SombraDef, error) {
	data, err := os.ReadFile(string(fn))
	if err != nil {
		if os.IsNotExist(err) {
			return &entities.SombraDef{Templates: make([]*entities.TemplateConfig, 0)}, nil
		}
		if os.IsPermission(err) {
			return nil, entities.Permission("read "+string(fn), err)
		}
		return nil, entities.Storage("read "+string(fn), err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var def entities.SombraDef
	if err := decoder.Decode(&def); err != nil {
		return nil, entities.Invalid("parse "+string(fn), err)
	}
	if def.Templates == nil {
		def.Templates = make([]*entities.TemplateConfig, 0)
	}
	return &def, nil
}

func (r *DefService) Save(fn entities.File, def *entities.SombraDef) error {
	data, err := yaml.Marshal(def)
	if err != nil {
		return entities.Invalid("marshal "+string(fn), err)
	}
	if err := writeAtomic(string(fn), data, 0o644); err != nil {
		return err
	}
	return nil
}

func writeAtomic(filename string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return writeFailure("create directory for "+filename, err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(filename), ".sombra-*")
	if err != nil {
		return writeFailure("create temporary file for "+filename, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return writeFailure("write "+filename, err)
	}
	if err := tmp.Close(); err != nil {
		return writeFailure("close temporary file for "+filename, err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return writeFailure("set mode on "+filename, err)
	}
	if err := os.Rename(tmpName, filename); err != nil {
		return writeFailure("replace "+filename, err)
	}
	return nil
}

func writeFailure(op string, err error) error {
	if os.IsPermission(err) {
		return entities.Permission(op, err)
	}
	return entities.Storage(op, err)
}

func NewDefService() *DefService {
	return &DefService{}
}
