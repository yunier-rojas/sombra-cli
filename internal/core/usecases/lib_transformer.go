package usecases

import "github.com/yunier-rojas/sombra-cli/internal/core/entities"

// TransformerPort applies ordered rewrites to text and file content.
type TransformerPort interface {
	Text(input string, rules []entities.Rewrite) (string, error)
	Content(content []byte, rules []entities.Rewrite, directives bool) ([]byte, error)
}
