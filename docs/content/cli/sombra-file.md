---
title: sombra.yaml File
weight: 1
aliases:
  - /user-guide/sombra-file/
---

## What is the `sombra.yaml` file?

The `sombra.yaml` file defines how templates are applied to a project. It lives in the root of a **target repository** — the repo you are creating or updating using a template.

This file tells `sombra` which template(s) to use, which variables to pass, and which version (tag) is currently applied.

---

## File Location

Place `sombra.yaml` at the root of your generated project:

```

my-app/
├── sombra.yaml
├── go.mod
├── README.md
└── ...

```

You normally do not write it by hand: `sombra local init` creates it (or appends to it) and prompts for the variables the template declares.

---

## Configuration Reference

### `templates`

List of one or more templates applied to the project. Each entry names a template repository and the variables bound to it.

```yaml
templates:
  - id: app
    uri: https://github.com/your-org/your-template.git
    vars:
      project: My Awesome Project
      author: Jane Doe
      email: jane@example.com
      entity: Settings
```

#### Fields:

* `id`: Optional short name for the template. Unique within the file. Used to target one template in `sombra local update`
* `uri`: The Git repository URL (or local path) of the template
* `path`: Optional subdirectory of the target project where the template is applied
* `current`: The version (Git tag) currently applied; written by `sombra`
* `vars`: Key-value pairs that are injected into the template

`id` only allows letters, numbers, underscores, and dashes. When you do not
provide one, `sombra local init` derives it from the repository name. If that name
is already taken, a numeric suffix is appended (`your-template-2`, `your-template-3`,
and so on).

`sombra local update` accepts either an `id` or a `uri`. It matches `id` first and
falls back to `uri`, so projects created before `id` existed keep working.

`current` is updated automatically when `sombra local update` succeeds. It lets a project record which template version it was generated from.

---

## Full Example

```yaml
templates:
  - id: app
    uri: https://github.com/your-org/your-template.git
    current: v1.2.0
    vars:
      project: Internal API
      author: Dev Team
      email: dev@example.com
      entity: Config
  - id: common-ci
    uri: https://github.com/your-org/common-ci.git
    path: .ci
    vars:
      project: Internal API
```

---

## Creating and Applying the File

Register a template (creates `sombra.yaml` if missing and prompts for its variables):

```bash
sombra local init https://github.com/your-org/your-template.git --id app
```

Apply or refresh the registered template by id or URI:

```bash
sombra local update app
```

For more, check out the [CLI Commands](commands.md) or [Installation Guide](../getting-started/installation.md).
