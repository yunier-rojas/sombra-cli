package usecases

import (
	"bytes"
	"path/filepath"
	"sort"

	"github.com/yunier-rojas/sombra-cli/internal/core/entities"
)

// pruneTarget reports (and, when prune is enabled, removes) files in targetDir
// that match a tombstone pattern (`delete: true`). The returned paths are
// relative to targetDir. Without the prune flag candidates are only reported.
func pruneTarget(scanner DirectoryManagerPort, files FileManagerPort, engine SombraEngineCase, targetDir string, patterns []*entities.Pattern, prune bool) ([]string, error) {
	if !hasDeletePatterns(patterns) {
		return nil, nil
	}

	var matchedFiles []entities.File
	var matchedDirs []entities.File

	tree := scanner.ScanTree(targetDir, []entities.Wildcard{"**/*"}, nil)
	for result := range tree {
		if result.Err != nil {
			return nil, result.Err
		}

		match, err := engine.MatchDelete(result.File, patterns)
		if err != nil {
			return nil, err
		}
		if !match {
			continue
		}

		if result.IsDir {
			matchedDirs = append(matchedDirs, result.File)
		} else {
			matchedFiles = append(matchedFiles, result.File)
		}
	}

	reported := make([]string, 0, len(matchedFiles)+len(matchedDirs))
	for _, file := range matchedFiles {
		reported = append(reported, string(file))
	}
	for _, dir := range matchedDirs {
		reported = append(reported, string(dir))
	}
	sort.Strings(reported)

	if !prune {
		return reported, nil
	}

	for _, file := range matchedFiles {
		if err := files.Remove(targetDir, file); err != nil {
			return reported, err
		}
	}
	for _, dir := range pruneDirectories(matchedFiles, matchedDirs) {
		// non empty directories are intentionally left in place
		_ = files.Remove(targetDir, dir)
	}

	return reported, nil
}

func hasDeletePatterns(patterns []*entities.Pattern) bool {
	for _, pattern := range patterns {
		if pattern.Delete {
			return true
		}
	}
	return false
}

// pruneDirectories returns the directories that may become empty after the
// matched files are removed, deepest first, so children are removed before
// their parents. The target root is never included.
func pruneDirectories(removed []entities.File, matchedDirs []entities.File) []entities.File {
	pending := make(map[entities.File]struct{})

	for _, dir := range matchedDirs {
		pending[dir] = struct{}{}
	}
	for _, file := range removed {
		for dir := entities.File(filepath.Dir(string(file))); dir != "/" && dir != "." && dir != ""; dir = entities.File(filepath.Dir(string(dir))) {
			pending[dir] = struct{}{}
		}
	}

	dirs := make([]entities.File, 0, len(pending))
	for dir := range pending {
		dirs = append(dirs, dir)
	}
	sort.Slice(dirs, func(i, j int) bool {
		return depth(dirs[i]) > depth(dirs[j])
	})
	return dirs
}

func depth(file entities.File) int {
	return bytes.Count([]byte(file), []byte("/"))
}
