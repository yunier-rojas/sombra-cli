---
title: Template Concepts
weight: 1
aliases:
  - /sombra-templates/concepts/
---

## Overview

A template is a repository with a definition file at `.sombra/template.yaml`. The
definition answers four separate questions:

1. Which files are copied: `parts` and `exclude`.
2. Who owns each file after it is copied: `mode`.
3. How paths, names and content are rewritten: `rewrites`.
4. What Sombra wrote last time: `.sombra/lock.yaml` in the target project.

The four aspects are independent. A file is never copied just because a rewrite matches
it. A rewrite never selects a file.

---

## File Location

```text
.
└── .sombra/
    ├── template.yaml
    └── overlay/
```

`.sombra/` is template metadata. It is never copied to a target project. The
definition file must declare `schema: 2`.

Unknown fields are errors. A typo fails the update instead of silently dropping
files.

---

## Variables

`vars` declares the values the consumer provides. Values are stored in the target's
`sombra.yaml` and reused on every update.

```yaml
vars:
  project:
    description: Project and command name
    default: app
  author: {}
```

A variable can be a scalar (the default) or an object with `description` and
`default`. `sombra local init` prints the description and offers the default.

Rewrite replacements use Go templates and Sprig functions:

```yaml
rewrites:
  - find: "sombra-cli"
    replace: "{{ .project | kebabcase }}"
```

A missing variable is an error. To insert a literal `{{`, escape it:
`{{ "{{" }}`.

---

## Selection

Every definition declares a `parts.base` part. It lists the files and directory
prefixes copied by default:

```yaml
parts:
  base:
    include:
      - /Makefile
      - /mk/build.mk
      - /cmd/app/
```

Rules:

- A path must start with `/` and is relative to the template root.
- A trailing `/` selects every file below the directory.
- `/` selects the whole repository.
- Globs are not allowed in `include`. Use explicit files and directories.
- `exclude` accepts globs and always wins:

```yaml
exclude:
  - "**/*_test.go"
  - "*.md"
```

A pattern without a slash matches at any depth. A pattern with a slash is anchored at
the root. A trailing `/` selects a subtree.

The following directories are always skipped: `.git`, `.sombra`, `.idea`, `.vscode`,
`node_modules` and `__pycache__`.

If an `include` path does not exist, the update fails. Missing files are never
silently ignored.

---

## Ownership Modes

Each entry has a `mode`. The default is `sync`.

### `sync`

Sombra owns the file. On every update:

- Missing file: create it.
- File matches the hash in the lock: update it.
- File differs from the hash: report a conflict and leave it untouched.

A file that exists but is not in the lock is also a conflict. Pass `--force` to
overwrite or `--adopt` to record it when its content already matches.

### `seed`

Sombra writes the file only when it does not exist. Later updates keep the consumer's
version. Use it for files the consumer is expected to edit, such as the first command,
the composition root or `go.mod`.

A seed file that already existed before Sombra wrote it is not recorded in the lock and
is never pruned.

```yaml
parts:
  base:
    include:
      - path: /cmd/app/main.go
        mode: seed
```

Modes, directives and rewriting inherit by the most specific matching path. An exact
file entry beats a directory entry. A longer directory beats a shorter one.

---

## Rewrites

`rewrites` is one ordered list. Each rule runs once per file, in declaration order.

```yaml
rewrites:
  - find: "github.com/acme/app"
    replace: "{{ .module }}"
    in: [path, name, content]

  - find: "app"
    as: word
    replace: "{{ .project }}"
    in: [path, name, content]

  - files: [/internal/core/entities/app.go]
    find: "app.go"
    replace: "entities.go"
    in: [name]
```

### Fields

| Field | Required | Default | Meaning |
| --- | --- | --- | --- |
| `find` | yes | | Text, word, regexp or structured path. |
| `as` | no | `literal` | `literal`, `word`, `regex`, `json`, `yaml` or `ini`. |
| `ignore_case` | no | `false` | Case-insensitive literal or word match. |
| `replace` | yes | | Rendered with the template variables. Use `""` to delete. |
| `in` | no | `[content]` | `path`, `name`, `content` or `all`. |
| `files` | no | all files | Globs matched against the source path. |

Rules apply in order. A later rule sees the output of an earlier one.

### Matching modes

- `literal` replaces every occurrence of `find`.
- `word` replaces only whole tokens. Token characters are ASCII letters, digits, `_`,
  `.` and `-`. This leaves `app-cli`, `app.yaml` and `app_name` intact when `find` is
  `app`.
- `regex` uses RE2. `replace` supports `$1` and `${name}`.
- `json`, `yaml` and `ini` update one value addressed by a dot separated path. They
  only apply to `content`.

```yaml
rewrites:
  - find: "scripts.test"
    as: json
    replace: "go test ./..."
```

### Scopes

- `path` rewrites the directory part of the source path.
- `name` rewrites the base name.
- `content` rewrites file bytes.
- `all` is shorthand for all three.

