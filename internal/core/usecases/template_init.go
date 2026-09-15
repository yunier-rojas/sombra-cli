package usecases

import (
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"sort"
)

type LocalFileAnalyserPort interface {
	GetAbstractCandidates() ([]*entities.AbstractMappingCandidate, error)
	GetFileAnalysis() ([]*entities.FileAnalysis, error)
	GetFileName() entities.File
}

type FileAnalyserRegistryPort interface {
	GetAnalyser(baseDir string, file entities.File) (LocalFileAnalyserPort, error)
}

type TemplateInitCase interface {
	TemplateInit(baseDir string, only []entities.Wildcard, exclude []entities.Wildcard) error
}

type DirectoryTemplateInitInteractor struct {
	scanner     DirectoryManagerPort
	registry    FileAnalyserRegistryPort
	templateDef TemplateDefManagerPort
	engine      SombraEngineCase
}

// TemplateInit initializes the repository with the provided template
func (l *DirectoryTemplateInitInteractor) TemplateInit(templateDir string, only []entities.Wildcard, exclude []entities.Wildcard) error {
	tree := l.scanner.ScanTree(templateDir, only, exclude)

	var analysers = make([]LocalFileAnalyserPort, 0)
	var analyser LocalFileAnalyserPort
	var err error
	for result := range tree {
		if result.Err != nil {
			return result.Err
		}
		analyser, err = l.registry.GetAnalyser(templateDir, result.File)
		if err != nil {
			return err
		}
		analysers = append(analysers, analyser)
	}

	abstractMappings, err := l.extractAbstractMappings(analysers, exclude)
	if err != nil {
		return err
	}

	vars, fileMappings, err := l.extractFileMappings(analysers, abstractMappings, only, exclude)
	if err != nil {
		return err
	}
	template, err := l.buildTemplate(vars, fileMappings)
	if err != nil {
		return err
	}

	templateDefFile := l.templateDef.GetFile(templateDir)
	err = l.templateDef.Save(templateDefFile, template)
	if err != nil {
		return err
	}
	return nil
}

func (l *DirectoryTemplateInitInteractor) buildTemplate(strings []string, files []*entities.Pattern) (*entities.TemplateDef, error) {
	// Create a new slice to store the combined patterns
	combinedPatterns := make([]*entities.Pattern, 0)

	combinedPatterns = append(combinedPatterns, files...)

	// Initialize the template definition
	template := &entities.TemplateDef{
		Vars:     l.deduplicateVars(strings), // The variables derived from the process
		Patterns: combinedPatterns,           // Consolidate global and file-specific patterns
	}
	return template, nil
}

// deduplicateVars removes duplicate variables from a list
func (l *DirectoryTemplateInitInteractor) deduplicateVars(input []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

// extractAbstractMappings extracts abstract mapping patterns from files
func (l *DirectoryTemplateInitInteractor) extractAbstractMappings(files []LocalFileAnalyserPort, exclude []entities.Wildcard) ([]*entities.Pattern, error) {
	candidates := make([]*entities.AbstractMappingCandidate, 0)

	for _, file := range files {
		proposed, err := file.GetAbstractCandidates()
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, proposed...)
	}
	return l.unifyAbstractMappings(candidates, exclude), nil
}

// unifyAbstractMappings consolidates and unifies abstract mapping candidates
func (l *DirectoryTemplateInitInteractor) unifyAbstractMappings(candidates []*entities.AbstractMappingCandidate, exclude []entities.Wildcard) []*entities.Pattern {
	varMap := make(map[string]*entities.AbstractMappingCandidate)

	for _, candidate := range candidates {
		name := candidate.Name
		winner, found := varMap[name]

		if !found || (candidate.Priority > winner.Priority) {
			varMap[name] = candidate
		}
	}

	mappings := make(map[entities.MappingType]entities.Mappings)
	for _, candidate := range varMap {
		if _, exists := mappings[candidate.For]; !exists {
			mappings[candidate.For] = make(entities.Mappings)
		}
		mappings[candidate.For][candidate.Key] = candidate.Value
	}

	return []*entities.Pattern{
		{
			Pattern:  "/**/*",
			Abstract: true,
			Default:  mappings[entities.MappingDefault],
			Path:     mappings[entities.MappingPath],
			Name:     mappings[entities.MappingName],
			Content:  mappings[entities.MappingContent],
			Except:   exclude,
		},
	}
}

