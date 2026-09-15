package sombra

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig/v3"
)

// rexData is the template context available to a `rex:` replacement.
type rexData struct {
	// Match holds the named and numbered submatches, keyed by name or by
	// number as a string ("0" is the whole match).
	Match map[string]string
	// Value is the whole match.
	Value string
	// Groups holds the numbered submatches, Groups[0] being the whole match.
	Groups []string
}

// replaceRex rewrites every match of pattern using replacement as a Go template
// evaluated once per match, with the submatches exposed as rexData.
//
// The template definition itself is a Go template using `{{ }}`, so a `rex:`
// replacement would be consumed before it could see match data. `rex:` values
// therefore use `[[ ]]` delimiters, which survive the definition render and are
// evaluated here. Sprig functions are available.
func replaceRex(content []byte, pattern string, replacement string) ([]byte, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("rex: %w", err)
	}

	expression, err := template.New("rex").Funcs(sprig.FuncMap()).Delims("[[", "]]").Parse(replacement)
	if err != nil {
		return nil, fmt.Errorf("rex: %w", err)
	}

	matches := re.FindAllSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	var builder bytes.Buffer
	builder.Grow(len(content))
	last := 0
	for _, match := range matches {
		builder.Write(content[last:match[0]])
		if err := expression.Execute(&builder, newRexData(content, re, match)); err != nil {
			return nil, fmt.Errorf("rex: %w", err)
		}
		last = match[1]
	}
	builder.Write(content[last:])

	return builder.Bytes(), nil
}

func newRexData(content []byte, re *regexp.Regexp, match []int) rexData {
	names := re.SubexpNames()
	groups := make([]string, 0, len(match)/2)
	values := make(map[string]string, len(match)/2)

	for i := 0; i*2 < len(match); i++ {
		value := ""
		if match[i*2] >= 0 {
			value = string(content[match[i*2]:match[i*2+1]])
		}
		groups = append(groups, value)
		values[strconv.Itoa(i)] = value
		if i < len(names) && names[i] != "" {
			values[names[i]] = value
		}
	}

	return rexData{
		Match:  values,
		Value:  groups[0],
		Groups: groups,
	}
}
