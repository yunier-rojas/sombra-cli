package directives

import (
	"fmt"
	"regexp"
	"strings"
)

var marker = regexp.MustCompile(`sombra:([A-Za-z0-9_-]+)`)

// Block is one parsed directive block.
type Block struct {
	Name string
	Arg  string
	// Lines are the lines between the opening and closing markers.
	Lines []string
}

// Handler transforms a block. Returning nil removes it.
type Handler func(block Block) ([]string, error)

// Registry maps directive names to handlers and parses the in-file syntax.
type Registry struct {
	handlers map[string]Handler
}

// NewRegistry returns the built-in registry. Only `skip` is enabled in v1.
func NewRegistry() *Registry {
	return &Registry{
		handlers: map[string]Handler{
			"skip": func(Block) ([]string, error) { return nil, nil },
		},
	}
}

// Apply evaluates every block whose name has a registered handler. Unknown
// directives are preserved byte for byte. Markers are comment agnostic: any
// line containing the marker is a marker line.
func (r *Registry) Apply(content []byte) ([]byte, error) {
	if !strings.Contains(string(content), "sombra:") {
		return content, nil
	}

	lines := strings.SplitAfter(string(content), "\n")
	frames := make([]*frame, 0)
	var output []string

	for index, line := range lines {
		lineNumber := index + 1
		match := marker.FindStringSubmatch(line)
		if match == nil {
			if len(frames) == 0 {
				output = append(output, line)
			} else {
				frames[len(frames)-1].lines = append(frames[len(frames)-1].lines, line)
			}
			continue
		}

		name := match[1]
		arg := strings.TrimSpace(strings.TrimPrefix(line[matchIndex(line):], match[0]))

		if name == "end" {
			if len(frames) == 0 {
				return nil, fmt.Errorf("line %d: sombra:end without an open directive", lineNumber)
			}
			current := frames[len(frames)-1]
			frames = frames[:len(frames)-1]
			current.end = line

			next, err := r.flush(current)
			if err != nil {
				return nil, err
			}
			if len(frames) == 0 {
				output = append(output, next...)
			} else {
				frames[len(frames)-1].lines = append(frames[len(frames)-1].lines, next...)
			}
			continue
		}

		frames = append(frames, &frame{
			name:    name,
			arg:     arg,
			opener:  line,
			lineNum: lineNumber,
		})
	}

	if len(frames) > 0 {
		current := frames[len(frames)-1]
		return nil, fmt.Errorf("line %d: sombra:%s without a matching sombra:end", current.lineNum, current.name)
	}

	return []byte(strings.Join(output, "")), nil
}

type frame struct {
	name    string
	arg     string
	lines   []string
	opener  string
	end     string
	lineNum int
}

func (r *Registry) flush(current *frame) ([]string, error) {
	handler, ok := r.handlers[current.name]
	if !ok {
		result := make([]string, 0, len(current.lines)+2)
		result = append(result, current.opener)
		result = append(result, current.lines...)
		result = append(result, current.end)
		return result, nil
	}

	result, err := handler(Block{Name: current.name, Arg: current.arg, Lines: current.lines})
	if err != nil {
		return nil, fmt.Errorf("line %d: sombra:%s: %w", current.lineNum, current.name, err)
	}
	return result, nil
}

// matchIndex returns the byte offset of the marker in a line.
func matchIndex(line string) int {
	index := strings.Index(line, "sombra:")
	if index < 0 {
		return 0
	}
	return index
}
