package usecases

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

type DirectoryLocalDiffInteractor struct {
	repoPrepare        RepositoryPrepareCase
	patchManager       PatchPort
	templateDefManager TemplateDefManagerPort
	sombraDefManager   SombraDefManagerPort
	versionManager     VersionManagerPort
	scanner            DirectoryManagerPort
	localFiles         FileManagerPort
	engine             SombraEngineCase
}

func NewDirectoryLocalDiffInteractor(
	repoPrepare RepositoryPrepareCase,
	patchManager PatchPort,
	templateDefManager TemplateDefManagerPort,
	sombraDefManager SombraDefManagerPort,
	versionManager VersionManagerPort,
	scanner DirectoryManagerPort,
	localFiles FileManagerPort,
	engine SombraEngineCase,
) *DirectoryLocalDiffInteractor {
	return &DirectoryLocalDiffInteractor{
		repoPrepare:        repoPrepare,
		patchManager:       patchManager,
		templateDefManager: templateDefManager,
		sombraDefManager:   sombraDefManager,
		versionManager:     versionManager,
		scanner:            scanner,
		localFiles:         localFiles,
		engine:             engine,
	}
}

func (diff *DirectoryLocalDiffInteractor) LocalUpdate(target, uri, tag string, prune bool) ([]string, error) {
	// Read sombra file
	sombraFile := diff.sombraDefManager.GetFile(target)
	def, err := diff.sombraDefManager.Load(sombraFile)
	if err != nil {
		return nil, err
	}

	// Download and prepare the version
	repo, err := diff.repoPrepare.Prepare(uri, "")
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
		version, err = diff.versionManager.GetLatest(tags, "*")
		if err != nil {
			return nil, err
		}
	} else {
		version = entities.Version(tag)
	}

	// Iterate over all templates
	var fromVersion entities.Version
	var sig int8
	var tpl *entities.TemplateDef
	var fn entities.File
	var removed []string
	for _, template := range def.Templates {
		if template.URI != uri {
			continue
		}
		if template.Current != "" {
			sig, err = diff.versionManager.Compare(template.Current, version)
			if err != nil {
				return removed, err
			}
			if sig >= 0 {
				continue
			}
			fromVersion = template.Current
		} else {
			// yes, see https://stackoverflow.com/questions/40883798/how-to-get-git-diff-of-the-first-commit
			fromVersion = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
		}

		targetDir := filepath.Join(target, template.Path)

		// Render TemplateConfig Definition using Sombra configuration
		fn = diff.templateDefManager.GetFile(repo.Dir())
		tpl, err = diff.templateDefManager.Render(fn, template.Vars)
		if err != nil {
			return removed, err
		}

		// `copy_only` patterns are owned by the copy update method; the diff
		// method must not see them.
		patterns := withoutCopyOnly(tpl.Patterns)

		// prepare the diff
		err = diff.applyDiff(repo, targetDir, patterns, fromVersion, version)
		if err != nil {
			return removed, err
		}

		// Remove files dropped by the template (tombstones)
		pruned, err := pruneTarget(diff.scanner, diff.localFiles, diff.engine, targetDir, patterns, prune)
		if err != nil {
			return removed, err
		}
		removed = append(removed, pruned...)

		// Update the template configuration
		template.Current = version
	}

	// Store sombra file
	err = diff.sombraDefManager.Save(sombraFile, def)
	if err != nil {
		return removed, err
	}

	return removed, nil
}

// withoutCopyOnly drops patterns marked `copy_only`, which are only applied by
// the copy update method. Tombstones and abstract bases are kept as-is so the
// remaining patterns behave exactly as they do on the copy path.
func withoutCopyOnly(patterns []*entities.Pattern) []*entities.Pattern {
	filtered := make([]*entities.Pattern, 0, len(patterns))
	for _, pattern := range patterns {
		if pattern.CopyOnly {
			continue
		}
		filtered = append(filtered, pattern)
	}
	return filtered
}

func (diff *DirectoryLocalDiffInteractor) applyDiff(repo RepositoryPort, targetDir string, patterns []*entities.Pattern, fromVersion, toVersion entities.Version) error {
	_, err := repo.Use(string(toVersion))
	if err != nil {
		return err
	}

	patch, err := repo.Diff(string(fromVersion))
	if err != nil {
		return err
	}

	newDiff, err := diff.transformPatch(patch, patterns)
	if err != nil {
		return err
	}

	return diff.patchManager.Apply(targetDir, newDiff)
}

func (diff *DirectoryLocalDiffInteractor) transformPatch(content []byte, patterns []*entities.Pattern) ([]byte, error) {
	// TODO: Move this code to a port/service
	// https://git-scm.com/docs/diff-format
	lines := bytes.Split(content, []byte("\n"))

	var isMatch bool
	var err error
	var all []*entities.Pattern
	var res *entities.MapResult
	data := make([][]byte, 0)

	// NOTE: notice how the regex defines the file name from the '/'
	// because filenames for sombra work in the same way that gitignore works
	startLine := regexp.MustCompile(`^diff\s+(--[a-z]+)?\s+a(/.*)\s+b(/.*)$`)

	for _, line := range lines {
		strLine := string(line)
		isDiffStart := startLine.MatchString(strLine)

		// `diff` lines state which file is being compared
		// mappings need to be collected for this file
		// if not mappings apply to the changes, it shouldn't be added
		if isDiffStart {
			groups := startLine.FindStringSubmatch(strLine)
			target := groups[2]
			isMatch, all, err = diff.engine.Match(entities.File(target), patterns)
			if err != nil {
				return nil, err
			}

			if all != nil {
				res = diff.engine.Combine(all)
			}
		}

		// diff heading need to be change based on collected mappings
		if (res != nil) && isMatch && isDiffStart {
			groups := startLine.FindStringSubmatch(strLine)
			aFile := groups[2]
			newA := diff.engine.NewFile(entities.File(aFile), res.Path, res.Name)
			bFile := groups[3]
			newB := diff.engine.NewFile(entities.File(bFile), res.Path, res.Name)
			line = []byte(fmt.Sprintf("diff %s a%s b%s", groups[1], string(newA), string(newB)))
		} else

		// both file names need to be change based on collected mappings
		if (res != nil) && isMatch && len(line) > 5 && (bytes.Equal([]byte("---"), line[:3]) || bytes.Equal([]byte("+++"), line[:3])) {
			target := string(line[5:])
			newFile := diff.engine.NewFile(entities.File(target), res.Path, res.Name)
			newPath := []byte(string(newFile))
			line = append(line[:5], newPath...)
		} else

		// change context needs to be updated following the mappings of the file
		if (res != nil) && isMatch && len(line) > 2 && bytes.Equal([]byte("@@"), line[:2]) {
			n := bytes.Index(line[2:], []byte("@@")) + 2
			newContent := diff.engine.NewContent(line[n:], res.Content)
			line = append(line[:n], newContent...)
		} else

		// file content needs to be updated following the mappings of the file
		if (res != nil) && isMatch && len(line) > 1 && (bytes.Equal([]byte("-"), line[:1]) || bytes.Equal([]byte("+"), line[:1])) {
			line = diff.engine.NewContent(line, res.Content)
		}

		if isMatch {
			data = append(data, line)
		}
	}

	return bytes.Join(data, []byte("\n")), nil

}

var _ LocalUpdateCase = (*DirectoryLocalDiffInteractor)(nil)
