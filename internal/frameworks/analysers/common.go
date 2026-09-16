package analysers

import (
	"github.com/bmatcuk/doublestar/v4"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"strings"
)

func pathMatch(fn string, pattern string) (bool, error) {
	if !strings.HasPrefix(pattern, "/") {
		pattern = "**/" + pattern
	}
	match, err := doublestar.PathMatch(pattern, fn)
	if err != nil {
		logger.Error("error while matching path pattern", err)
		return false, err
	}
	return match, nil
}