// extractFileMappings processes individual file mappings and removes duplicates
func (l *DirectoryTemplateInitInteractor) extractFileMappings(files []LocalFileAnalyserPort, wildcardPatterns []*entities.Pattern, only, exclude []entities.Wildcard) ([]string, []*entities.Pattern, error) {
	res := make([]*entities.Pattern, 0)
	vars := []string{}
	ignore := []entities.Wildcard{}
	var err error
	var analysis, allAnalysis []*entities.FileAnalysis

	for _, wildcard := range only {
		wildcardPatterns = append(wildcardPatterns, &entities.Pattern{
			Pattern: wildcard,
			Default: make(entities.Mappings),
			Path:    make(entities.Mappings),
			Name:    make(entities.Mappings),
			Content: make(entities.Mappings),
			Except:  exclude,
		})
	}

	for _, file := range files {
		analysis, err = file.GetFileAnalysis()
		if err != nil {
			return nil, nil, err
		}

		allAnalysis = append(allAnalysis, analysis...)
	}

	for _, analyser := range allAnalysis {
		if len(analyser.Exclude) > 0 {
			ignore = append(ignore, analyser.Exclude...)
		}
		vars = append(vars, analyser.Vars...)
	}

	ignore = append(ignore, exclude...)

	for _, analyser := range allAnalysis {
		if analyser.Pattern == nil {
			continue
		}

		if analyser.IsWildcard {
			wildcardPatterns = append(wildcardPatterns, analyser.Pattern)
		}
	}

	err = l.updateWildcardsWithIgnores(wildcardPatterns, ignore)
	if err != nil {
		return nil, nil, err
	}

	res = append(res, wildcardPatterns...)

	var matching []*entities.Pattern
	var include bool

	for _, analyser := range allAnalysis {
		if analyser.Pattern == nil {
			continue
		}

		pattern := analyser.Pattern

		name := pattern.Pattern
		include, matching, err = l.engine.Match(entities.File(name), wildcardPatterns)
		if err != nil {
			return nil, nil, err
		}

		if !include && !analyser.IsMandatory {
			continue
		}

		l.removeDuplicateMappings(pattern, matching)

		if !l.isRelevantPattern(pattern) && include {
			// Skip if the cleaned mapping has no unique values beyond the wildcard
			continue
		}
		res = append(res, pattern)
	}

	res, err = l.combinePatterns(res)
	if err != nil {
		return nil, nil, err
	}

	return vars, res, nil
}

// removeDuplicateMappings removes redundant mappings between file and global patterns
func (l *DirectoryTemplateInitInteractor) removeDuplicateMappings(pattern *entities.Pattern, globalPatterns []*entities.Pattern) {
	for _, globalPattern := range globalPatterns {
		l.removeMappingForCategory(pattern, globalPattern.Default)
		l.removeMappingForCategory(pattern, globalPattern.Path)
		l.removeMappingForCategory(pattern, globalPattern.Name)
		l.removeMappingForCategory(pattern, globalPattern.Content)
	}
}

// removeMappingForCategory removes redundant mappings from the specified category
func (l *DirectoryTemplateInitInteractor) removeMappingForCategory(targetPattern *entities.Pattern, referenceMappings entities.Mappings) {
	l.removeMappings(targetPattern.Default, referenceMappings)
	l.removeMappings(targetPattern.Path, referenceMappings)
	l.removeMappings(targetPattern.Name, referenceMappings)
	l.removeMappings(targetPattern.Content, referenceMappings)
}

