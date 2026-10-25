package vars

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
)

// FileReader prompts for template variables on stdin.
type FileReader struct {
	input io.Reader
	out   io.Writer
}

func (l *FileReader) GetValues(specs []entities.VarSpec) (entities.Mappings, error) {
	reader := bufio.NewReader(l.input)
	mappings := make(entities.Mappings, len(specs))

	for _, spec := range specs {
		if spec.Description != "" {
			_, _ = fmt.Fprintf(l.out, "%s: ", spec.Description)
		} else {
			_, _ = fmt.Fprintf(l.out, "Enter value for %s: ", spec.Name)
		}
		if spec.Default != "" {
			_, _ = fmt.Fprintf(l.out, "[%s] ", spec.Default)
		}

		text, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, entities.Invalid("read "+spec.Name, err)
		}
		value := strings.TrimRight(text, "\r\n")
		if value == "" {
			value = spec.Default
		}
		if value == "" {
			return nil, entities.Invalid(spec.Name+" requires a value", nil)
		}
		mappings[spec.Name] = value
	}
	return mappings, nil
}

func NewReader() *FileReader {
	return &FileReader{input: os.Stdin, out: os.Stdout}
}

var _ usecases.VariableReaderPort = (*FileReader)(nil)
