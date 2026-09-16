---
title: CLI Commands
---

## Overview

The `sombra` CLI provides commands to generate and update projects from templates, and to create templates from existing projects.

Run `sombra --help` at any time to view global help.

---


## `local` Commands

Used to apply and update templates in your working project.

### `sombra local init`

Register a template in the project's `sombra.yaml`. The command resolves the template
definition, prompts for the variables it declares, and creates or updates
`sombra.yaml` with the template URI and the captured values. It does not copy files;
run `sombra local update` to apply the template.

```bash
sombra local init TEMPLATE
```

#### Positional:

* `TEMPLATE`: Git repo URL of the template

#### Example:

```bash
sombra local init github.com/your-org/your-template
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

* `TEMPLATE`: Git repo URL of the template. It must match the `uri` of a template
  already registered by `sombra local init`.

#### Options:

* `--tag`: Specific git tag or version to use (defaults to the latest tag)
* `--method`: `copy` (default) or `diff` for smarter merging. `diff` ignores patterns marked [`copy_only`](../sombra-templates/concepts.md#copy_only-copy-method-only-patterns)
* `--prune`: Remove target files matching a `delete: true` pattern
* `--help, -h`: Show help

#### Example:

```bash
sombra local update --tag v1.2.0 --method diff --prune github.com/your-org/your-template
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

For detailed usage, see the [Sombra File](sombra-file.md) or [Template Guide](../sombra-templates/index.md).
