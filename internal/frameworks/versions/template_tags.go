package versions

import (
	"github.com/Masterminds/semver/v3"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"github.com/yunier-rojas/sombra-cli/internal/core/usecases"
	"github.com/yunier-rojas/sombra-cli/internal/frameworks/logger"
	"sort"
)

type SemVerList []*semver.Version

func (s SemVerList) Len() int {
	return len(s)
}

func (s SemVerList) Less(i, j int) bool {
	return s[i].LessThan(s[j])
}

func (s SemVerList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

var _ sort.Interface = (SemVerList)(nil)

type TemplateTagManagerService struct {
}

func (t *TemplateTagManagerService) GetLatest(tags []string, constraint string) (entities.Version, error) {
	var err error
	var v *semver.Version

	if constraint == "" {
		logger.Info("No constraint provided. Defaulting to '*'.")
		constraint = "*"
	}

	c, err := semver.NewConstraint(constraint)
	if err != nil {
		logger.Error("Failed to create constraint", err)
		return "", entities.Invalid("parse version constraint "+constraint, err)
	}

	vs := make(SemVerList, 0)
	for _, value := range tags {
		v, err = semver.NewVersion(value)
		if err != nil {
			logger.Info("Skipping invalid version: " + value)
			continue
		}

		vs = append(vs, v)
	}

	if len(vs) == 0 {
		logger.Info("No valid versions available after parsing.")
	}

	sort.Sort(vs)

	var last *semver.Version
	for _, res := range vs {
		if c.Check(res) {
			last = res
		}
	}

	if last == nil {
		logger.Error("No matching versions found", err)
		return "", entities.NotFound("matching version for "+constraint, nil)
	}

	logger.Info("Latest version found: " + last.Original())
	return entities.Version(last.Original()), nil
}

func NewTemplateTagManagerService() *TemplateTagManagerService {
	return &TemplateTagManagerService{}
}

var _ usecases.VersionManagerPort = (*TemplateTagManagerService)(nil)
