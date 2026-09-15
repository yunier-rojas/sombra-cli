package sombra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Structured edit directives update a single value addressed by a path while
// leaving the rest of the document untouched. The path is a dot separated list
// of object keys and array indexes, for example `json:scripts.test` or
// `yaml:jobs.0.runs-on`. A path that cannot be resolved is an error. Empty
// values are treated as a null/empty value.

func patchJSON(content []byte, path string, replacement string) ([]byte, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, fmt.Errorf("json directive requires a path")
	}

	segments := strings.Split(trimmed, ".")
	start, end, found, err := locateJSON(content, segments)
	if err != nil {
		return nil, fmt.Errorf("json path %q: %w", path, err)
	}
	if !found {
		return nil, fmt.Errorf("json path %q not found", path)
	}

	encoded, err := encodeJSONValue(replacement)
	if err != nil {
		return nil, fmt.Errorf("json path %q: %w", path, err)
	}

	result := make([]byte, 0, len(content)-(end-start)+len(encoded))
	result = append(result, content[:start]...)
	result = append(result, encoded...)
	result = append(result, content[end:]...)

	return result, nil
}

// locateJSON returns the byte range of the value at path within content. All
// offsets are absolute within content, which lets patchJSON splice the new
// value in place and preserve unrelated formatting byte for byte.
func locateJSON(content []byte, path []string) (int, int, bool, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()

	token, err := decoder.Token()
	if err != nil {
		return 0, 0, false, err
	}

	delimiter, ok := token.(json.Delim)
	if !ok {
		return 0, 0, false, fmt.Errorf("expected an object or array")
	}

	switch delimiter {
	case '{':
		return locateJSONObject(content, path)
	case '[':
		return locateJSONArray(content, path)
	default:
		return 0, 0, false, fmt.Errorf("expected an object or array")
	}
}

func locateJSONObject(content []byte, path []string) (int, int, bool, error) {
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()

	if _, err := decoder.Token(); err != nil {
		return 0, 0, false, err
	}

	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return 0, 0, false, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return 0, 0, false, fmt.Errorf("expected an object key")
		}

		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return 0, 0, false, err
		}

		if key != path[0] {
			continue
		}

		valueStart := int(decoder.InputOffset()) - len(raw)
		if len(path) == 1 {
			return valueStart, int(decoder.InputOffset()), true, nil
		}

		start, end, found, err := locateJSON(raw, path[1:])
		if err != nil || !found {
			return 0, 0, false, err
		}
		return valueStart + start, valueStart + end, true, nil
	}

	return 0, 0, false, nil
}

func locateJSONArray(content []byte, path []string) (int, int, bool, error) {
	index, err := strconv.Atoi(path[0])
	if err != nil {
		return 0, 0, false, fmt.Errorf("invalid array index %q", path[0])
	}

	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.UseNumber()

	if _, err := decoder.Token(); err != nil {
		return 0, 0, false, err
	}

	for i := 0; decoder.More(); i++ {
		var raw json.RawMessage
		if err := decoder.Decode(&raw); err != nil {
			return 0, 0, false, err
		}

		if i != index {
			continue
		}

		valueStart := int(decoder.InputOffset()) - len(raw)
		if len(path) == 1 {
			return valueStart, int(decoder.InputOffset()), true, nil
		}

		start, end, found, err := locateJSON(raw, path[1:])
		if err != nil || !found {
			return 0, 0, false, err
		}
		return valueStart + start, valueStart + end, true, nil
	}

	return 0, 0, false, nil
}

func encodeJSONValue(value string) ([]byte, error) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &raw); err == nil {
			return raw, nil
		}
	}
	return json.Marshal(value)
}

