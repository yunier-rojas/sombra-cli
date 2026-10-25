package sombra

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/sombra/directives"
)

// Transformer applies ordered rewrites to text and file content.
type Transformer struct {
	directives *directives.Registry
}

var _ usecases.TransformerPort = (*Transformer)(nil)

// NewTransformer returns a transformer with the built-in directive registry.
func NewTransformer() *Transformer {
	return &Transformer{directives: directives.NewRegistry()}
}

// Text applies path or name rewrites in declaration order.
func (t *Transformer) Text(input string, rules []entities.Rewrite) (string, error) {
	var err error
	for _, rule := range rules {
		input, err = applyText(input, rule)
		if err != nil {
			return "", entities.Invalid("rewrite "+rule.Find, err)
		}
	}
	return input, nil
}

// Content applies directives and then content rewrites in declaration order.
func (t *Transformer) Content(content []byte, rules []entities.Rewrite, enabled bool) ([]byte, error) {
	if isBinary(content) {
		return content, nil
	}

	if enabled {
		stripped, err := t.directives.Apply(content)
		if err != nil {
			return nil, entities.Invalid("apply directives", err)
		}
		content = stripped
	}

	var err error
	for _, rule := range rules {
		content, err = applyContent(content, rule)
		if err != nil {
			return nil, entities.Invalid("rewrite "+rule.Find, err)
		}
	}
	return content, nil
}

func applyText(input string, rule entities.Rewrite) (string, error) {
	replacement := ""
	if rule.Replace != nil {
		replacement = *rule.Replace
	}

	switch rule.As {
	case "", "literal":
		if rule.IgnoreCase {
			expr := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(rule.Find))
			return expr.ReplaceAllStringFunc(input, func(string) string { return replacement }), nil
		}
		return strings.ReplaceAll(input, rule.Find, replacement), nil
	case "word":
		return replaceTokens(input, rule, replacement), nil
	case "regex":
		expr, err := regexp.Compile(rule.Find)
		if err != nil {
			return "", fmt.Errorf("compile %q: %w", rule.Find, err)
		}
		return expr.ReplaceAllString(input, replacement), nil
	case "json", "yaml", "ini":
		return "", fmt.Errorf("%s rewrites cannot apply to paths or names", rule.As)
	default:
		return "", fmt.Errorf("unsupported rewrite mode %q", rule.As)
	}
}

func applyContent(content []byte, rule entities.Rewrite) ([]byte, error) {
	replacement := ""
	if rule.Replace != nil {
		replacement = *rule.Replace
	}

	switch rule.As {
	case "json":
		return patchJSON(content, rule.Find, replacement)
	case "yaml":
		return patchYAML(content, rule.Find, replacement)
	case "ini":
		return patchINI(content, rule.Find, replacement)
	default:
		updated, err := applyText(string(content), rule)
		if err != nil {
			return nil, err
		}
		return []byte(updated), nil
	}
}

// replaceTokens rewrites literal occurrences that sit on token boundaries.
// Replacement is done by index so the replacement string is never interpreted
// as a regexp template.
func replaceTokens(content string, rule entities.Rewrite, replacement string) string {
	if rule.Find == "" {
		return content
	}

	expr := regexp.QuoteMeta(rule.Find)
	if rule.IgnoreCase {
		expr = "(?i)" + expr
	}

	matches := regexp.MustCompile(expr).FindAllStringIndex(content, -1)
	if len(matches) == 0 {
		return content
	}

	var builder strings.Builder
	last := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		if !hasTokenBoundary(content, start, end) {
			continue
		}
		builder.WriteString(content[last:start])
		builder.WriteString(replacement)
		last = end
	}
	builder.WriteString(content[last:])
	return builder.String()
}

func hasTokenBoundary(content string, start, end int) bool {
	if start > 0 && isTokenCharacter(content[start-1]) {
		return false
	}
	if end < len(content) && isTokenCharacter(content[end]) {
		return false
	}
	return true
}

func isTokenCharacter(char byte) bool {
	switch {
	case char >= 'a' && char <= 'z':
		return true
	case char >= 'A' && char <= 'Z':
		return true
	case char >= '0' && char <= '9':
		return true
	case char == '_', char == '.', char == '-':
		return true
	default:
		return false
	}
}

const binarySniffLength = 8000

func isBinary(content []byte) bool {
	limit := len(content)
	if limit > binarySniffLength {
		limit = binarySniffLength
	}
	return bytes.IndexByte(content[:limit], 0) >= 0
}
