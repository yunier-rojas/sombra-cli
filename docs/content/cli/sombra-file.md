---
title: sombra.yaml File
weight: 1
aliases:
  - /user-guide/sombra-file/
---

## What is the `sombra.yaml` file?

`sombra.yaml` declares which templates are applied to a target project. It lives in the
root of the project.

The file stores the template URI, the selected parts and the variable values. The
applied version and the file hashes live in `.sombra/lock.yaml`, not here.

`sombra local init` creates or updates this file. `sombra local update` reads it.

---

## File Location

```text
my-app/
├── sombra.yaml
├── .sombra/
│   └── lock.yaml
├── go.mod
└── ...
```

Commit both files.

---

## Configuration Reference

### `templates`

List of templates applied to the project.

```yaml
templates:
  - id: app
    uri: https://github.com/acme/app-template.git
    parts: [base, ci, docs]
    vars:
      project: hello-cli
      module: example.com/hello-cli
      author: Acme Inc.
```

| Field | Required | Meaning |
| --- | --- | --- |
| `id` | no | Short name used by `sombra local update`. Unique in the file. |
| `uri` | yes | Git repository URL or local path. |
| `path` | no | Destination subdirectory for the whole template. |
| `parts` | no | Parts to apply. Defaults to `[base]` when omitted or empty. |
| `vars` | yes | Values bound to the template. |

`id` allows letters, numbers, underscores and dashes. When omitted,
`sombra local init` derives it from the repository name. A collision gets a numeric
suffix.

`sombra local update` accepts an `id` or a `uri`. It matches `id` first and falls back
to `uri`.

Unknown fields are errors.

---

## Parts

Every template declares a required `base` part and may declare more. Selecting parts
records them in `sombra.yaml`:

```bash
sombra local init https://github.com/acme/app-template.git --only base,ci,docs
```

`sombra local init` selects every part when `--only` is omitted. `--only` replaces the
selection. `base` is only selected when named. For example, `--only ci` copies `ci`
files only.

Later updates use the stored selection:

```bash
sombra local update app
```

When the stored selection is empty, updates use `base`.

Replace the selection with `--only`:

```bash
sombra local update app --only ci
sombra local update app --only base,ci
sombra local update app --only ''
```

An empty value resets the selection to `base`.

Dropping a part does not delete files immediately. Run `--prune` to remove files that
are unmodified. Modified files are reported and kept.

---

## Multiple Templates

```yaml
templates:
  - id: app
    uri: https://github.com/acme/app-template.git
    parts: [base]
    vars:
      project: Internal API
  - id: common-ci
    uri: https://github.com/acme/common-ci.git
    parts: [ci]
    vars:
      owner: platform
```

Each template contributes only its selected parts. In this example, `common-ci` copies
its `ci` part and not its `base` part. That keeps shared base files from colliding.

The same destination cannot be owned by two templates. The update fails if it detects
a collision.

---

## The Applied Version

The version is recorded in `.sombra/lock.yaml`:

```yaml
lock: 1
templates:
  app:
    version: v1.2.0
    parts: [base, ci]
    files:
      /Makefile:
        hash: sha256:...
        mode: sync
        source: /Makefile
```

`sombra.yaml` does not contain a `current` field. Use `--tag` to pin a version in a
specific update.

---

## Creating and Applying the File

Register a template:

```bash
sombra local init https://github.com/acme/app-template.git --id app --only base,ci
```

Apply or refresh it:

```bash
sombra local update app
```

For every option, see [CLI Commands](commands.md).
