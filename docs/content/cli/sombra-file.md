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
  - uri: github.com/your-org/your-template
    vars:
      project: My Awesome Project
      author: Jane Doe
      email: jane@example.com
      entity: Settings
```

#### Fields:

* `uri`: The Git repository URL (or local path) of the template
* `path`: Optional subdirectory of the target project where the template is applied
* `current`: The version (Git tag) currently applied; written by `sombra`
* `vars`: Key-value pairs that are injected into the template

`current` is updated automatically when `sombra local update` succeeds. It lets a project record which template version it was generated from.

---

## Full Example

```yaml
templates:
  - uri: github.com/your-org/your-template
    current: v1.2.0
    vars:
      project: Internal API
      author: Dev Team
      email: dev@example.com
      entity: Config
  - uri: github.com/your-org/common-ci
    path: .ci
    vars:
      project: Internal API
```

---

## Creating and Applying the File

Register a template (creates `sombra.yaml` if missing and prompts for its variables):

```bash
sombra local init github.com/your-org/your-template
```

Apply or refresh the registered template:

```bash
sombra local update github.com/your-org/your-template
```

For more, check out the [CLI Commands](commands.md) or [Installation Guide](../getting-started/installation.md).
