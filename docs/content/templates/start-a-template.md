---
title: Start a Template
weight: 2
aliases:
  - /sombra-templates/start-a-template/
---

## Overview

This guide walks you through creating a template from an existing codebase — no
changes to the source code required. The worked example is this repository's definition:
[`.sombra/default.yaml`](https://github.com/yunier-rojas/sombra-cli/blob/main/.sombra/default.yaml)
in this repository, which turns `sombra-cli` into a reusable Go project skeleton.

Use this when you want to:

- Reuse a production-ready project as a starting point
- Share best practices and config across multiple teams or services
- Turn internal tooling into reusable templates

---

## The example

`sombra-cli` is a Go CLI. Its template copies a curated set of files — the CLI entry
point, the composition root, the domain entities, the logger and the build / docs / CI /
QA scaffolding — into a new project and renames the module, commands and identifiers
along the way. The product's business logic and the rest of the frameworks are left out
on purpose, so the result is a lean structure to build on rather than a project that
compiles as-is.

Every step below is taken from that file.

---

## Step 1: Create `.sombra/default.yaml`

Add the definition at the root of the repository you want to reuse:

```text
.
└── .sombra/
    └── default.yaml
```

Everything inside `.sombra/` is treated as template metadata: it is never copied to the
target project and can hold reference material for the template itself.

---

## Step 2: Declare the variables

Variables are the questions `sombra local init` asks. Here the answers are the new
project's identity plus three switches that enable optional components:

```yaml
vars:
  - project
  - module
  - repository
  - author
  - include_ci
  - include_docs
  - include_qa
```

The values are stored in the target project's `sombra.yaml` and reused on every update,
so consumers only answer once.

---

## Step 3: Apply global renames with an abstract pattern

An `abstract: true` pattern contributes mappings to every file it matches but never
includes a file on its own. This one turns the template's own names into the consumer's:

```yaml
patterns:
  - pattern: "/**/*"
    abstract: true
    default:
      "github.com/yunier-rojas/sombra-cli": "{{ .module }}"
      "yunier-rojas/sombra-cli": "{{ .repository }}"
      "sombra-cli": "{{ .project }}"
      "Yunier Rojas García": "{{ .author }}"
      "Yunier": "{{ .author }}"
      "word:sombra": "{{ .project }}"
```

`word:` only matches whole tokens, so `sombra-cli` and `sombra.yaml` are left for the
longer, more specific keys instead of being mangled by the generic `sombra` rule.

---

## Step 4: Copy the Go skeleton

The Go files are the heart of the template. Each pattern renames its file or the
identifiers inside it:

```yaml
  - pattern: "/cmd/sombra/main.go"
    copy_only: true
    content:
      "Local": "Greet"
      "subcommand:local": "subcommand:greet"
    block_directives: true

  - pattern: "/cmd/sombra/cmd_local_init.go"
    copy_only: true
    name:
      "cmd_local_init.go": "cmd_greeter.go"
    content:
      "LocalInit": "Greeter"
      "args.Template": "args.Say"
    block_directives: true
```

`copy_only: true` ties the pattern to the **copy** update method: the skeleton is
seeded by `sombra local update --method copy` (the default) and ignored by
`--method diff`, so routine updates never overwrite code the consumer has edited. See
[`copy_only`](concepts.md#copy_only-copy-method-only-patterns).

---

## Step 5: Hide template-only content

The source repository contains code that only exists to make it a working project.
Wrap those regions with `sombra:skip` / `sombra:end` markers and enable the block on
the owning pattern:

```yaml
  - pattern: "/cmd/sombra/main.go"
    block_directives: true
```

```go
var args struct {
	Local *LocalSubcommand `arg:"subcommand:local"`
	// sombra:skip
	Template *TemplateSubcommand `arg:"subcommand:template"`
	// sombra:end
}
```

Here the generated project keeps the `local` command but never exposes the
template-management command. The markers and every line between them are removed when
the file is copied. Markers can nest, and block directives are opt-in per pattern so
ordinary `sombra:` text is copied untouched. See
[In-file Block Directives](concepts.md#in-file-block-directives).

---

## Step 6: Add scaffolding and optional components

Everything else is declared the same way. The build and repository files are copied as
they are, with the project name substituted where it appears:

```yaml
  - pattern: "/Makefile"
    content:
      "word:sombra": "{{ .project }}"

  - pattern: "/.editorconfig"
  - pattern: "/.gitignore"
  - pattern: "/LICENSE"
```

Optional components are gated with `when`, a Go template expression evaluated against
the variables before the definition is parsed:

```yaml
  - pattern: "/.github/**"
    when: '{{ eq .include_ci "true" }}'
    content:
      "word-ci:sombra": "{{ .project }}"
      "sombra-": "{{ .project }}-"

  - pattern: "/docs/**"
    abstract: true
    when: '{{ eq .include_docs "true" }}'

  - pattern: "/mk/qa.mk"
    when: '{{ eq .include_qa "true" }}'
```

When the expression is false the pattern is skipped entirely, so optional trees such as
`.github/`, `docs/` and `qa/` need no `except` denylist. See
[`when`](concepts.md#when-conditional-patterns) and the full
[Pattern Field Reference](concepts.md#pattern-field-reference).

---

## Step 7: Tag a release

To make the template available for versioned use:

```bash
git tag v1.0.0
git push origin v1.0.0
```

---

## Using the template

Consumers register it and answer the variables:

```bash
sombra local init https://github.com/yunier-rojas/sombra-cli.git
```

That records them in the target project's `sombra.yaml`:

```yaml
templates:
  - uri: https://github.com/yunier-rojas/sombra-cli.git
    vars:
      project: hello-cli
      module: github.com/acme/hello-cli
      repository: acme/hello-cli
      author: Acme Inc.
      include_ci: "true"
      include_docs: "false"
      include_qa: "false"
```

Running `sombra local update https://github.com/yunier-rojas/sombra-cli.git` then copies the files and
records the applied tag in `current`. Because the Go skeleton is `copy_only`, the first
`--method copy` seeds it and later `--method diff` updates everything else.

---

## Summary

To convert any repo into a template:

1. Add `.sombra/default.yaml`
2. Declare `vars` for the consumer to answer
3. Rename globally with an `abstract` pattern
4. Mark seed-once files `copy_only` and map their `name` / `content`
5. Hide template-only code with `sombra:skip` block directives
6. Gate optional components with `when`
7. Tag a release and consume it with `sombra local init`

Next: learn about [Template Concepts](concepts.md) to understand pattern structure in depth.
