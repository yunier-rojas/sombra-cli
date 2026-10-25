package templates

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"gopkg.in/yaml.v3"
)

// DefinitionService loads .sombra/template.yaml.
type DefinitionService struct{}

var _ usecases.DefinitionPort = (*DefinitionService)(nil)

func (s *DefinitionService) GetFile(dir string) entities.File {
	return entities.File(filepath.Join(dir, ".sombra", "template.yaml"))
}

func (s *DefinitionService) Load(file entities.File) (*entities.TemplateDef, error) {
	data, err := os.ReadFile(string(file))
	if err != nil {
		if os.IsNotExist(err) {
			legacy := filepath.Join(filepath.Dir(filepath.Dir(string(file))), ".sombra", "default.yaml")
			if _, statErr := os.Stat(legacy); statErr == nil {
				return nil, entities.Invalid("legacy template definition "+legacy, nil)
			}
			return nil, entities.NotFound("template definition "+string(file), err)
		}
		if os.IsPermission(err) {
			return nil, entities.Permission("read template definition "+string(file), err)
		}
		return nil, entities.Unavailable("read template definition "+string(file), err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var raw rawDefinition
	if err := decoder.Decode(&raw); err != nil {
		return nil, entities.Invalid("parse template definition "+string(file), err)
	}

	def := raw.toEntities()
	if err := validate(def); err != nil {
		return nil, entities.Invalid("invalid template definition "+string(file), err)
	}
	return def, nil
}

func (s *DefinitionService) Save(file entities.File, def *entities.TemplateDef) error {
	data, err := s.Marshal(def)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(string(file)), 0o755); err != nil {
		return writeFailure("create definition directory", err)
	}
	return writeAtomic(string(file), data, 0o644)
}

func (s *DefinitionService) Marshal(def *entities.TemplateDef) ([]byte, error) {
	data, err := yaml.Marshal(def)
	if err != nil {
		return nil, entities.Invalid("marshal template definition", err)
	}
	return data, nil
}

func (s *DefinitionService) Exists(file entities.File) (bool, error) {
	_, err := os.Stat(string(file))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	if os.IsPermission(err) {
		return false, entities.Permission("stat template definition "+string(file), err)
	}
	return false, entities.Unavailable("stat template definition "+string(file), err)
}

func (s *DefinitionService) RenderValue(text string, vars entities.Mappings) (string, error) {
	rendered, err := renderValue(text, vars)
	if err != nil {
		return "", entities.Invalid("render value", err)
	}
	return rendered, nil
}

func (s *DefinitionService) RenderOverlay(dir string, source entities.File, vars entities.Mappings) ([]byte, bool, error) {
	base := filepath.Join(dir, ".sombra", "overlay")
	relative := strings.TrimPrefix(string(source), "/")
	fn := filepath.Join(base, filepath.FromSlash(relative))

	rel, err := filepath.Rel(base, fn)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return nil, false, entities.Invalid(fmt.Sprintf("invalid overlay path %q", source), err)
	}

	data, err := os.ReadFile(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		if os.IsPermission(err) {
			return nil, false, entities.Permission("read overlay "+string(source), err)
		}
		return nil, false, entities.Unavailable("read overlay "+string(source), err)
	}

	rendered, err := renderValue(string(data), vars)
	if err != nil {
		return nil, false, entities.Invalid("render overlay "+string(source), err)
	}
	return []byte(rendered), true, nil
}

func renderValue(text string, vars entities.Mappings) (string, error) {
	tpl, err := template.New("value").Funcs(sprig.FuncMap()).Option("missingkey=error").Parse(text)
	if err != nil {
		return "", fmt.Errorf("parse template value: %w", err)
	}

	var buffer bytes.Buffer
	if err := tpl.Execute(&buffer, vars); err != nil {
		return "", fmt.Errorf("render template value: %w", err)
	}
	return buffer.String(), nil
}