`files` is evaluated against the source path, before any rename. Use it to scope a
rule to one file or directory.

---

## Overlays

An overlay replaces the content of a source file with a file that lives in the
template metadata:

```text
.sombra/overlay/README.md
```

When a selected source file is `/README.md`, Sombra uses the overlay instead of the
repository's `README.md`. The overlay is rendered with the template variables, then
rewrites apply.

Overlays are the way to ship reduced or template-only content while keeping a full
file in the template repository. Overlay files are never copied directly.

---

## In-File Directives

Set `directives: true` on an entry to interpret marker lines in the copied file:

```yaml
- path: /cmd/app/main.go
  directives: true
```

```go
// sombra:skip
templateOnly := true
// sombra:end
```

Rules:

- Markers are comment agnostic. Any line containing the marker is a marker line.
- Blocks nest.
- An unbalanced or mismatched block fails the update with a line number.
- Directives are off by default.
- Unknown marker names are copied untouched.

Only `skip` exists today. Directives are opt-in so files that mention marker text in
prose are copied unchanged unless the entry enables them.

---

## Parts

Parts are named bundles of entries. `base` is required and is selected by default.
Other parts are optional.

```yaml
parts:
  base:
    description: Core project files
    include:
      - /Makefile
      - /go.mod
  ci:
    description: GitHub Actions workflows
    include:
      - /.github/
  docs:
    description: Hugo documentation site
    include:
      - /docs/
      - /mk/docs.mk
```

The `base` key is required but its `include` list may be empty. This lets a repository
ship only optional parts. Every other part must contain at least one entry.

Select parts with `--only`:

```bash
sombra local init <uri> --only base,ci
sombra local update <id> --only ci
```

`--only` replaces the selection. It does not add to it. `base` is only implicit when
no part is named during an update. In these examples, the first command selects `base`
and `ci`. The second command selects `ci` only and omits the base files.

`sombra local init` without `--only` selects every part. The selected parts are stored
in the target's `sombra.yaml`. Updating without `--only` uses the stored selection.
When the stored selection is empty, Sombra uses `base`.

Two selected entries that select the same file with the same specificity fail the
update. Keep overlapping parts disjoint, or make one entry more specific.

Parts are flat. A part cannot include another part. Compose different templates in
the target's `sombra.yaml` instead:

```yaml
templates:
  - id: app
    uri: https://github.com/acme/app-template.git
    parts: [base, ci]
  - id: common-ci
    uri: https://github.com/acme/common-ci.git
    parts: [ci]
```

The second template copies only its `ci` part. Its `base` part is not copied, so the
two templates do not collide on shared base files.

Dropping a part makes its files prune candidates. They are removed only with
`--prune`, and only when they are unmodified.

---

## The Lock File

The target project keeps `.sombra/lock.yaml`. Commit it.

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
        part: base
        source: /Makefile
      /cmd/app/main.go:
        hash: sha256:...
        mode: seed
        part: base
        source: /cmd/app/main.go
```

The lock records the hash of the bytes Sombra wrote. It is how `sync` tells an edited
file from an untouched one. Do not edit it by hand.

If the lock is missing, every existing destination is a conflict. Use `--adopt` when
the content already matches, or `--force` to overwrite.

---

## Update Flow

`sombra local update`:

1. Reads the target `sombra.yaml`.
2. Fetches the template at the requested tag.
3. Resolves the selected parts.
4. Renders overlays, directives and rewrites for every file.
5. Plans an action for every destination.
6. Writes, keeps or reports conflicts.
7. Updates the lock and the target configuration.

Nothing is written if a transform, a path escape or a duplicate destination fails.

Reports use these labels: `created`, `updated`, `unchanged`, `kept`, `adopted`,
`removed`, `prunable` and `conflict`.

---

## Safety

- Only paths selected by the active parts are written.
- A `sync` file is overwritten only when it is unmodified.
- A `seed` file is never overwritten.
- A file is deleted only when it is in the lock, gone from the template, unmodified
  and `--prune` is passed.
- Files recorded with `--adopt` are never pruned.
- Destination paths must stay inside the target root.
- Writes are atomic.
- File modes are preserved.
- Binary files are copied byte for byte.
- Symlinks are skipped, never followed.

---

## Limitations

- One source file produces one destination. Multiple outputs are not supported.
- Rewrites are text, regex and structured edits. There is no AST mutation.
- A template has one definition file.
- Parts are flat and cannot include other parts.
- `--method diff` no longer exists. Updates use the lock.
- The definition shape is not backward compatible with `.sombra/default.yaml` or a
  top-level `include` list.

## Migrating from the Flat Include List

Schema 2 no longer accepts a top-level `include` list. Move it under the required
`base` part. The schema number does not change:

```yaml
# before
schema: 2
include:
  - /Makefile
```

```yaml
# after
schema: 2
parts:
  base:
    include:
      - /Makefile
```

Next: [Start a Template](start-a-template.md).
