package usecases

import (
	"fmt"
	"sort"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

type LocalInitCase interface {
	// LocalInit records a template in the target project. A nil parts slice
	// selects every part declared by the template.
	LocalInit(target, uri, id string, parts []string) error
}

type VariableReaderPort interface {
	GetValues(vars []entities.VarSpec) (entities.Mappings, error)
}

type LocalInitInteractor struct {
	repoPrepare RepositoryPrepareCase
	definition  DefinitionPort
	sombraDef   SombraDefManagerPort
	varsSource  VariableReaderPort
}

func (l *LocalInitInteractor) LocalInit(target, uri, id string, parts []string) error {
	repo, err := l.repoPrepare.Prepare(uri, "")
	if err != nil {
		return err
	}
	defer func() { _ = repo.Clean() }()

	def, err := l.definition.Load(l.definition.GetFile(repo.Dir()))
	if err != nil {
		return err
	}

	if parts == nil {
		parts = partNames(def)
	}
	parts = normalizeParts(parts)
	if _, err := activeEntries(def, parts); err != nil {
		return err
	}

	sombraFile := l.sombraDef.GetFile(target)
	targetDef, err := l.sombraDef.Load(sombraFile)
	if err != nil {
		return err
	}

	if id == "" {
		id = targetDef.UniqueID(entities.DeriveID(uri))
	} else if !entities.ValidID(id) {
		return entities.Invalid(fmt.Sprintf("invalid template id %q: use letters, numbers, underscores or dashes", id), nil)
	} else if targetDef.HasID(id) {
		return entities.Invalid(fmt.Sprintf("template id %q already exists", id), nil)
	}

	specs := make([]entities.VarSpec, 0, len(def.Vars))
	for name, spec := range def.Vars {
		spec.Name = name
		specs = append(specs, spec)
	}
	sort.Slice(specs, func(i, j int) bool { return specs[i].Name < specs[j].Name })

	values, err := l.varsSource.GetValues(specs)
	if err != nil {
		return err
	}

	targetDef.Templates = append(targetDef.Templates, &entities.TemplateConfig{
		ID:    id,
		URI:   uri,
		Parts: append([]string(nil), parts...),
		Vars:  values,
	})

	return l.sombraDef.Save(sombraFile, targetDef)
}

var _ LocalInitCase = (*LocalInitInteractor)(nil)

func NewLocalInitInteractor(
	repoPrepare RepositoryPrepareCase,
	definition DefinitionPort,
	sombraDef SombraDefManagerPort,
	varsSource VariableReaderPort,
) *LocalInitInteractor {
	return &LocalInitInteractor{
		repoPrepare: repoPrepare,
		definition:  definition,
		sombraDef:   sombraDef,
		varsSource:  varsSource,
	}
}
