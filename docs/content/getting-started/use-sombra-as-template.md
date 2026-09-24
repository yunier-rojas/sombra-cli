---
title: Use sombra-cli as a Template
weight: 2
aliases:
  - /user-guide/use-sombra-as-template/
---

## Overview

The `sombra-cli` repository is also a template. Point `sombra` at it to scaffold a
new Go CLI with the same structure: domain entities, use cases, a composition root,
a logger, and build, CI, QA, and docs scaffolding.

This page covers using that template. To build your own, see
[Start a Template](../templates/start-a-template.md).

---

## What the Template Provides

Core files:

- `cmd/<project>/main.go`: CLI entry point built with go-arg
- `internal/core/entities`: domain types
- `internal/core/usecases`: use case interactors and ports
- `internal/runtime`: dependency injection and composition
- `internal/frameworks/logger`: zerolog adapter
- `Makefile` and `mk/`: build, test, QA, and docs targets
- `.editorconfig`, `.gitignore`, `.dockerignore`, `LICENSE`, `README.md`, `AGENTS.md`

Optional components, enabled by variables:

- `.github/` workflows (`include_ci`)
- `qa/` import rules (`include_qa`)
- `docs/` Hugo site (`include_docs`)

The template copies a curated subset. Business logic and the remaining framework
adapters are left out on purpose, so you build on a lean skeleton instead of
inheriting a finished product.

---

## Prerequisites

- `sombra` installed (see [Install sombra](installation.md))
- Git
- Go 1.26 or later, to build the generated project

---

## Step 1: Register the Template

Create the target project directory and register the template:

```bash
mkdir hello-cli
cd hello-cli
sombra local init github.com/yunier-rojas/sombra-cli --id starter
```

`--id` gives the template a short name you reuse on updates. Without it, `sombra`
derives an id from the repository name.

`sombra local init` clones the template, reads the variables it declares, and
prompts for a value for each. It writes them to `sombra.yaml`. It does not copy
files yet.

---

## Step 2: Answer the Variables

| Variable       | Purpose                                              |
|----------------|------------------------------------------------------|
| `github_user`  | Owner used in the module path and metadata           |
| `project`      | Project name, used for the command and identifiers   |
| `repository`   | Repository name, used in the module path             |
| `author`       | Author name                                          |
| `include_ci`   | `true` copies the `.github/` workflows               |
| `include_docs` | `true` copies the `docs/` site                       |
| `include_qa`   | `true` copies the `qa/` import rules                 |

Example `sombra.yaml`:

```yaml
templates:
  - id: starter
    uri: github.com/yunier-rojas/sombra-cli
    vars:
      github_user: acme
      project: hello-cli
      repository: hello-cli
      author: Acme Inc.
      include_ci: "true"
      include_docs: "false"
      include_qa: "false"
```

The values are reused on every update, so you answer once.

---

## Step 3: Apply the Template

```bash
sombra local update starter
```

This copies the files into the current directory and records the applied tag in
`current`. The default `--method copy` seeds the Go skeleton. Skeleton files are
marked `copy_only`, so later `--method diff` updates never overwrite code you edit.

---

## Step 4: Build and Test the Project

The generated project compiles on its own. From the project directory, build
everything and run the version command:

```bash
go build ./...
go run ./cmd/hello-cli version
```

`go run` prints the project name and the version. The version is `dev` until a
build injects one.

Build a binary with the Makefile:

```bash
VERSION=v0.1.0 make build
./build/hello-cli-linux-amd64 version
```

`make build` writes the binary to `build/`. The name follows
`<project>-<goos>-<goarch>`, with a `.exe` suffix on Windows. Replace the OS and
architecture in the example with your own. On a Git repository, `make build`
derives the version from `git describe`. In a directory that is not a Git
repository, pass `VERSION` explicitly as shown, otherwise the binary reports an
empty version.

Run the tests:

```bash
make test
```

The skeleton ships without tests, so coverage is 0%. Add tests next to the code
as `*_test.go` files.

---

## Updating the Project

When the template changes, update to a new tag:

```bash
sombra local update --tag v1.0.0 --method diff starter
```

`--method diff` refreshes everything except the `copy_only` skeleton, so your edits
are preserved. See [CLI Commands](../cli/commands.md#sombra-local-update).

---

## Limitations

- The generated project is a skeleton, not a finished product. It ships a working
  `version` command as the sample vertical slice (command, runtime and use case);
  add your own commands and business logic.
- Renames are text based. Review the output when copied files contain names that
  also appear in your own code.
- Optional components are copied at generation time from the `include_*` values.
  Setting a value to `false` later does not remove files that were already copied.

---

Next: [CLI Commands](../cli/commands.md) or the [`sombra.yaml` file](../cli/sombra-file.md).
