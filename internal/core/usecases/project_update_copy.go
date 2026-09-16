package usecases

import (
	"fmt"
	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
	"path/filepath"
)

type LocalCopyInteractor struct {
	repoPrepare        RepositoryPrepareCase
	templateDefManager TemplateDefManagerPort
	sombraDefManager   SombraDefManagerPort
	versionManager     VersionManagerPort
	scanner            DirectoryManagerPort
	localFiles         FileManagerPort
	engine             SombraEngineCase
}

func NewLocalCopyInteractor(
	repoPrepare RepositoryPrepareCase,
	templateDefManager TemplateDefManagerPort,
	sombraDefManager SombraDefManagerPort,
	versionManager VersionManagerPort,
	scanner DirectoryManagerPort,
	localFiles FileManagerPort,
	engine SombraEngineCase,
) *LocalCopyInteractor {
	return &LocalCopyInteractor{
		repoPrepare:        repoPrepare,
		templateDefManager: templateDefManager,
		sombraDefManager:   sombraDefManager,
		versionManager:     versionManager,
		scanner:            scanner,
		localFiles:         localFiles,
		engine:             engine,
	}
}

func (copy *LocalCopyInteractor) LocalUpdate(target, uri, tag string, prune bool) ([]string, error) {
	// Read sombra file
	sombraFile := copy.sombraDefManager.GetFile(target)
	def, err := copy.sombraDefManager.Load(sombraFile)
	if err != nil {
		return nil, err
	}

	// Download and prepare the version
	repo, err := copy.repoPrepare.Prepare(uri, "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = repo.Clean() }()

	// If the tag variable is empty, find the latest tag
	var version entities.Version
	var tags []string
	if tag == "" {
		tags, err = repo.GetTags()
		if err != nil {
			return nil, err
		}
		version, err = copy.versionManager.GetLatest(tags, "*")
		if err != nil {
			return nil, err
		}
	} else {
		version = entities.Version(tag)
	}

	// Iterate over all templates
	var tpl *entities.TemplateDef
	var fn entities.File
	var removed []string
	for _, template := range def.Templates {
		if template.URI != uri {
			continue
		}

		targetDir := filepath.Join(target, template.Path)

		// Render TemplateConfig Definition using Sombra configuration
		fn = copy.templateDefManager.GetFile(repo.Dir())
		tpl, err = copy.templateDefManager.Render(fn, template.Vars)
		if err != nil {
			return removed, err
		}

		// Execute the mappings
		err = copy.copyFiles(repo.Dir(), targetDir, tpl, template.Vars)
		if err != nil {
			return removed, err
		}

		// Remove files dropped by the template (tombstones)
		pruned, err := pruneTarget(copy.scanner, copy.localFiles, copy.engine, targetDir, tpl.Patterns, prune)
		if err != nil {
			return removed, err
		}
		removed = append(removed, pruned...)

		// Update the template configuration
		template.Current = version
	}

	// Store sombra file
	err = copy.sombraDefManager.Save(sombraFile, def)
	if err != nil {
		return removed, err
	}

	return removed, nil
}

func (copy *LocalCopyInteractor) copyFiles(templateDir, targetDir string, templateConfig *entities.TemplateDef, vars entities.Mappings) error {
	tree := copy.scanner.ScanTree(templateDir, []entities.Wildcard{"**/*"}, nil)
	var fn entities.File
	var items *entities.MapResult
	for result := range tree {
		if result.Err != nil {
			return result.Err
		}

		fn = result.File

		match, res, err := copy.engine.Match(fn, templateConfig.Patterns)
		if err != nil {
			return err
		}

		// Notice that match can be false even if res is not empty
		// This is because the mapping engine only matches a file
		// if there are non-abstract mappings matching the file
		if !match {
			continue
		}

		items = copy.engine.Combine(res)

		if result.IsDir {
			err = copy.processDir(targetDir, fn, items)
		} else {
			err = copy.processFile(templateDir, targetDir, fn, items, vars)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

func (copy *LocalCopyInteractor) processDir(target string, path entities.File, res *entities.MapResult) error {
	newDir := copy.engine.NewFile(path, res.Path, res.Name)
	return copy.localFiles.EnsureDir(target, newDir)
}

func (copy *LocalCopyInteractor) processFile(src string, target string, file entities.File, res *entities.MapResult, vars entities.Mappings) error {
	newFile := copy.engine.NewFile(file, res.Path, res.Name)

	var content []byte
	var err error
	if res.Replace != nil {
		// `replace` points to a file inside the template's .sombra directory;
		// its rendered content is used instead of the matched file.
		content, err = copy.templateDefManager.RenderReplace(src, *res.Replace, vars)
		if err != nil {
			return err
		}
	} else {
		content, err = copy.localFiles.Read(src, file)
		if err != nil {
			return err
		}
	}

	newContent, err := copy.engine.TransformFile(content, res.Content, vars, res.BlockDirectives)
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	err = copy.localFiles.Write(target, newFile, newContent)
	return err
}

var _ LocalUpdateCase = (*LocalCopyInteractor)(nil)
