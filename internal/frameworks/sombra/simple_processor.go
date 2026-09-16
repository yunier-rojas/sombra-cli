package sombra

import (
	"bytes"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"regexp"
	"strings"
)

type directiveKind int

const (
	directiveLiteral directiveKind = iota
	directiveRegex
	directiveWord
	directiveCaseInsensitive
	directiveWordCaseInsensitive
	directiveJSON
	directiveYAML
	directiveINI
	directiveRex
	directiveFile
)

type directive struct {
	kind    directiveKind
	payload string
}

func (d directive) caseInsensitive() bool {
	return d.kind == directiveCaseInsensitive || d.kind == directiveWordCaseInsensitive
}

func (d directive) tokenAware() bool {
	return d.kind == directiveWord || d.kind == directiveWordCaseInsensitive
}

func (d directive) wholeFileOnly() bool {
	switch d.kind {
	case directiveJSON, directiveYAML, directiveINI, directiveFile:
		return true
	default:
		return false
	}
}

// parseDirective splits a mapping key into a directive and its payload. Keys
// without a recognized prefix are treated as literal strings, including keys
// that embed a colon such as `https://example.com`.
func parseDirective(key string) directive {
	prefix, payload, found := strings.Cut(key, ":")
	if !found {
		return directive{kind: directiveLiteral, payload: key}
	}

	switch prefix {
	case "re":
		return directive{kind: directiveRegex, payload: payload}
	case "rex":
		return directive{kind: directiveRex, payload: payload}
	case "word":
		return directive{kind: directiveWord, payload: payload}
	case "ci":
		return directive{kind: directiveCaseInsensitive, payload: payload}
	case "word-ci":
		return directive{kind: directiveWordCaseInsensitive, payload: payload}
	case "json":
		return directive{kind: directiveJSON, payload: payload}
	case "yaml":
		return directive{kind: directiveYAML, payload: payload}
	case "ini":
		return directive{kind: directiveINI, payload: payload}
	case "file":
		return directive{kind: directiveFile, payload: payload}
	default:
		return directive{kind: directiveLiteral, payload: key}
	}
}

type Processor struct {
}

func (l *Processor) ProcessString(target string, mapping entities.MapList) string {
	var old, replacement string
	for _, item := range mapping {
		old = item.Key
		replacement = item.Value
		target = l.stringReplace(target, old, replacement)
	}
	return target
}

func (l *Processor) ProcessContent(content []byte, mapping entities.MapList) []byte {
	for _, item := range mapping {
		if parseDirective(item.Key).wholeFileOnly() {
			// whole-file directives only apply to complete files (see ProcessFile)
			continue
		}
		updated, err := l.applyContentMapping(content, item.Key, item.Value)
		if err != nil {
			// diff fragments are best effort; leave the content untouched
			continue
		}
		content = updated
	}
	return content
}

// ProcessFile applies whole-file transformations: when blockDirectives is
// enabled, in-file block directives are evaluated first, then each mapping is
// applied in order. Structured directives (json:, yaml:, ini:) and `rex:`
// require a complete file and are handled here rather than in ProcessContent,
// which is also used for diff fragments. Block directives are opt-in per
// pattern so arbitrary files that merely contain `sombra:` text can be copied
// untouched. Binary files are copied byte-for-byte so regex and structured
// mappings cannot corrupt them.
func (l *Processor) ProcessFile(content []byte, mapping entities.MapList, _ entities.Mappings, blockDirectives bool) ([]byte, error) {
	if isBinary(content) {
		return content, nil
	}

	if blockDirectives {
		stripped, err := stripBlockDirectives(content)
		if err != nil {
			return nil, err
		}
		content = stripped
	}

	var err error
	for _, item := range mapping {
		content, err = l.applyContentMapping(content, item.Key, item.Value)
		if err != nil {
			return nil, err
		}
	}

	return content, nil
}

// binarySniffLength is the number of leading bytes inspected when deciding
// whether a file is binary.
const binarySniffLength = 8000

// isBinary reports whether content looks binary. It mirrors Git's heuristic of
// scanning the first bytes for a NUL byte.
func isBinary(content []byte) bool {
	limit := len(content)
	if limit > binarySniffLength {
		limit = binarySniffLength
	}
	return bytes.IndexByte(content[:limit], 0) >= 0
}

// applyContentMapping dispatches a single content mapping. Directives that can
// fail return the error to whole-file callers; the diff path ignores it.
func (l *Processor) applyContentMapping(content []byte, old string, replacement string) ([]byte, error) {
	switch d := parseDirective(old); d.kind {
	case directiveRegex:
		return regexp.MustCompile(d.payload).ReplaceAll(content, ([]byte)(replacement)), nil
	case directiveRex:
		return replaceRex(content, d.payload, replacement)
	case directiveWord, directiveCaseInsensitive, directiveWordCaseInsensitive:
		return []byte(replaceTokens(string(content), d, replacement)), nil
	case directiveJSON:
		return patchJSON(content, d.payload, replacement)
	case directiveYAML:
		return patchYAML(content, d.payload, replacement)
	case directiveINI:
		return patchINI(content, d.payload, replacement)
	case directiveFile:
		// `file:` ignores the previous content and replaces the whole file
		return []byte(replacement), nil
	default:
		// looks like `old` is not a directive, let's do a string replace then
		return bytes.ReplaceAll(content, ([]byte)(old), ([]byte)(replacement)), nil
	}
}

func (l *Processor) stringReplace(content string, old string, replacement string) string {
	d := parseDirective(old)
	switch d.kind {
	case directiveRegex:
		prev := regexp.MustCompile(d.payload)
		return prev.ReplaceAllString(content, replacement)
	case directiveWord, directiveCaseInsensitive, directiveWordCaseInsensitive:
		return replaceTokens(content, d, replacement)
	case directiveRex:
		result, err := replaceRex([]byte(content), d.payload, replacement)
		if err != nil {
			return content
		}
		return string(result)
	case directiveJSON, directiveYAML, directiveINI, directiveFile:
		// whole-file directives do not apply to paths or names
		return content
	default:
		// looks like `old` is not a directive, let's do a string replace then
		return strings.ReplaceAll(content, old, replacement)
	}
}

// replaceTokens rewrites every literal occurrence of the payload that is
// allowed by the directive. Replacement is performed by index so the
// replacement string is never interpreted as a regexp template, which keeps
// values containing `$` safe. Boundaries are checked without consuming
// characters, so adjacent tokens are replaced too.
func replaceTokens(content string, d directive, replacement string) string {
	if d.payload == "" {
		return content
	}

	expr := regexp.QuoteMeta(d.payload)
	if d.caseInsensitive() {
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
		if d.tokenAware() && !hasTokenBoundary(content, start, end) {
			continue
		}
		builder.WriteString(content[last:start])
		builder.WriteString(replacement)
		last = end
	}
	builder.WriteString(content[last:])

	return builder.String()
}

// hasTokenBoundary reports whether content[start:end] is delimited by
// characters that are not part of a token.
func hasTokenBoundary(content string, start, end int) bool {
	if start > 0 && isTokenCharacter(content[start-1]) {
		return false
	}
	if end < len(content) && isTokenCharacter(content[end]) {
		return false
	}
	return true
}

// isTokenCharacter defines what counts as part of a token for the `word:`
// directive. Letters, digits, `_`, `.` and `-` are token characters so that
// derived names such as `app-cli`, `app_cli` and `app.yaml` are left untouched.
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

func NewProcessor() *Processor {
	return &Processor{}
}

var _ usecases.SombraStringsPort = (*Processor)(nil)
