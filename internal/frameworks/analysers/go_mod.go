package analysers

import (
	"bufio"
	"fmt"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"os"
	"path/filepath"
	"strings"
)

type goModAnalyzer struct {
	BaseDir string
	Fn      string
	Module  string
}

// Load reads the go.mod file and extracts the module information and repo details if available.
func (g *goModAnalyzer) Load() error {
	logger.Info("Loading go.mod file")
	filePath := filepath.Join(g.BaseDir, g.Fn)
	file, err := os.Open(filePath)
	if err != nil {
		logger.Error("failed to open go.mod", err)
		return err
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			g.Module = strings.TrimPrefix(line, "module ")
			logger.Info("Module identified: " + g.Module)
			break
		}
	}

	if err = scanner.Err(); err != nil {
		logger.Error("error reading go.mod file", err)
		return err
	}
	logger.Info("Successfully loaded go.mod")
	return nil
}

// GetAbstractCandidates returns abstract mapping candidates for the go.mod configuration.
func (g *goModAnalyzer) GetAbstractCandidates() ([]*entities.AbstractMappingCandidate, error) {
	logger.Info("Getting abstract candidates for go.mod")
	candidates := []*entities.AbstractMappingCandidate{}
	if g.Module != "" {
		candidates = append(candidates, &entities.AbstractMappingCandidate{
			For:      entities.MappingDefault,
			Name:     "project_name",
			Key:      g.Module,
			Value:    "{{ .project_name }}",
			Priority: 0,
		})
	}
	logger.Info("Abstract candidates generated")
	return candidates, nil
}

// GetFileAnalysis returns a file analysis for go.mod
func (g *goModAnalyzer) GetFileAnalysis() ([]*entities.FileAnalysis, error) {
	logger.Info("Performing file analysis on go.mod")
	err := g.Load()
	if err != nil {
		logger.Error("failed to load go.mod", err)
		return nil, err
	}

	modFile := g.getModAnalysis()
	ignore := g.getIgnoreAnalysis()

	logger.Info("File analysis completed")
	return []*entities.FileAnalysis{modFile, ignore}, nil
}

func (g *goModAnalyzer) getModAnalysis() *entities.FileAnalysis {
	modFile := &entities.FileAnalysis{
		Pattern: &entities.Pattern{
			Pattern: entities.Wildcard(g.Fn),
			Content: entities.Mappings{},
		},
		Vars: []string{},
	}

	if g.Module != "" {
		modFile.Pattern.Content[g.Module] = "{{ .project_name }}"
		modFile.Vars = append(modFile.Vars, "project_name")
	}
	return modFile
}

func (g *goModAnalyzer) getIgnoreAnalysis() *entities.FileAnalysis {
	dir := filepath.Dir(g.Fn)
	sumFile := filepath.Join(dir, "go.sum")
	internalDir := filepath.Join(dir, "internal")
	ignoreFiles := &entities.FileAnalysis{
		Pattern: &entities.Pattern{
			Pattern: entities.Wildcard(fmt.Sprintf("%s/**/*", dir)),
		},
		Exclude: []entities.Wildcard{entities.Wildcard(sumFile), entities.Wildcard(internalDir)},
	}

	return ignoreFiles
}

func (g *goModAnalyzer) GetFileName() entities.File {
	return entities.File(g.Fn)
}

// Factory method to create a new GoModAnalyzer instance.
func newGoModAnalyzer(baseDir, fn string) (usecases.LocalFileAnalyserPort, error) {
	logger.Info("Creating new GoModAnalyzer instance")
	obj := &goModAnalyzer{BaseDir: baseDir, Fn: fn}
	err := obj.Load()
	if err != nil {
		logger.Error("failed to create new GoModAnalyzer instance", err)
	}
	return obj, err
}

// Check if the file matches go.mod pattern.
func checkGoModFile(_, fn string) bool {
	logger.Info("Checking if file matches go.mod pattern")
	match, err := pathMatch(fn, "**/go\\.mod")
	if err != nil {
		logger.Error("error while matching path pattern", err)
		return false
	}
	return match
}

var _ usecases.LocalFileAnalyserPort = (*goModAnalyzer)(nil)

func init() {
	// Register the GoMod analyzer during package initialization.
	Register(checkGoModFile, newGoModAnalyzer, 0)
}
