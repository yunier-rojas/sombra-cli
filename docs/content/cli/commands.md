---
title: CLI Commands
weight: 2
aliases:
  - /user-guide/commands/
---

## Overview

`sombra` generates and updates projects from templates and turns existing projects
into templates.

Run `sombra --help` to see the global help.

---

## `version`

Print the Sombra version. A binary built without version information prints `dev`.

```bash
sombra version
```

---

## `local` Commands

Used to apply and update templates in a project.

### `sombra local init`

Register a template in the project's `sombra.yaml`. The command loads the template
definition, validates the selected parts, prompts for variables and writes
`sombra.yaml`. It does not copy files.

```bash
sombra local init [--id ID] [--with PART]... TEMPLATE
```

#### Positional

- `TEMPLATE`: Git repository URL or local path.

#### Options

- `--id`: Unique short name. Derived from the URI when omitted.
- `--with`: Parts to select. Repeat the flag or use commas: `--with base,ci,docs`.
  When omitted, `base` is selected. When you name any part, `base` is only selected
  if you name it too.
- `--help, -h`: Show help.

#### Example

```bash
sombra local init https://github.com/acme/app-template.git --id app --with base,ci,docs
```

Prompts print the variable description and the default. Press Enter to accept the
default. A variable without a default requires a value.

---

### `sombra local update`

Apply or refresh a registered template.

```bash
sombra local update [--tag TAG] [--with PART]... [--dry-run] [--force] [--adopt] [--prune] TEMPLATE
```

#### Positional

- `TEMPLATE`: Template id or URI already registered with `sombra local init`.

#### Options

- `--tag`: Tag or commit to use. Defaults to the latest semantic tag.
- `--with`: Replace the stored parts. Repeat or use commas. `base` is only selected
  when named. An empty value resets the selection to `base`.
- `--dry-run`: Print the plan. No files are written and the lock is not changed.
- `--force`: Overwrite files even when they were modified or are not managed yet.
- `--adopt`: Record existing files whose content already matches the template. No
  files are overwritten. Adopted files are never pruned.
- `--prune`: Remove template files that are no longer produced and are unmodified.
- `--help, -h`: Show help.

#### Output

Every destination is reported as one of:

`created`, `updated`, `unchanged`, `kept`, `adopted`, `removed`, `prunable`,
`conflict`.

`prunable` files are only reported. Run again with `--prune` to remove them.
`conflict` files are never overwritten. The command exits with a non-zero status when
conflicts remain, including in a dry run.

#### Examples

```bash
sombra local update app
sombra local update app --tag v1.2.0
sombra local update app --with ci --dry-run
sombra local update app --prune
sombra local update app --adopt
```

#### Conflict handling

A conflict means one of:

- the file exists and is not recorded in `.sombra/lock.yaml`;
- the file was edited since the last update;
- a file that is no longer produced was edited, so `--prune` keeps it.

Resolve a conflict by restoring the file, by adopting it when the content matches, or
by forcing an overwrite when that is the intent.

---

## `template` Commands

Used to turn an existing project into a template.

### `sombra template init`

Generate `.sombra/template.yaml`.

```bash
sombra template init [--only GLOB]... [--exclude GLOB]... [--force] [--dry-run] [--include-untracked] [DIR]
```

#### Positional

- `DIR`: Project directory. Defaults to the current directory.

#### Options

- `--only, -o`: Glob selecting files. Repeatable. The generated definition contains
  concrete paths, not globs.
- `--exclude, -e`: Glob excluding files. Repeatable. Written to `exclude`.
- `--force`: Overwrite an existing definition.
- `--dry-run`: Print the definition without writing it.
- `--include-untracked`: Include untracked files. Only applies to git repositories.
- `--help, -h`: Show help.

#### Behavior

- In a git repository with commits, Sombra lists tracked files with `git ls-files`.
  Ignored and untracked files are excluded by default.
- Without git, Sombra walks the directory and skips common build and cache
  directories.
- Files are grouped into top level entries such as `/cmd/` and `/Makefile`.
- `.sombra/` and the root `sombra.yaml` are never included.
- Variables and rewrites are seeded from `go.mod`, the git remote and the git user.
  A missing value produces a warning and no rewrite for that value.
- The generated definition is a starting point. Review `parts.base`, `exclude` and
  `rewrites` before tagging a release.

#### Examples

```bash
sombra template init .
sombra template init --only '/cmd/**' --only '/internal/**' .
sombra template init --exclude '*.log' --exclude '/secrets/**' .
sombra template init --dry-run .
```
