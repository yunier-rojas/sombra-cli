package sombra

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	markerSkip = "sombra:skip"
	markerEnd  = "sombra:end"
)

// stripBlockDirectives removes lines guarded by `sombra:skip` / `sombra:end`
// blocks. Directives are comment agnostic: any line containing the marker is
// treated as a directive. Blocks can be nested; an unbalanced directive is
// reported as an error so files never fail silently.
func stripBlockDirectives(content []byte) ([]byte, error) {
	if !bytes.Contains(content, []byte("sombra:")) {
		return content, nil
	}

	lines := strings.SplitAfter(string(content), "\n")
	var builder strings.Builder
	builder.Grow(len(content))
	depth := 0

	for index, line := range lines {
		lineNumber := index + 1
		switch {
		case strings.Contains(line, markerSkip):
			depth++
		case strings.Contains(line, markerEnd):
			if depth == 0 {
				return nil, fmt.Errorf("line %d: %s without a matching %s", lineNumber, markerEnd, markerSkip)
			}
			depth--
		default:
			if depth == 0 {
				builder.WriteString(line)
			}
		}
	}

	if depth != 0 {
		return nil, fmt.Errorf("%s without a matching %s", markerSkip, markerEnd)
	}

	return []byte(builder.String()), nil
}
