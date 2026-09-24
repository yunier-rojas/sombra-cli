package entities

// sombra:skip

import "strings"

// sombra:end

type File string

// sombra:skip

type RepoUpdateInfo struct {
	Branch         string
	CurrentVersion string
}

// File is the relative path of a file in the base directory.

type FileScanResult struct {
	// File is the relative path without the initial "/"
	File  File
	IsDir bool
	Err   error
}

type MappingType int

type Wildcard string

const (
	MappingDefault MappingType = iota + 1
	MappingPath
	MappingName
	MappingContent
)

type AbstractMappingCandidate struct {
	For      MappingType
	Name     string
	Key      string
	Value    string
	Priority int
}

type FileAnalysis struct {
	Pattern     *Pattern
	IsWildcard  bool
	IsMandatory bool
	Vars        []string
	Exclude     []Wildcard
}

type Mappings map[string]string

type TemplateConfig struct {
	ID      string   `yaml:"id,omitempty"`
	URI     string   `yaml:"uri" validate:"required"`
	Path    string   `yaml:"path,omitempty"`
	Current Version  `yaml:"current,omitempty"`
	Vars    Mappings `yaml:"vars" validate:"required"`
}

type Pattern struct {
	Pattern         Wildcard   `yaml:"pattern" validate:"required"`
	Abstract        bool       `yaml:"abstract,omitempty"`
	CopyOnly        bool       `yaml:"copy_only,omitempty"`
	Verbatim        bool       `yaml:"verbatim,omitempty"`
	BlockDirectives bool       `yaml:"block_directives,omitempty"`
	Delete          bool       `yaml:"delete,omitempty"`
	Replace         string     `yaml:"replace,omitempty"`
	When            *Condition `yaml:"when,omitempty"`
	Default         Mappings   `yaml:"default,omitempty"`
	Path            Mappings   `yaml:"path,omitempty"`
	Name            Mappings   `yaml:"name,omitempty"`
	Content         Mappings   `yaml:"content,omitempty"`
	Except          []Wildcard `yaml:"except,omitempty"`
}

// Condition is the tri-state value behind the `when` pattern field. A nil
// pointer means "enabled". It decodes leniently: the template definition can be
// loaded before it is rendered (see `sombra init`), in which case the field
// still holds a Go-template expression. Such values are treated as enabled;
// only an explicit falsey value disables the pattern.
type Condition bool

func (c *Condition) UnmarshalText(text []byte) error {
	switch strings.ToLower(strings.TrimSpace(string(text))) {
	case "false", "0", "no", "off":
		*c = false
	default:
		*c = true
	}
	return nil
}

func (c Condition) MarshalText() ([]byte, error) {
	if c {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}

type TemplateDef struct {
	Vars     []string   `yaml:"vars"`
	Patterns []*Pattern `yaml:"patterns" validate:"required"`
}

type Version string

type MapItem struct {
	Key   string
	Value string
}

type MapList []MapItem

type MapResult struct {
	Path            MapList
	Name            MapList
	Content         MapList
	Replace         *string
	BlockDirectives bool
}

type SombraUpdateInfo struct {
	Branch  string
	Changes []*SombraTemplateUpdateInfo
}

type SombraDef struct {
	Templates []*TemplateConfig `yaml:"templates" validate:"required"`
}

type SombraTemplateUpdateInfo struct {
	Operation string
	Template  string
	Version   string
}

// sombra:end