func writeAtomic(filename string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(filename), ".sombra-*")
	if err != nil {
		return writeFailure("create temporary file", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return writeFailure("write temporary file", err)
	}
	if err := tmp.Close(); err != nil {
		return writeFailure("close temporary file", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return writeFailure("set file mode", err)
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

// rawDefinition mirrors the YAML shape before normalization. The raw types
// accept the scalar shorthand forms so a definition can stay terse.
type rawDefinition struct {
	Schema   int                `yaml:"schema"`
	Vars     map[string]rawVar  `yaml:"vars"`
	Parts    map[string]rawPart `yaml:"parts"`
	Exclude  []string           `yaml:"exclude"`
	Rewrites []rawRewrite       `yaml:"rewrites"`
}

type rawEntry struct {
	Path       string `yaml:"path"`
	Mode       string `yaml:"mode"`
	Directives bool   `yaml:"directives"`
	Rewrite    *bool  `yaml:"rewrite"`
}

func (e *rawEntry) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		e.Path = node.Value
		return nil
	}
	if err := checkKeys(node, "path", "mode", "directives", "rewrite"); err != nil {
		return err
	}
	type plain rawEntry
	var value plain
	if err := node.Decode(&value); err != nil {
		return err
	}
	*e = rawEntry(value)
	return nil
}

type rawPart struct {
	Description string     `yaml:"description"`
	Include     []rawEntry `yaml:"include"`
}

func (p *rawPart) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.SequenceNode {
		var entries []rawEntry
		if err := node.Decode(&entries); err != nil {
			return err
		}
		p.Include = entries
		return nil
	}
	if err := checkKeys(node, "description", "include"); err != nil {
		return err
	}
	type plain rawPart
	var value plain
	if err := node.Decode(&value); err != nil {
		return err
	}
	*p = rawPart(value)
	return nil
}

type rawVar struct {
	Description string `yaml:"description"`
	Default     string `yaml:"default"`
}

func (v *rawVar) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		v.Default = node.Value
		return nil
	}
	if err := checkKeys(node, "description", "default"); err != nil {
		return err
	}
	type plain rawVar
	var value plain
	if err := node.Decode(&value); err != nil {
		return err
	}
	*v = rawVar(value)
	return nil
}

type rawRewrite struct {
	Find       string  `yaml:"find"`
	As         string  `yaml:"as"`
	IgnoreCase bool    `yaml:"ignore_case"`
	Replace    *string `yaml:"replace"`
	In         rawList `yaml:"in"`
	Files      rawList `yaml:"files"`
}

type rawList []string

func (l *rawList) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		*l = []string{node.Value}
		return nil
	}
	var values []string
	if err := node.Decode(&values); err != nil {
		return err
	}
	*l = values
	return nil
}

func (d *rawDefinition) toEntities() *entities.TemplateDef {
	def := &entities.TemplateDef{
		Schema:  d.Schema,
		Vars:    make(map[string]entities.VarSpec, len(d.Vars)),
		Parts:   make(map[string]entities.Part, len(d.Parts)),
		Exclude: append([]string(nil), d.Exclude...),
	}

	for name, value := range d.Vars {
		def.Vars[name] = entities.VarSpec{Name: name, Description: value.Description, Default: value.Default}
	}
	for name, part := range d.Parts {
		converted := entities.Part{Description: part.Description}
		for _, entry := range part.Include {
			converted.Include = append(converted.Include, entry.toEntity())
		}
		def.Parts[name] = converted
	}
	for _, rewrite := range d.Rewrites {
		converted := entities.Rewrite{
			Find:       rewrite.Find,
			As:         rewrite.As,
			IgnoreCase: rewrite.IgnoreCase,
			Replace:    rewrite.Replace,
			Files:      append([]string(nil), rewrite.Files...),
		}
		for _, scope := range rewrite.In {
			converted.In = append(converted.In, entities.Scope(scope))
		}
		def.Rewrites = append(def.Rewrites, converted)
	}
	return def
}

