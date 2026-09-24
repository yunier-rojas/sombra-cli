package usecases

import (
	"fmt"
)

type LocalUpdateCase interface {
	LocalUpdate(target, ref, tag string, prune bool) ([]string, error)
}

type CliUpdateCase interface {
	DoLocalUpdate(target, ref, tag, method string, prune bool) ([]string, error)
}

type CliUpdateInteractor struct {
	copyCase LocalUpdateCase
	diffCase LocalUpdateCase
}

func (l *CliUpdateInteractor) DoLocalUpdate(target, ref, tag, method string, prune bool) ([]string, error) {
	var useCase LocalUpdateCase
	switch method {
	case "diff":
		useCase = l.diffCase
	case "copy":
		useCase = l.copyCase
	default:
		return nil, fmt.Errorf("method %s not supported", method)
	}
	return useCase.LocalUpdate(target, ref, tag, prune)
}

func NewCliUpdateInteractor(copyCase LocalUpdateCase, diffCase LocalUpdateCase) *CliUpdateInteractor {
	return &CliUpdateInteractor{copyCase: copyCase, diffCase: diffCase}
}

var _ CliUpdateCase = (*CliUpdateInteractor)(nil)
