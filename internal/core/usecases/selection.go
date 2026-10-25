package usecases

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

type partEntry struct {
	Part  string
	Entry entities.Entry
}

// normalizeParts removes duplicates and falls back to the required base part
// when no part is selected.
func normalizeParts(parts []string) []string {
	result := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, name := range parts {
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		result = append(result, name)
	}
	if len(result) == 0 {
		return []string{entities.BasePart}
	}
	return result
}

// selectedParts validates a selection against the definition.
func selectedParts(def *entities.TemplateDef, parts []string) ([]string, error) {
	normalized := normalizeParts(parts)
	for _, name := range normalized {
		if _, ok := def.Parts[name]; !ok {
			return nil, entities.Invalid(fmt.Sprintf("unknown part %q (available: %s)", name, strings.Join(partNames(def), ", ")), nil)
		}
	}
	return normalized, nil
}

// activeEntries returns the entries of the selected parts, tagged with the part
// that contributed them.
func activeEntries(def *entities.TemplateDef, parts []string) ([]partEntry, error) {
	selected, err := selectedParts(def, parts)
	if err != nil {
		return nil, err
	}

	var entries []partEntry
	for _, name := range selected {
		for _, entry := range def.Parts[name].Include {
			entries = append(entries, partEntry{Part: name, Entry: entry})
		}
	}
	return entries, nil
}

func partNames(def *entities.TemplateDef) []string {
	names := make([]string, 0, len(def.Parts))
	for name := range def.Parts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// uniqueEntries removes identical entries contributed by more than one part.
// Entries with the same path but different policies are kept so the conflict
// still surfaces as an ambiguity.
func uniqueEntries(entries []partEntry) []partEntry {
	seen := make(map[string]bool, len(entries))
	result := make([]partEntry, 0, len(entries))
	for _, item := range entries {
		key := entryPolicyKey(item.Entry)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, item)
	}
	return result
}

func entryPolicyKey(entry entities.Entry) string {
	rewrite := "default"
	if entry.Rewrite != nil {
		if *entry.Rewrite {
			rewrite = "true"
		} else {
			rewrite = "false"
		}
	}
	return fmt.Sprintf("%s\x00%s\x00%t\x00%s", entry.Path, entry.Mode, entry.Directives, rewrite)
}

// resolveSelection lists every file selected by the active entries and applies
// the global excludes. When a file is listed more than once, the most specific
// entry wins.
func resolveSelection(
	root string,
	def *entities.TemplateDef,
	parts []string,
	lister FileListerPort,
	matcher PathMatcherPort,
) ([]entities.SelectedFile, error) {
	entries, err := activeEntries(def, parts)
	if err != nil {
		return nil, err
	}
	entries = uniqueEntries(entries)

	sources := make(map[entities.File]entities.SourceFile)
	for _, item := range entries {
		files, err := lister.List(root, item.Entry)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			sources[file.Path] = file
		}
	}

	excluded := make(map[entities.File]bool)
	for source := range sources {
		for _, pattern := range def.Exclude {
			match, err := matcher.Match(string(source), pattern)
			if err != nil {
				return nil, err
			}
			if match {
				excluded[source] = true
				break
			}
		}
	}

	selected := make([]entities.SelectedFile, 0, len(sources))
	for source, file := range sources {
		if excluded[source] {
			continue
		}
		entry, err := specificEntry(entries, source)
		if err != nil {
			return nil, err
		}
		rewrite := entry.Entry.Rewrite == nil || *entry.Entry.Rewrite
		selected = append(selected, entities.SelectedFile{
			Source:     source,
			Mode:       modeOrDefault(entry.Entry.Mode),
			Directives: entry.Entry.Directives,
			Rewrite:    rewrite,
			Part:       entry.Part,
			SourceMode: file.Mode,
		})
	}

	sort.Slice(selected, func(i, j int) bool { return selected[i].Source < selected[j].Source })
	return selected, nil
}

