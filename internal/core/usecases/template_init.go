package usecases

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

type TemplateInitCase interface {
	TemplateInit(dir string, opts entities.TemplateInitOptions) (*entities.TemplateInitResult, error)
}

type DirectoryTemplateInitInteractor struct {
	workspace  WorkspacePort
	definition DefinitionPort
	matcher    PathMatcherPort
}

func NewDirectoryTemplateInitInteractor(
	workspace WorkspacePort,
	definition DefinitionPort,
	matcher PathMatcherPort,
) *DirectoryTemplateInitInteractor {
	return &DirectoryTemplateInitInteractor{
		workspace:  workspace,
		definition: definition,
		matcher:    matcher,
	}
}

func (t *DirectoryTemplateInitInteractor) TemplateInit(dir string, opts entities.TemplateInitOptions) (*entities.TemplateInitResult, error) {
	all, err := t.workspace.ListFiles(dir, opts.IncludeUntracked)
	if err != nil {
		return nil, err
	}
	if len(all) == 0 {
		return nil, entities.NotFound(fmt.Sprintf("%s: no files found", dir), nil)
	}

	files, err := t.applyExcludes(all, opts.Exclude)
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, entities.NotFound(fmt.Sprintf("%s: no files remain after exclude patterns", dir), nil)
	}

	var selected []entities.File
	if len(opts.Only) > 0 {
		selected, err = t.applyOnly(files, opts.Only)
		if err != nil {
			return nil, err
		}
	} else {
		selected = files
	}

	entries := minimalEntries(files, selected)
	identity := t.workspace.Identity(dir)
	def := buildDefinition(entries, identity, opts.Exclude)

	file := t.definition.GetFile(dir)
	exists, err := t.definition.Exists(file)
	if err != nil {
		return nil, err
	}
	if exists && !opts.Force && !opts.DryRun {
		return nil, entities.Invalid(fmt.Sprintf("%s already exists; pass --force to overwrite", file), nil)
	}

	result := &entities.TemplateInitResult{
		Definition: def,
		Path:       file,
		DryRun:     opts.DryRun,
		Warnings:   identityWarnings(identity),
	}

	data, err := t.definition.Marshal(def)
	if err != nil {
		return nil, err
	}
	result.YAML = data

	if !opts.DryRun {
		if err := t.definition.Save(file, def); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (t *DirectoryTemplateInitInteractor) applyExcludes(files []entities.File, patterns []string) ([]entities.File, error) {
	if len(patterns) == 0 {
		return files, nil
	}

	result := make([]entities.File, 0, len(files))
	for _, file := range files {
		excluded := false
		for _, pattern := range patterns {
			match, err := t.matcher.Match(string(file), pattern)
			if err != nil {
				return nil, err
			}
			if match {
				excluded = true
				break
			}
		}
		if !excluded {
			result = append(result, file)
		}
	}
	return result, nil
}

func (t *DirectoryTemplateInitInteractor) applyOnly(files []entities.File, patterns []string) ([]entities.File, error) {
	matchedPattern := make([]bool, len(patterns))
	var result []entities.File

	for _, file := range files {
		for index, pattern := range patterns {
			match, err := t.matcher.Match(string(file), pattern)
			if err != nil {
				return nil, err
			}
			if match {
				matchedPattern[index] = true
				result = append(result, file)
				break
			}
		}
	}

	for index, matched := range matchedPattern {
		if !matched {
			return nil, entities.NotFound(fmt.Sprintf("--only %q matched no files", patterns[index]), nil)
		}
	}
	if len(result) == 0 {
		return nil, entities.NotFound("--only matched no files", nil)
	}
	return result, nil
}

// minimalEntries reduces a selected file set to exact files and directory
// prefixes. A top-level directory is used only when every file below it is
// selected.
func minimalEntries(all, selected []entities.File) []entities.Entry {
	allByTop := make(map[string]int)
	for _, file := range all {
		allByTop[topLevel(file)]++
	}

	selectedByTop := make(map[string][]entities.File)
	for _, file := range selected {
		selectedByTop[topLevel(file)] = append(selectedByTop[topLevel(file)], file)
	}

	seen := make(map[string]bool)
	var entries []entities.Entry

	for top, files := range selectedByTop {
		if top == "" {
			for _, file := range files {
				key := string(file)
				if !seen[key] {
					seen[key] = true
					entries = append(entries, entities.Entry{Path: key})
				}
			}
			continue
		}
		if len(files) == allByTop[top] {
			key := "/" + top + "/"
			if !seen[key] {
				seen[key] = true
				entries = append(entries, entities.Entry{Path: key})
			}
			continue
		}
		for _, file := range files {
			key := string(file)
			if !seen[key] {
				seen[key] = true
				entries = append(entries, entities.Entry{Path: key})
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries
}

func topLevel(file entities.File) string {
	value := strings.TrimPrefix(string(file), "/")
	if index := strings.Index(value, "/"); index >= 0 {
		return value[:index]
	}
	return ""
}

func buildDefinition(entries []entities.Entry, identity entities.Identity, exclude []string) *entities.TemplateDef {
	def := &entities.TemplateDef{
		Schema: 2,
		Parts: map[string]entities.Part{
			entities.BasePart: {Include: entries},
		},
		Exclude: append([]string(nil), exclude...),
		Vars: map[string]entities.VarSpec{
			"project":    {Name: "project", Description: "Project and command name", Default: identity.Project},
			"module":     {Name: "module", Description: "Go module path", Default: identity.Module},
			"repository": {Name: "repository", Description: "Repository slug, usually owner/repo", Default: identity.Repository},
			"author":     {Name: "author", Description: "Author name", Default: identity.Author},
		},
	}

	type candidate struct {
		find     string
		variable string
		as       string
		ignore   bool
		scopes   []entities.Scope
	}
	var candidates []candidate

	if identity.Module != "" {
		candidates = append(candidates, candidate{
			find:     identity.Module,
			variable: "module",
			scopes:   []entities.Scope{entities.ScopePath, entities.ScopeName, entities.ScopeContent},
		})
	}
	if identity.Repository != "" {
		candidates = append(candidates, candidate{
			find:     identity.Repository,
			variable: "repository",
			scopes:   []entities.Scope{entities.ScopePath, entities.ScopeName, entities.ScopeContent},
		})
	}
	if identity.Project != "" {
		candidates = append(candidates, candidate{
			find:     identity.Project,
			variable: "project",
			as:       "word",
			ignore:   true,
			scopes:   []entities.Scope{entities.ScopePath, entities.ScopeName, entities.ScopeContent},
		})
	}
	if identity.Author != "" {
		candidates = append(candidates, candidate{
			find:     identity.Author,
			variable: "author",
			scopes:   []entities.Scope{entities.ScopeContent},
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool { return len(candidates[i].find) > len(candidates[j].find) })

	for _, item := range candidates {
		replace := "{{ ." + item.variable + " }}"
		def.Rewrites = append(def.Rewrites, entities.Rewrite{
			Find:       item.find,
			As:         item.as,
			IgnoreCase: item.ignore,
			Replace:    &replace,
			In:         item.scopes,
		})
	}
	return def
}

func identityWarnings(identity entities.Identity) []string {
	var warnings []string
	if identity.Module == "" {
		warnings = append(warnings, "no go.mod module detected; module rewrite not generated")
	}
	if identity.Repository == "" {
		warnings = append(warnings, "no git origin remote detected; repository rewrite not generated")
	}
	if identity.Author == "" {
		warnings = append(warnings, "no git user.name detected; author rewrite not generated")
	}
	return warnings
}

var _ TemplateInitCase = (*DirectoryTemplateInitInteractor)(nil)