func (e rawEntry) toEntity() entities.Entry {
	return entities.Entry{
		Path:       e.Path,
		Mode:       entities.Mode(e.Mode),
		Directives: e.Directives,
		Rewrite:    e.Rewrite,
	}
}

func checkKeys(node *yaml.Node, allowed ...string) error {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	known := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		known[key] = struct{}{}
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		if _, ok := known[key]; !ok {
			return fmt.Errorf("field %q not found", key)
		}
	}
	return nil
}

func validate(def *entities.TemplateDef) error {
	if def.Schema != 2 {
		return fmt.Errorf("schema must be 2, got %d", def.Schema)
	}
	if _, ok := def.Parts[entities.BasePart]; !ok {
		return fmt.Errorf("parts must declare a %q part", entities.BasePart)
	}

	for name, part := range def.Parts {
		if name != entities.BasePart && len(part.Include) == 0 {
			return fmt.Errorf("part %q has no include entries", name)
		}
		if err := validateEntries(part.Include, "part "+name); err != nil {
			return err
		}
	}
	for _, pattern := range def.Exclude {
		if !doublestar.ValidatePathPattern(pattern) {
			return fmt.Errorf("invalid exclude pattern %q", pattern)
		}
	}
	for index, rule := range def.Rewrites {
		if err := validateRewrite(rule); err != nil {
			return fmt.Errorf("rewrite %d: %w", index+1, err)
		}
	}
	return nil
}

func validateEntries(entries []entities.Entry, where string) error {
	for _, entry := range entries {
		if entry.Path == "" {
			return fmt.Errorf("%s: path is required", where)
		}
		if !strings.HasPrefix(entry.Path, "/") {
			return fmt.Errorf("%s: path %q must start with /", where, entry.Path)
		}
		if hasParentSegment(entry.Path) {
			return fmt.Errorf("%s: path %q must not contain a parent segment", where, entry.Path)
		}
		if strings.ContainsAny(entry.Path, "*?[") {
			return fmt.Errorf("%s: path %q must not contain globs", where, entry.Path)
		}
		switch entry.Mode {
		case "", entities.ModeSync, entities.ModeSeed:
		default:
			return fmt.Errorf("%s: invalid mode %q", where, entry.Mode)
		}
	}
	return nil
}

func validateRewrite(rule entities.Rewrite) error {
	if strings.TrimSpace(rule.Find) == "" {
		return fmt.Errorf("find is required")
	}
	if rule.Replace == nil {
		return fmt.Errorf("replace is required (use an empty string to delete)")
	}

	switch rule.As {
	case "", "literal", "word", "regex", "json", "yaml", "ini":
	default:
		return fmt.Errorf("invalid as %q", rule.As)
	}

	if rule.As == "regex" {
		if _, err := regexp.Compile(rule.Find); err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
	}

	structured := rule.As == "json" || rule.As == "yaml" || rule.As == "ini"
	in := rule.In
	if len(in) == 0 {
		in = []entities.Scope{entities.ScopeContent}
	}
	for _, scope := range in {
		switch scope {
		case entities.ScopePath, entities.ScopeName, entities.ScopeContent:
		case "all":
			if structured {
				return fmt.Errorf("%s rewrites may only use the content scope", rule.As)
			}
		default:
			return fmt.Errorf("invalid scope %q", scope)
		}
		if structured && scope != entities.ScopeContent {
			return fmt.Errorf("%s rewrites may only use the content scope", rule.As)
		}
	}
	for _, pattern := range rule.Files {
		if !doublestar.ValidatePathPattern(normalizePattern(pattern)) {
			return fmt.Errorf("invalid files pattern %q", pattern)
		}
	}
	return nil
}

func hasParentSegment(value string) bool {
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func normalizePattern(pattern string) string {
	if pattern == "" || strings.HasPrefix(pattern, "/") {
		return pattern
	}
	return "/" + pattern
}

// NewDefinitionService returns a definition service.
func NewDefinitionService() *DefinitionService {
	return &DefinitionService{}
}
