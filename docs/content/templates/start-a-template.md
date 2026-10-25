---
title: Start a Template
weight: 2
aliases:
  - /sombra-templates/start-a-template/
---

## Overview

This guide turns an existing repository into a template. The example is this
repository's own definition:
[`.sombra/template.yaml`](https://github.com/yunier-rojas/sombra-cli/blob/main/.sombra/template.yaml).

The result copies a curated set of files, renames the project identity and leaves the
rest of the repository alone. The source repository keeps working as a normal project.

---

## Step 1: Generate a definition

Run:

```bash
sombra template init .
```

The command writes `.sombra/template.yaml`. It uses tracked files from git, or walks
the directory when git is not available.

The generated file contains:

- a required `parts.base` with entries grouped by top level directory;
- `vars` seeded from `go.mod`, the git remote and the git user;
- an ordered rewrite list for the detected identity;
- your `--exclude` patterns.

Review the file before continuing. Remove business logic from `parts.base` or add it to
`exclude`. The generator is conservative: a new top level directory is not copied
until you add it.

Useful flags:

```bash
sombra template init --only '/cmd/**' --only '/internal/**' .
sombra template init --exclude '*.log' --exclude '/secrets/**' .
sombra template init --dry-run .
```

`--only` accepts globs and writes concrete paths. `--dry-run` prints the definition
without writing it. `--force` overwrites an existing definition.

---

## Step 2: Declare variables

Keep the variables the consumer must answer. Remove the ones that are not used.

```yaml
vars:
  project:
    description: Project and command name
    default: app
  module:
    description: Go module path
    default: example.com/app
  author: {}
```

`sombra local init` prompts in alphabetical order and offers the defaults.

---

## Step 3: Choose seed and sync files

Most scaffolding should be `sync`. Files the consumer will edit should be `seed`.

```yaml
parts:
  base:
    include:
      - /Makefile
      - /mk/build.mk
      - /mk/test.mk
      - /go.mod

      - path: /cmd/app/main.go
        mode: seed
        directives: true
      - path: /internal/runtime/container.go
        mode: seed
```

`sync` updates the file when it is unmodified. `seed` writes it once.

---

## Step 4: Write the rewrites

Start with the full identity, then the less specific names. The first rule wins for
overlapping text because rules run in order.

```yaml
rewrites:
  - find: "github.com/acme/app"
    replace: "{{ .module }}"
    in: [path, name, content]

  - find: "acme/app"
    replace: "{{ .repository }}"
    in: [path, name, content]

  - find: "app"
    as: word
    ignore_case: true
    replace: "{{ .project }}"
    in: [path, name, content]

  - files: [/internal/core/entities/app.go]
    find: "app.go"
    replace: "entities.go"
    in: [name]
```

Use `word` for short names. It avoids changing `app-cli`, `app.yaml` or `app_name`
when you only meant the bare name.

---

## Step 5: Hide template-only content

Two options:

### Directives

Enable directives on the file and wrap the region:

```yaml
- path: /cmd/app/main.go
  mode: seed
  directives: true
```

```go
// sombra:skip
templateOnly := true
// sombra:end
```

The marker lines and the block are removed from the copy.

### Overlays

When a whole file differs, place the reduced version under `.sombra/overlay/`:

```text
.sombra/overlay/README.md
```

The overlay is rendered with the variables and used instead of the repository's
`README.md`.

---

## Step 6: Add optional parts

`base` is required. Move optional trees into other parts:

```yaml
parts:
  base:
    include:
      - /Makefile
  ci:
    description: GitHub Actions workflows
    include:
      - /.github/
  docs:
    description: Hugo documentation site
    include:
      - /docs/
      - /mk/docs.mk
  qa:
    description: QA configuration
    include:
      - /qa/
      - /mk/qa.mk
```

Consumers select them. `--with` replaces the selection, so name `base` when you want
the default files too:

```bash
sombra local init https://github.com/acme/app.git --with base,ci,docs
```

---

## Step 7: Test in a sandbox

Create a throwaway directory with a `sombra.yaml` and run:

```bash
mkdir /tmp/sandbox && cd /tmp/sandbox
sombra local update --dry-run <id-or-uri>
sombra local update <id-or-uri>
```

Check:

- the expected files are created;
- paths and names are renamed;
- skipped regions are gone;
- `seed` files are present;
- selected parts are copied;
- unselected parts are not.

Run the update again. It should report `unchanged` or `kept`, with no conflicts.

Edit a `sync` file and run the update again. It should report a conflict. Restore the
file, or use `--force` when you meant to overwrite.

---

## Step 8: Tag a release

```bash
git add .sombra/template.yaml
git commit -m "Add Sombra template"
git tag v1.0.0
git push origin main --tags
```

Consumers can now pin the tag:

```bash
sombra local update v1.0.0 <id-or-uri>
```

---

## Summary

1. Generate with `sombra template init`.
2. Curate `parts.base` and `exclude`.
3. Declare `vars`.
4. Mark editable files `seed`.
5. Write ordered rewrites.
6. Hide template-only content with directives or overlays.
7. Move optional trees into parts.
8. Test in a sandbox, then tag.

Next: [Template Concepts](concepts.md) and [Best Practices](best-practices.md).