// removeMappings removes matching mappings from a target mapping collection
func (l *DirectoryTemplateInitInteractor) removeMappings(target, reference entities.Mappings) {
	for selector, refValue := range reference {
		if targetValue, exists := target[selector]; exists && targetValue == refValue {
			delete(target, selector)
		}
	}
}

func (l *DirectoryTemplateInitInteractor) combinePatterns(patterns []*entities.Pattern) ([]*entities.Pattern, error) {
	unified := make(map[entities.Wildcard]map[bool]*entities.Pattern)

	for _, pattern := range patterns {
		patternMap, ok := unified[pattern.Pattern]
		if !ok {
			unified[pattern.Pattern] = make(map[bool]*entities.Pattern)
			unified[pattern.Pattern][pattern.Abstract] = pattern
			continue
		}
		patternAbstract, ok := patternMap[pattern.Abstract]
		if !ok {
			patternMap[pattern.Abstract] = pattern
			continue
		}

		l.combineMappings(patternAbstract, pattern)
	}

	res := make([]*entities.Pattern, 0)
	for _, patternMap := range unified {
		for _, pattern := range patternMap {
			res = append(res, pattern)
		}
	}

	sort.Slice(res, func(i, j int) bool {
		if res[i].Abstract && !res[j].Abstract {
			return true
		}
		if !res[i].Abstract && res[j].Abstract {
			return false
		}
		return res[i].Pattern < res[j].Pattern
	})

	return res, nil
}

func (l *DirectoryTemplateInitInteractor) updateWildcardsWithIgnores(globalPatterns []*entities.Pattern, ignore []entities.Wildcard) error {
	var err error
	var matching []*entities.Pattern
	for _, fn := range ignore {
		_, matching, err = l.engine.Match(entities.File(fn), globalPatterns)

		if err != nil {
			return err
		}

		if len(matching) == 0 {
			continue
		}

		for _, global := range matching {
			if global.Abstract {
				continue
			}
			global.Except = append(global.Except, fn)
		}
	}
	return err
}

func (l *DirectoryTemplateInitInteractor) isRelevantPattern(pattern *entities.Pattern) bool {
	if len(pattern.Default) != 0 {
		return true
	}
	if len(pattern.Path) != 0 {
		return true
	}
	if len(pattern.Name) != 0 {
		return true
	}
	if len(pattern.Content) != 0 {
		return true
	}
	if len(pattern.Except) != 0 {
		return true
	}
	return pattern.Verbatim || pattern.CopyOnly || pattern.Abstract || pattern.Replace != ""
}

func (l *DirectoryTemplateInitInteractor) combineMappings(target, source *entities.Pattern) {
	if target.Default == nil {
		target.Default = make(entities.Mappings)
	}
	if target.Path == nil {
		target.Path = make(entities.Mappings)
	}
	if target.Name == nil {
		target.Name = make(entities.Mappings)
	}
	if target.Content == nil {
		target.Content = make(entities.Mappings)
	}
	for k, v := range source.Default {
		target.Default[k] = v
	}
	for k, v := range source.Path {
		target.Path[k] = v
	}
	for k, v := range source.Name {
		target.Name[k] = v
	}
	for k, v := range source.Content {
		target.Content[k] = v
	}
	target.Except = append(target.Except, source.Except...)
	if target.Replace == "" {
		target.Replace = source.Replace
	}
	target.Verbatim = target.Verbatim || source.Verbatim
	target.CopyOnly = target.CopyOnly || source.CopyOnly
	target.Abstract = target.Abstract || source.Abstract
}

func NewDirectoryTemplateInitInteractor(
	scanner DirectoryManagerPort,
	registry FileAnalyserRegistryPort,
	templateDef TemplateDefManagerPort,
	engine SombraEngineCase,
) *DirectoryTemplateInitInteractor {
	return &DirectoryTemplateInitInteractor{
		scanner:     scanner,
		registry:    registry,
		templateDef: templateDef,
		engine:      engine,
	}
}

var _ TemplateInitCase = (*DirectoryTemplateInitInteractor)(nil)
