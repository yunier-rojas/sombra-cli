package entities

// AlwaysIgnoreDirs are never walked when copying a template.
var AlwaysIgnoreDirs = []string{
	".git",
	".sombra",
	".idea",
	".vscode",
	"node_modules",
	"__pycache__",
}

// InitIgnoreDirs are skipped by template init when the source is not a git
// repository. A tracked git repository uses git ls-files instead.
var InitIgnoreDirs = []string{
	".git",
	".sombra",
	".idea",
	".vscode",
	"node_modules",
	"__pycache__",
	"vendor",
	"build",
	"dist",
}