func patchYAML(content []byte, path string, replacement string) ([]byte, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, fmt.Errorf("yaml directive requires a path")
	}

	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, err
	}

	node, err := findYAMLNode(&document, strings.Split(trimmed, "."))
	if err != nil {
		return nil, err
	}
	if err := replaceYAMLNode(node, replacement); err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(detectYAMLIndent(content))
	if err := encoder.Encode(&document); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func findYAMLNode(node *yaml.Node, path []string) (*yaml.Node, error) {
	if len(path) == 0 {
		return node, nil
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil, fmt.Errorf("yaml document is empty")
		}
		return findYAMLNode(node.Content[0], path)
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == path[0] {
				return findYAMLNode(node.Content[i+1], path[1:])
			}
		}
		return nil, fmt.Errorf("yaml path %q not found", strings.Join(path, "."))
	case yaml.SequenceNode:
		index, err := strconv.Atoi(path[0])
		if err != nil || index < 0 || index >= len(node.Content) {
			return nil, fmt.Errorf("invalid yaml index %q", path[0])
		}
		return findYAMLNode(node.Content[index], path[1:])
	default:
		return nil, fmt.Errorf("yaml path %q not found", strings.Join(path, "."))
	}
}

func replaceYAMLNode(target *yaml.Node, replacement string) error {
	if strings.TrimSpace(replacement) == "" {
		target.Kind = yaml.ScalarNode
		target.Tag = "!!null"
		target.Value = ""
		target.Content = nil
		target.Style = 0
		return nil
	}

	var document yaml.Node
	if err := yaml.Unmarshal([]byte(replacement), &document); err != nil {
		return err
	}

	value := &document
	if document.Kind == yaml.DocumentNode && len(document.Content) > 0 {
		value = document.Content[0]
	}

	target.Kind = value.Kind
	target.Tag = value.Tag
	target.Value = value.Value
	target.Content = value.Content
	target.Style = value.Style

	return nil
}

// detectYAMLIndent guesses the indentation used by content so re-encoding keeps
// the file close to its original layout. yaml.v3 defaults to four spaces, which
// would otherwise reformat two-space files.
func detectYAMLIndent(content []byte) int {
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" || trimmed == line {
			continue
		}
		if indent := len(line) - len(trimmed); indent > 0 && indent <= 8 {
			return indent
		}
	}
	return 2
}

func patchINI(content []byte, path string, replacement string) ([]byte, error) {
	section, key, err := parseINIPath(path)
	if err != nil {
		return nil, err
	}

	lines := strings.SplitAfter(string(content), "\n")
	currentSection := ""
	for index, line := range lines {
		body, ending := splitLineEnding(line)
		trimmed := strings.TrimSpace(body)

		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSection = trimmed[1 : len(trimmed)-1]
			continue
		}
		if currentSection != section {
			continue
		}

		separator := iniSeparator(body)
		if separator == -1 || strings.TrimSpace(body[:separator]) != key {
			continue
		}

		prefix := strings.TrimRight(body[:separator+1], " \t")
		lines[index] = prefix + " " + replacement + ending
		return []byte(strings.Join(lines, "")), nil
	}

	return nil, fmt.Errorf("ini path %q not found", path)
}

func parseINIPath(path string) (string, string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", "", fmt.Errorf("ini directive requires a path")
	}
	if index := strings.LastIndex(trimmed, "."); index != -1 {
		return trimmed[:index], trimmed[index+1:], nil
	}
	return "", trimmed, nil
}

func iniSeparator(line string) int {
	equals := strings.Index(line, "=")
	colon := strings.Index(line, ":")
	switch {
	case equals == -1 && colon == -1:
		return -1
	case equals == -1:
		return colon
	case colon == -1:
		return equals
	case equals < colon:
		return equals
	default:
		return colon
	}
}

func splitLineEnding(line string) (string, string) {
	switch {
	case strings.HasSuffix(line, "\r\n"):
		return line[:len(line)-2], "\r\n"
	case strings.HasSuffix(line, "\n"):
		return line[:len(line)-1], "\n"
	default:
		return line, ""
	}
}
