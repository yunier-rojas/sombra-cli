---
title: CLI Commands
weight: 2
aliases:
  - /user-guide/commands/
---

## Overview

`sombra` provides commands to generate and update projects from templates, and to create templates from existing projects.

Run `sombra --help` at any time to view global help.

---

## `version`

Print the `sombra` version. The value is the release name or Git tag the binary
was built from. A binary built without version information prints `dev`.

```bash
sombra version
```

---

## `local` Commands

Used to apply and update templates in your working project.

### `sombra local init`

Register a template in the project's `sombra.yaml`. The command resolves the template
definition, prompts for the variables it declares, and creates or updates
`sombra.yaml` with the template URI and the captured values. It does not copy files;
run `sombra local update` to apply the template.

```bash
sombra local init [--id ID] TEMPLATE
```

#### Positional:

* `TEMPLATE`: Git repo URL of the template

#### Options:

* `--id`: Unique short name for the template. When omitted, `sombra` derives it
  from the repository name. The id only allows letters, numbers, underscores, and
  dashes. An id given explicitly must be unique; a derived id gets a numeric suffix
  on collision
* `--help, -h`: Show help

#### Example:

```bash
sombra local init github.com/your-org/your-template --id app
```

---

### `sombra local update`

Apply (or refresh) a template that is registered in `sombra.yaml`. Files are copied
using the variables stored for that template, and its `current` version is updated to
the resolved tag.

```bash
sombra local update [--tag TAG] [--method METHOD] [--prune] TEMPLATE
```

#### Positional:

* `TEMPLATE`: Id or Git repo URL of a template already registered by `sombra local init`.
  An id is matched first; the URI is used as a fallback so older entries without an id
  keep working.

#### Options:

* `--tag`: Specific git tag or version to use (defaults to the latest tag)
* `--method`: `copy` (default) or `diff` for smarter merging. `diff` ignores patterns marked [`copy_only`](../templates/concepts.md#copy_only-copy-method-only-patterns)
* `--prune`: Remove target files matching a `delete: true` pattern
* `--help, -h`: Show help

#### Example:

```bash
sombra local update --tag v1.2.0 --method diff --prune app
```

---

## `template` Commands

Used to turn existing codebases into reusable templates.

### `sombra template init`

Initialize a `.sombra/default.yaml` template from an existing project.

```bash
sombra template init [--exclude PATTERN] [--only PATTERN] [DIR]
```

#### Positional:

* `DIR`: Path to the project directory to convert (default: current dir)

#### Options:

* `--exclude, -e`: Glob to exclude files (e.g. `"*.pyc"`)
* `--only, -o`: Glob to include files (default: all files, `/**/*`)
* `--help, -h`: Show help

#### Example:

```bash
sombra template init --exclude "README.md" ./my-project
```

---

For detailed usage, see the [`sombra.yaml` file](sombra-file.md) or [Template Guide](../templates/index.md).
