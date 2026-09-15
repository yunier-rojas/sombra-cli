package vars

import (
	"bufio"
	"fmt"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"os"
	"strings"
)

type FileReader struct {
	file *os.File
}

func (l *FileReader) GetValues(vars []string) *entities.Mappings {
	reader := bufio.NewReader(l.file)
	mappings := make(entities.Mappings)
	for _, varName := range vars {
		fmt.Printf("Enter value for %s: ", varName)
		text, _ := reader.ReadString('\n')
		mappings[varName] = strings.Trim(text, "\n")
	}
	return &mappings
}

func NewReader() *FileReader {
	return &FileReader{
		file: os.Stdin,
	}
}

var _ usecases.VariableReaderPort = (*FileReader)(nil)
