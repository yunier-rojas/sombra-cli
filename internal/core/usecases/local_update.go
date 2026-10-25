package usecases

import (
	"fmt"
	"path"
	"strings"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

// LocalUpdateCase applies and updates a registered template.
type LocalUpdateCase interface {
	LocalUpdate(target, ref string, opts entities.LocalUpdateOptions) (entities.UpdateResult, error)
}

// LocalUpdateInteractor implements the single lock-based update path.
type LocalUpdateInteractor struct {
	repoPrepare    RepositoryPrepareCase
	definition     DefinitionPort
	sombraDef      SombraDefManagerPort
	versionManager VersionManagerPort
	lister         FileListerPort
	matcher        PathMatcherPort
	store          FileStorePort
	lock           LockStorePort
	hasher         HasherPort
	transformer    TransformerPort
}

type templatePlan struct {
	key    string
	locked entities.LockedTemplate
	plan   []entities.PlannedFile
}

func NewLocalUpdateInteractor(
	repoPrepare RepositoryPrepareCase,
	definition DefinitionPort,
	sombraDef SombraDefManagerPort,
	versionManager VersionManagerPort,
	lister FileListerPort,
	matcher PathMatcherPort,
	store FileStorePort,
	lock LockStorePort,
	hasher HasherPort,
	transformer TransformerPort,
) *LocalUpdateInteractor {
	return &LocalUpdateInteractor{
		repoPrepare:    repoPrepare,
		definition:     definition,
		sombraDef:      sombraDef,
		versionManager: versionManager,
		lister:         lister,
		matcher:        matcher,
		store:          store,
		lock:           lock,
		hasher:         hasher,
		transformer:    transformer,
	}
}

func (u *LocalUpdateInteractor) LocalUpdate(target, ref string, opts entities.LocalUpdateOptions) (entities.UpdateResult, error) {
	result := entities.UpdateResult{DryRun: opts.DryRun}

	sombraFile := u.sombraDef.GetFile(target)
	targetDef, err := u.sombraDef.Load(sombraFile)
	if err != nil {
		return result, err
	}

	matches := entities.ResolveTemplates(targetDef, ref)
	if len(matches) == 0 {
		return result, entities.NotFound(fmt.Sprintf("template %q", ref), nil)
	}
	uri := matches[0].URI
	for _, template := range matches {
		if template.URI != uri {
			return result, entities.Invalid(fmt.Sprintf("template %q resolves to multiple repositories", ref), nil)
		}
	}

	version, repo, err := u.prepare(uri, opts.Tag)
	if err != nil {
		return result, err
	}
	defer func() { _ = repo.Clean() }()

	lock, err := u.lock.Load(target)
	if err != nil {
		return result, err
	}

	plans := make([]templatePlan, 0, len(matches))
	configChanged := false
	for _, template := range matches {
		parts := normalizeParts(template.Parts)
		if opts.PartsSet {
			parts = normalizeParts(opts.Parts)
		}
		plan, err := u.planTemplate(target, repo.Dir(), version, template, parts, lock, opts, &result)
		if err != nil {
			return result, err
		}
		plans = append(plans, plan)
		if opts.PartsSet && !sameStrings(template.Parts, parts) {
			template.Parts = append([]string(nil), parts...)
			configChanged = true
		}
	}

	claimed := make(map[string]string)
	for _, plan := range plans {
		for _, planned := range plan.plan {
			if planned.Action == entities.ActionRemove {
				continue
			}
			destination := string(planned.Destination)
			if owner, ok := claimed[destination]; ok && owner != plan.key {
				return result, entities.Invalid(fmt.Sprintf("%s is produced by both template %q and template %q", destination, owner, plan.key), nil)
			}
			claimed[destination] = plan.key
		}
	}

	if opts.DryRun {
		for _, plan := range plans {
			recordActions(&result, plan.plan)
		}
		return result, nil
	}

	for _, plan := range plans {
		if err := u.apply(target, plan.plan, plan.locked.Files, &result); err != nil {
			return result, err
		}
		lock.Templates[plan.key] = plan.locked
	}

	if err := u.lock.Save(target, lock); err != nil {
		return result, err
	}
	if configChanged {
		if err := u.sombraDef.Save(sombraFile, targetDef); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (u *LocalUpdateInteractor) planTemplate(
	target, templateDir string,
	version entities.Version,
	template *entities.TemplateConfig,
	parts []string,
	lock *entities.Lock,
	opts entities.LocalUpdateOptions,
	result *entities.UpdateResult,
) (templatePlan, error) {
	base, err := normalizeBase(template.Path)
	if err != nil {
		return templatePlan{}, err
	}

	def, err := u.definition.Load(u.definition.GetFile(templateDir))
	if err != nil {
		return templatePlan{}, err
	}

	selected, err := resolveSelection(templateDir, def, parts, u.lister, u.matcher)
	if err != nil {
		return templatePlan{}, err
	}

	rules, err := u.renderRules(def.Rewrites, template.Vars)
	if err != nil {
		return templatePlan{}, err
	}

	key := templateKey(template)
	locked, exists := lock.Templates[key]
	if !exists {
		locked = entities.LockedTemplate{Files: make(map[string]entities.LockedFile)}
	}
	if locked.Files == nil {
		locked.Files = make(map[string]entities.LockedFile)
	}
	locked.Version = string(version)
	locked.Parts = append([]string(nil), parts...)

	destinations := make(map[entities.File]entities.File, len(selected))
	plannedKeys := make(map[string]bool, len(selected))
	plan := make([]entities.PlannedFile, 0, len(selected))

	for _, file := range selected {
		destination, err := resolveDestination(file.Source, rules, u.transformer, u.matcher)
		if err != nil {
			return templatePlan{}, err
		}
		if previous, ok := destinations[destination]; ok {
			return templatePlan{}, entities.Invalid(fmt.Sprintf("source %s and %s both resolve to %s", previous, file.Source, destination), nil)
		}
		destinations[destination] = file.Source

		destKey := path.Join(base, string(destination))
		plannedKeys[destKey] = true
		if owner := destinationOwner(lock, key, destKey); owner != "" {
			return templatePlan{}, entities.Invalid(fmt.Sprintf("%s is already owned by template %q", destKey, owner), nil)
		}

		content, err := u.renderFile(templateDir, file, rules, template.Vars)
		if err != nil {
			return templatePlan{}, err
		}

		existing, found, err := u.store.ReadIfExists(target, destKey)
		if err != nil {
			return templatePlan{}, err
		}

		planned := entities.PlannedFile{
			Source:      file.Source,
			Destination: entities.File(destKey),
			Mode:        file.Mode,
			Part:        file.Part,
			Content:     content,
			Hash:        u.hasher.Hash(content),
			SourceMode:  file.SourceMode,
		}

		lockedFile, lockedExists := locked.Files[destKey]
		action, reason, adopted := decideAction(existing, found, planned.Hash, file.Mode, lockedFile, lockedExists, opts, u.hasher)
		planned.Action = action
		planned.Reason = reason
		planned.Adopted = adopted || lockedFile.Adopted
		plan = append(plan, planned)

		if action == entities.ActionConflict {
			result.Conflicts = append(result.Conflicts, entities.Conflict{Path: destKey, Reason: reason})
		}
	}

	for destKey, lockedFile := range locked.Files {
		if plannedKeys[destKey] {
			continue
		}
		if lockedFile.Adopted {
			continue
		}

		existing, found, err := u.store.ReadIfExists(target, destKey)
		if err != nil {
			return templatePlan{}, err
		}
		if !found {
			delete(locked.Files, destKey)
			continue
		}
		if u.hasher.Hash(existing) != lockedFile.Hash {
			result.Conflicts = append(result.Conflicts, entities.Conflict{Path: destKey, Reason: "modified; not pruned"})
			continue
		}

		if !opts.Prune {
			result.Prunable = append(result.Prunable, entities.File(destKey))
			continue
		}

		plan = append(plan, entities.PlannedFile{
			Destination: entities.File(destKey),
			Action:      entities.ActionRemove,
			Mode:        lockedFile.Mode,
			Part:        lockedFile.Part,
			Hash:        lockedFile.Hash,
		})
	}

	return templatePlan{key: key, locked: locked, plan: plan}, nil
}

func (u *LocalUpdateInteractor) renderRules(rules []entities.Rewrite, vars entities.Mappings) ([]entities.Rewrite, error) {
	result := make([]entities.Rewrite, len(rules))
	for index, rule := range rules {
		result[index] = rule
		if rule.Replace == nil {
			return nil, entities.Invalid(fmt.Sprintf("rewrite %q has no replace value", rule.Find), nil)
		}
		rendered, err := u.definition.RenderValue(*rule.Replace, vars)
		if err != nil {
			return nil, entities.Invalid(fmt.Sprintf("rewrite %q", rule.Find), err)
		}
		result[index].Replace = &rendered
	}
	return result, nil
}

func (u *LocalUpdateInteractor) renderFile(templateDir string, file entities.SelectedFile, rules []entities.Rewrite, vars entities.Mappings) ([]byte, error) {
	content, found, err := u.definition.RenderOverlay(templateDir, file.Source, vars)
	if err != nil {
		return nil, err
	}
	if !found {
		content, err = u.store.Read(templateDir, string(file.Source))
		if err != nil {
			return nil, err
		}
	}

	if !file.Rewrite {
		rules = nil
	}
	contentRules, err := rulesFor(rules, entities.ScopeContent, file.Source, u.matcher)
	if err != nil {
		return nil, err
	}
	return u.transformer.Content(content, contentRules, file.Directives)
}

func (u *LocalUpdateInteractor) apply(target string, plan []entities.PlannedFile, files map[string]entities.LockedFile, result *entities.UpdateResult) error {
	for _, planned := range plan {
		switch planned.Action {
		case entities.ActionCreate, entities.ActionUpdate:
			if err := u.store.WriteAtomic(target, string(planned.Destination), planned.Content, planned.SourceMode); err != nil {
				return err
			}
			files[string(planned.Destination)] = entities.LockedFile{
				Hash:   planned.Hash,
				Mode:   planned.Mode,
				Part:   planned.Part,
				Source: string(planned.Source),
			}
		}
	}

	for _, planned := range plan {
		switch planned.Action {
		case entities.ActionAdopt:
			files[string(planned.Destination)] = entities.LockedFile{
				Hash:    planned.Hash,
				Mode:    planned.Mode,
				Part:    planned.Part,
				Source:  string(planned.Source),
				Adopted: true,
			}
		case entities.ActionUnchanged:
			files[string(planned.Destination)] = entities.LockedFile{
				Hash:    planned.Hash,
				Mode:    planned.Mode,
				Part:    planned.Part,
				Source:  string(planned.Source),
				Adopted: planned.Adopted,
			}
		case entities.ActionRemove:
			if err := u.store.Remove(target, string(planned.Destination)); err != nil {
				return err
			}
			delete(files, string(planned.Destination))
		}
	}
	recordActions(result, plan)
	return nil
}

func (u *LocalUpdateInteractor) prepare(uri, tag string) (entities.Version, RepositoryPort, error) {
	repo, err := u.repoPrepare.Prepare(uri, tag)
	if err != nil {
		return "", nil, err
	}
	if tag != "" {
		return entities.Version(tag), repo, nil
	}

	tags, err := repo.GetTags()
	if err != nil {
		return "", repo, err
	}
	version, err := u.versionManager.GetLatest(tags, "*")
	if err != nil {
		return "", repo, fmt.Errorf("resolve latest tag: %w", err)
	}
	if _, err := repo.Use(string(version)); err != nil {
		return "", repo, err
	}
	return version, repo, nil
}

func decideAction(
	existing []byte,
	found bool,
	renderedHash string,
	mode entities.Mode,
	locked entities.LockedFile,
	lockedExists bool,
	opts entities.LocalUpdateOptions,
	hasher HasherPort,
) (entities.Action, string, bool) {
	if !found {
		return entities.ActionCreate, "", false
	}
	if opts.Force {
		return entities.ActionUpdate, "", false
	}
	if !lockedExists && opts.Adopt && hasher.Hash(existing) == renderedHash {
		return entities.ActionAdopt, "", true
	}
	if mode == entities.ModeSeed {
		return entities.ActionKept, "seed file already exists", false
	}
	if !lockedExists {
		return entities.ActionConflict, "file exists and is not managed by sombra", false
	}
	if hasher.Hash(existing) != locked.Hash {
		return entities.ActionConflict, "file was modified since the last update", false
	}
	if hasher.Hash(existing) == renderedHash {
		return entities.ActionUnchanged, "", false
	}
	return entities.ActionUpdate, "", false
}

func destinationOwner(lock *entities.Lock, key, destination string) string {
	for name, template := range lock.Templates {
		if name == key {
			continue
		}
		if _, ok := template.Files[destination]; ok {
			return name
		}
	}
	return ""
}

func templateKey(template *entities.TemplateConfig) string {
	if template.ID != "" {
		return template.ID
	}
	return entities.DeriveID(template.URI)
}

func normalizeBase(value string) (string, error) {
	if hasParentSegment(value) {
		return "", entities.Invalid(fmt.Sprintf("template path %q must not contain a parent segment", value), nil)
	}
	cleaned := path.Clean("/" + strings.TrimPrefix(value, "/"))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return "", nil
	}
	return cleaned, nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func recordActions(result *entities.UpdateResult, plan []entities.PlannedFile) {
	for _, planned := range plan {
		switch planned.Action {
		case entities.ActionCreate:
			result.Created = append(result.Created, planned.Destination)
		case entities.ActionUpdate:
			result.Updated = append(result.Updated, planned.Destination)
		case entities.ActionUnchanged:
			result.Unchanged = append(result.Unchanged, planned.Destination)
		case entities.ActionKept:
			result.Kept = append(result.Kept, planned.Destination)
		case entities.ActionAdopt:
			result.Adopted = append(result.Adopted, planned.Destination)
		case entities.ActionRemove:
			result.Removed = append(result.Removed, planned.Destination)
		}
	}
}

var _ LocalUpdateCase = (*LocalUpdateInteractor)(nil)
