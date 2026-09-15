package analysers

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
)

type licenseAnalyser struct {
	BaseDir string
	Fn      string
}

func (l *licenseAnalyser) GetAbstractCandidates() ([]*entities.AbstractMappingCandidate, error) {
	logger.Info("GetAbstractCandidates called")
	return []*entities.AbstractMappingCandidate{}, nil
}

func (l *licenseAnalyser) GetFileAnalysis() ([]*entities.FileAnalysis, error) {
	logger.Info("GetFileAnalysis called with filename: " + l.Fn)
	source := filepath.Dir(l.Fn)
	name := filepath.Base(l.Fn)

	holder := l.computeNewLicenseName()

	fileAnalysis := &entities.FileAnalysis{
		Pattern: &entities.Pattern{
			Pattern: entities.Wildcard(l.Fn),
			Path: entities.Mappings{
				source: "/vendors",
			},
			Name: entities.Mappings{
				name: fmt.Sprintf("%s.%s", holder, name),
			},
			Verbatim: true,
		},
		Vars: []string{},
	}

	return []*entities.FileAnalysis{fileAnalysis}, nil
}

func (l *licenseAnalyser) computeNewLicenseName() string {
	logger.Info("computeNewLicenseName called for base directory: " + l.BaseDir)
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(l.BaseDir+l.Fn)).String()
}

func (l *licenseAnalyser) GetFileName() entities.File {
	logger.Info("GetFileName called, returning: " + l.Fn)
	return entities.File(l.Fn)
}

func newLicenseAnalyser(baseDir, fn string) (usecases.LocalFileAnalyserPort, error) {
	logger.Info("Creating new licenseAnalyser for baseDir: " + baseDir + ", filename: " + fn)
	return &licenseAnalyser{BaseDir: baseDir, Fn: fn}, nil
}

func acceptLicenseFile(_, fn string) bool {
	filename := filepath.Base(fn)
	return filename == "LICENSE"
}

var _ usecases.LocalFileAnalyserPort = (*licenseAnalyser)(nil)

func init() {
	// Registering a new item during package initialization.
	Register(acceptLicenseFile, newLicenseAnalyser, 1)
}
