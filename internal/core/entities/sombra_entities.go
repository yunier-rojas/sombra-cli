package entities

// File is a slash separated path relative to a workspace root. It always
// starts with "/".
type File string

// Version is a template version, usually a git tag or commit.
type Version string

// Mappings holds the values bound to a template.
type Mappings map[string]string

// sombra:skip

// BasePart is the required part that is selected when no other part is named.
const BasePart = "base"

// TemplateConfig registers a template in a target project.
type TemplateConfig struct {
	ID    string   `yaml:"id,omitempty"`
	URI   string   `yaml:"uri"`
	Path  string   `yaml:"path,omitempty"`
	Parts []string `yaml:"parts,omitempty"`
	Vars  Mappings `yaml:"vars"`
}

// SombraDef is the target project's sombra.yaml file.
type SombraDef struct {
	Templates []*TemplateConfig `yaml:"templates"`
}

// Mode controls who owns a copied file after it is written.
type Mode string

const (
	// ModeSync updates the file only while it is unmodified.
	ModeSync Mode = "sync"
	// ModeSeed writes the file once and never updates it.
	ModeSeed Mode = "seed"
)

// Scope is the part of a path a rewrite applies to.
type Scope string

const (
	ScopePath    Scope = "path"
	ScopeName    Scope = "name"
	ScopeContent Scope = "content"
)

// Entry selects a file or a directory subtree.
type Entry struct {
	// Path is an absolute-looking path inside the template checkout. A trailing
	// slash selects a directory subtree. "/" selects everything.
	Path       string `yaml:"path"`
	Mode       Mode   `yaml:"mode,omitempty"`
	Directives bool   `yaml:"directives,omitempty"`
	Rewrite    *bool  `yaml:"rewrite,omitempty"`
}

// Part is an optional bundle of entries selected with --only.
type Part struct {
	Description string  `yaml:"description,omitempty"`
	Include     []Entry `yaml:"include"`
}

// Rewrite is a single ordered transformation.
type Rewrite struct {
	Find       string   `yaml:"find"`
	As         string   `yaml:"as,omitempty"`
	IgnoreCase bool     `yaml:"ignore_case,omitempty"`
	Replace    *string  `yaml:"replace"`
	In         []Scope  `yaml:"in,omitempty"`
	Files      []string `yaml:"files,omitempty"`
}

// VarSpec describes a variable the consumer is asked to provide.
type VarSpec struct {
	Name        string `yaml:"-"`
	Description string `yaml:"description,omitempty"`
	Default     string `yaml:"default,omitempty"`
}

// TemplateDef is the template definition stored in .sombra/template.yaml.
type TemplateDef struct {
	Schema   int                `yaml:"schema"`
	Vars     map[string]VarSpec `yaml:"vars,omitempty"`
	Parts    map[string]Part    `yaml:"parts"`
	Exclude  []string           `yaml:"exclude,omitempty"`
	Rewrites []Rewrite          `yaml:"rewrites,omitempty"`
}

// Lock records what Sombra wrote in a target project.
type Lock struct {
	Version   int                       `yaml:"lock"`
	Templates map[string]LockedTemplate `yaml:"templates"`
}

// LockedTemplate is the per-template section of the lock.
type LockedTemplate struct {
	Version string                `yaml:"version,omitempty"`
	Parts   []string              `yaml:"parts,omitempty"`
	Files   map[string]LockedFile `yaml:"files"`
}

// LockedFile is one destination recorded in the lock.
type LockedFile struct {
	Hash    string `yaml:"hash"`
	Mode    Mode   `yaml:"mode"`
	Part    string `yaml:"part,omitempty"`
	Source  string `yaml:"source,omitempty"`
	Adopted bool   `yaml:"adopted,omitempty"`
}

// Action is what the planner intends to do with one destination.
type Action string

const (
	ActionCreate    Action = "create"
	ActionUpdate    Action = "update"
	ActionUnchanged Action = "unchanged"
	ActionKept      Action = "kept"
	ActionConflict  Action = "conflict"
	ActionAdopt     Action = "adopt"
	ActionRemove    Action = "remove"
)

// PlannedFile is one destination in a plan.
type PlannedFile struct {
	Source      File
	Destination File
	Mode        Mode
	Part        string
	Action      Action
	Reason      string
	Content     []byte
	Hash        string
	SourceMode  uint32
	Adopted     bool
}

// Plan is the complete result of resolving a template before writing.
type Plan struct {
	Template  string
	Version   string
	Parts     []string
	Files     []PlannedFile
	Conflicts []Conflict
}

// Conflict describes why a destination was left untouched.
type Conflict struct {
	Path   string
	Reason string
}

// UpdateResult is the user-facing summary of an update.
type UpdateResult struct {
	Created   []File
	Updated   []File
	Unchanged []File
	Kept      []File
	Adopted   []File
	Removed   []File
	Prunable  []File
	Conflicts []Conflict
	DryRun    bool
}

// SourceFile is a file discovered in a template checkout.
type SourceFile struct {
	Path File
	Mode uint32
}

// SelectedFile is one source file resolved to a destination and a policy.
type SelectedFile struct {
	Source      File
	Destination File
	Mode        Mode
	Directives  bool
	Rewrite     bool
	Part        string
	SourceMode  uint32
}

// Identity is what template init can detect about the source repository.
type Identity struct {
	Module     string
	Repository string
	Project    string
	Author     string
}

// LocalUpdateOptions controls an update.
type LocalUpdateOptions struct {
	Tag      string
	Parts    []string
	PartsSet bool
	Prune    bool
	Force    bool
	Adopt    bool
	DryRun   bool
}

// TemplateInitOptions controls template init.
type TemplateInitOptions struct {
	Only             []string
	Exclude          []string
	Force            bool
	DryRun           bool
	IncludeUntracked bool
}

// TemplateInitResult is the output of template init.
type TemplateInitResult struct {
	Definition *TemplateDef
	Path       File
	YAML       []byte
	Warnings   []string
	DryRun     bool
}

// sombra:end