// specificEntry returns the most specific entry that selects a source file.
// Exact file entries beat directory entries; longer directories beat shorter.
func specificEntry(entries []partEntry, source entities.File) (partEntry, error) {
	best := -1
	var winner partEntry
	for _, item := range entries {
		if !entrySelects(item.Entry, source) {
			continue
		}
		specificity := len(strings.TrimSuffix(item.Entry.Path, "/"))
		if item.Entry.Path == string(source) {
			specificity += 1_000_000
		}
		if specificity > best {
			best = specificity
			winner = item
			continue
		}
		if specificity == best {
			return partEntry{}, entities.Invalid(fmt.Sprintf("ambiguous entries select %s (parts %q and %q)", source, winner.Part, item.Part), nil)
		}
	}
	if best < 0 {
		return partEntry{}, entities.Invalid(fmt.Sprintf("no entry selects %s", source), nil)
	}
	return winner, nil
}

func entrySelects(entry entities.Entry, source entities.File) bool {
	value := string(source)
	if entry.Path == "/" {
		return true
	}
	if strings.HasSuffix(entry.Path, "/") {
		return strings.HasPrefix(value, entry.Path)
	}
	return value == entry.Path
}

func modeOrDefault(mode entities.Mode) entities.Mode {
	if mode == "" {
		return entities.ModeSync
	}
	return mode
}

// resolveDestination applies path and name rewrites and validates the result.
func resolveDestination(
	source entities.File,
	rules []entities.Rewrite,
	transformer TransformerPort,
	matcher PathMatcherPort,
) (entities.File, error) {
	dir, name := path.Split(string(source))

	pathRules, err := rulesFor(rules, entities.ScopePath, source, matcher)
	if err != nil {
		return "", err
	}
	nameRules, err := rulesFor(rules, entities.ScopeName, source, matcher)
	if err != nil {
		return "", err
	}

	newDir, err := transformer.Text(dir, pathRules)
	if err != nil {
		return "", entities.Invalid(fmt.Sprintf("%s: rewrite path", source), err)
	}
	newName, err := transformer.Text(name, nameRules)
	if err != nil {
		return "", entities.Invalid(fmt.Sprintf("%s: rewrite name", source), err)
	}

	destination := newDir + "/" + newName
	if hasParentSegment(destination) {
		return "", entities.Invalid(fmt.Sprintf("%s: rewrite escapes the target: %s", source, destination), nil)
	}
	destination = path.Join(newDir, newName)
	if destination == "" || destination == "." {
		return "", entities.Invalid(fmt.Sprintf("%s: rewrite produced an empty path", source), nil)
	}
	destination = strings.TrimPrefix(destination, "/")
	if hasParentSegment(destination) {
		return "", entities.Invalid(fmt.Sprintf("%s: rewrite escapes the target: %s", source, destination), nil)
	}
	cleaned := path.Clean(destination)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", entities.Invalid(fmt.Sprintf("%s: rewrite escapes the target: %s", source, destination), nil)
	}
	return entities.File(cleaned), nil
}

func hasParentSegment(value string) bool {
	for _, segment := range strings.Split(value, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

// rulesFor filters rewrites by scope and by the source path.
func rulesFor(
	rules []entities.Rewrite,
	scope entities.Scope,
	source entities.File,
	matcher PathMatcherPort,
) ([]entities.Rewrite, error) {
	result := make([]entities.Rewrite, 0, len(rules))
	for _, rule := range rules {
		in := rule.In
		if len(in) == 0 {
			in = []entities.Scope{entities.ScopeContent}
		}
		applies := false
		for _, candidate := range in {
			if candidate == scope || candidate == "all" {
				applies = true
				break
			}
		}
		if !applies {
			continue
		}
		matches, err := ruleMatchesSource(rule, source, matcher)
		if err != nil {
			return nil, err
		}
		if !matches {
			continue
		}
		result = append(result, rule)
	}
	return result, nil
}

func ruleMatchesSource(rule entities.Rewrite, source entities.File, matcher PathMatcherPort) (bool, error) {
	if len(rule.Files) == 0 {
		return true, nil
	}
	for _, pattern := range rule.Files {
		match, err := matcher.Match(string(source), pattern)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}
