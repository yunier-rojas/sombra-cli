---
title: Use sombra-cli as a Template
weight: 2
aliases:
  - /user-guide/use-sombra-as-template/
---

## Overview

The `sombra-cli` repository is also a template. Point `sombra` at it to scaffold a
new Go CLI with the same structure: domain entities, use cases, a composition root,
a logger and the build scaffolding.

This page covers using that template. To build your own, see
[Start a Template](../templates/start-a-template.md).

---

## What the Template Provides

Core files (`base` part):

- `cmd/<project>/main.go`: CLI entry point built with go-arg
- `internal/core/entities`: shared domain types
- `internal/core/usecases`: the version use case
- `internal/runtime`: dependency injection and composition
- `internal/frameworks/logger`: zerolog adapter
- `Makefile` and `mk/`: build and test targets
- `.editorconfig`, `.gitignore`, `LICENSE`, `README.md`, `AGENTS.md`,
  `opencode.jsonc`

Optional parts:

- `ci`: `.github/` workflows
- `docs`: the Hugo documentation site
- `qa`: QA import rules, ast-grep rules, ADRs and ADR checks

The template copies a curated subset. Business logic and the remaining framework
adapters are left out on purpose, so you build on a lean skeleton.

The copied Go files are `seed`. Sombra writes them once and never overwrites your
edits.

---

## Prerequisites

- `sombra` installed (see [Install sombra](installation.md))
- Git
- Go 1.26 or later, to build the generated project

---

## Step 1: Register the Template

Create the target directory and register the template with the parts you want:

```bash
mkdir hello-cli
cd hello-cli
sombra local init https://github.com/yunier-rojas/sombra-cli.git --id starter --only base,ci
```

`--only` accepts a comma separated list, for example `--only base,ci,docs,qa`.
`--only` replaces the selection, so name `base` when you want the core files. When
`--only` is omitted, every part is selected.

`sombra local init` clones the template, validates the parts and prompts for the
variables. It writes `sombra.yaml` and does not copy files yet.

---

## Step 2: Answer the Variables

| Variable      | Purpose                                            |
|---------------|----------------------------------------------------|
| `project`     | Project and command name                           |
| `module`      | Go module path, for example `example.com/hello-cli` |
| `repository`  | Repository slug, for example `acme/hello-cli`       |
| `github_user` | GitHub user or organization                         |
| `author`      | Author name                                         |

Example `sombra.yaml`:

```yaml
templates:
  - id: starter
    uri: https://github.com/yunier-rojas/sombra-cli.git
    parts: [base, ci]
    vars:
      project: hello-cli
      module: example.com/hello-cli
      repository: acme/hello-cli
      github_user: acme
      author: Acme Inc.
```

The values are reused on every update, so you answer once.

---

## Step 3: Apply the Template

```bash
sombra local update starter
```

This copies the files and writes `.sombra/lock.yaml`. Commit the lock file.

The command reports `created`, `updated`, `unchanged`, `kept`, `adopted`, `removed`,
`prunable` and `conflict`. It exits with a non-zero status when conflicts remain.

Use `--dry-run` first when you are unsure:

```bash
sombra local update starter --dry-run
```

---

## Step 4: Build and Test the Project

The generated project compiles on its own:

```bash
go build ./...
go run ./cmd/hello-cli version
```

Build a binary with the Makefile:

```bash
VERSION=v0.1.0 make build
./build/hello-cli-linux-amd64 version
```

The binary name follows `<project>-<goos>-<goarch>`, with a `.exe` suffix on Windows.

Run the tests:

```bash
make test
```

The skeleton ships without tests, so coverage is 0%. Add tests next to the code as
`*_test.go` files.

---

## Updating the Project

```bash
sombra local update starter --tag v1.0.0
```

`sync` files are updated when they are unmodified. `seed` files are kept. Edited
`sync` files are reported as conflicts and left untouched.

Add or change parts with `--only`:

```bash
sombra local update starter --only base,ci,docs
```

Dropping a part reports its files as `prunable`. Remove them with:

```bash
sombra local update starter --only base,ci --prune
```

Pruning only removes unmodified files.

If you already have a project with the same files, use `--adopt` to record the files
whose content already matches:

```bash
sombra local update starter --adopt
```

---

## Limitations

- The generated project is a skeleton. It ships a working `version` command as the
  sample vertical slice. Add your own commands and business logic.
- Renames are text based. Review the output when copied files contain names that also
  appear in your own code.
- Parts are applied only when selected. `base` is selected by default, but naming any
  other part omits it unless you also name `base`. Removing a part requires `--prune`
  to delete its files.
- The template definition shape is not compatible with `.sombra/default.yaml` or a
  top-level `include` list.

---

Next: [CLI Commands](../cli/commands.md) or the [`sombra.yaml` file](../cli/sombra-file.md).
