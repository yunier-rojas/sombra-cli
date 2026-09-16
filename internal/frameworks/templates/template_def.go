package templates

import (
	"bytes"
	"fmt"
	"github.com/Masterminds/sprig/v3"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type DirectoryTemplateDefService struct {
}

func (c *DirectoryTemplateDefService) GetFile(dir string) entities.File {
	return entities.File(filepath.Join(dir, ".sombra", "default.yaml"))
}

func (c *DirectoryTemplateDefService) Load(def entities.File) (*entities.TemplateDef, error) {
	// Open the file
	file, err := os.Open(string(def))
	if err != nil {
		return nil, fmt.Errorf("failed to open template definition file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Read the file content
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read template definition file: %w", err)
	}

	// Parse the YAML content
	var templateDef entities.TemplateDef
	err = yaml.Unmarshal(data, &templateDef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template definition: %w", err)
	}

	return &templateDef, nil
}

func (c *DirectoryTemplateDefService) Save(def entities.File, templateDef *entities.TemplateDef) error {
	logger.Info("Marshalling template definition to YAML")
	yamlData, err := yaml.Marshal(templateDef)
	if err != nil {
		logger.Error("Failed to marshal template to YAML", err)
		return err
	}

	filePath := string(def)
	logger.Info(fmt.Sprintf("Ensuring directory exists for file: %s", filePath))
	if err = c.ensureDir(filepath.Dir(filePath), nil); err != nil {
		logger.Error("Failed to ensure directory for YAML file", err)
		return err
	}

	logger.Info(fmt.Sprintf("Writing template YAML file to: %s", filePath))
	if err = c.writeFile(filePath, yamlData, 0644); err != nil {
		logger.Error("Failed to write YAML file", err)
		return err
	}

	logger.Info("Successfully saved template definition")
	return nil
}

func (c *DirectoryTemplateDefService) Render(def entities.File, vars entities.Mappings) (*entities.TemplateDef, error) {
	var conf *entities.TemplateDef

	fn := string(def)

	logger.Info(fmt.Sprintf("Attempting to read template file: %s", fn))
	data, err := os.ReadFile(fn)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Error(fmt.Sprintf("default template file not found in repository: %s", fn), err)
			return conf, err
		}
		logger.Error(fmt.Sprintf("failed to read file: %s", fn), err)
		return conf, err
	}

	logger.Info(fmt.Sprintf("Parsing template file: %s", fn))
	rendered, err := renderTemplate(data, vars)
	if err != nil {
		logger.Error("failed to execute template", err)
		return conf, err
	}

	logger.Info("Unmarshalling template YAML to struct")
	err = yaml.Unmarshal(rendered, &conf)
	if err != nil {
		logger.Error("failed to unmarshal YAML", err)
		return conf, err
	}

	logger.Info("Successfully retrieved template definition")
	return conf, nil
}

// RenderReplace loads a file from the template's .sombra directory and renders
// it with the project variables. The name is relative to .sombra and may not
// escape that directory.
func (c *DirectoryTemplateDefService) RenderReplace(dir, name string, vars entities.Mappings) ([]byte, error) {
	base := filepath.Join(dir, ".sombra")
	fn := filepath.Join(base, name)

	rel, err := filepath.Rel(base, fn)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, fmt.Errorf("invalid replace file %q", name)
	}

	data, err := os.ReadFile(fn)
	if err != nil {
		return nil, fmt.Errorf("failed to read replace file %q: %w", name, err)
	}

	return renderTemplate(data, vars)
}

func renderTemplate(data []byte, vars entities.Mappings) ([]byte, error) {
	tmp, err := template.New("template").Funcs(sprig.FuncMap()).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	buf := bytes.NewBufferString("")
	if err := tmp.Execute(buf, vars); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

func (c *DirectoryTemplateDefService) ensureDir(destDir string, _ os.FileInfo) error {
	var err error
	if _, err = os.Stat(destDir); os.IsNotExist(err) {
		if err = os.MkdirAll(destDir, 0777); err != nil {
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

func (c *DirectoryTemplateDefService) writeFile(fn string, content []byte, mode os.FileMode) error {
	err := os.WriteFile(fn, content, mode)
	if err != nil {
		logger.Error("error while writing to file", err)
		return err
	}
	logger.Info("File written successfully: " + fn)
	return nil
}

func NewDefService() *DirectoryTemplateDefService {
	return &DirectoryTemplateDefService{}
}

var _ usecases.TemplateDefManagerPort = (*DirectoryTemplateDefService)(nil)
