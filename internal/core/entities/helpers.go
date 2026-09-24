package entities

import (
	"fmt"
	"strings"
)

func ConvertToWildcards(strs []string) []Wildcard {
	wildcards := make([]Wildcard, len(strs))
	for i, s := range strs {
		if !strings.HasPrefix(s, "/") {
			s = fmt.Sprintf("/%s", s)
		}
		wildcards[i] = Wildcard(s)
	}
	return wildcards
}

func isIDRune(r rune) bool {
	return r >= 'a' && r <= 'z' ||
		r >= 'A' && r <= 'Z' ||
		r >= '0' && r <= '9' ||
		r == '_' || r == '-'
}

// DeriveID builds a template id from a repository URI using its name. Invalid
// characters are replaced with a dash. It never returns an empty string.
func DeriveID(uri string) string {
	name := strings.TrimRight(uri, "/")
	name = strings.TrimSuffix(name, ".git")
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}

	id := strings.Map(func(r rune) rune {
		if isIDRune(r) {
			return r
		}
		return '-'
	}, name)
	id = strings.Trim(id, "-")
	if id == "" {
		return "template"
	}
	return id
}

// ValidID reports whether id only contains letters, numbers, underscores and
// dashes.
func ValidID(id string) bool {
	if id == "" {
		return false
	}
	return strings.IndexFunc(id, func(r rune) bool { return !isIDRune(r) }) == -1
}

// HasID reports whether a template with the given id is already configured.
func (d *SombraDef) HasID(id string) bool {
	if d == nil || id == "" {
		return false
	}
	for _, template := range d.Templates {
		if template.ID == id {
			return true
		}
	}
	return false
}

// UniqueID returns base when it is free, otherwise it appends a numeric suffix
// (-2, -3, ...) until an unused id is found.
func (d *SombraDef) UniqueID(base string) string {
	if !d.HasID(base) {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !d.HasID(candidate) {
			return candidate
		}
	}
}

// ResolveTemplates returns the templates a reference points to. A reference is
// matched against template ids first and falls back to the URI, so existing
// projects that do not define ids keep working.
func ResolveTemplates(def *SombraDef, ref string) []*TemplateConfig {
	if def == nil || ref == "" {
		return nil
	}

	var byID []*TemplateConfig
	for _, template := range def.Templates {
		if template.ID == ref {
			byID = append(byID, template)
		}
	}
	if len(byID) > 0 {
		return byID
	}

	var byURI []*TemplateConfig
	for _, template := range def.Templates {
		if template.URI == ref {
			byURI = append(byURI, template)
		}
	}
	return byURI
}
