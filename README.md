# Sombra CLI

**Sombra** is a command-line tool that turns production repositories into reusable,
version-controlled project templates.

Use it to:

- start new projects with a consistent setup;
- reuse real, tested boilerplate without changing it;
- update shared scaffolding without overwriting local edits;
- keep business code out of the scaffold.

---

## Features

- Use any Git repository as a template source.
- Declare a required `base` part, optional parts and `exclude`.
- Copy optional bundles with named parts.
- Rewrite paths, names and content with ordered rules.
- Replace whole files with overlays.
- Strip template-only regions with in-file directives.
- Enforce architecture with import rules, ast-grep rules and ADRs.
- Update files safely with a lock file.
- Prune files that a template no longer produces.
- Version templates with Git tags.

---

## Install

### Option 1: via Go

```bash
go install github.com/yunier-rojas/sombra-cli/cmd/sombra@latest
```

### Option 2: prebuilt binaries

Download from [GitHub Releases](https://github.com/yunier-rojas/sombra-cli/releases).

### Option 3: build from source

```bash
git clone https://github.com/yunier-rojas/sombra-cli.git
cd sombra-cli
make build WHAT=sombra
```

See the [installation guide](https://yunier-rojas.github.io/sombra-cli/getting-started/installation/).

---

## Quick Start

### 1. Create a template

Turn an existing repository into a template:

```bash
cd my-app
sombra template init .
```

This writes `.sombra/template.yaml`. Review `parts.base`, `exclude` and `rewrites`, then
commit the file and tag a release.

### 2. Register a template

In a new project:

```bash
sombra local init https://github.com/acme/app-template.git --id app --only base,ci
```

This prompts for the template variables and writes `sombra.yaml`. Without `--only`,
every part is selected.

### 3. Apply and update

```bash
sombra local update app
sombra local update app --tag v1.0.0
```

Sombra writes the files and records hashes in `.sombra/lock.yaml`. Later updates only
overwrite files you have not edited. Edited files are reported as conflicts.

---

## Documentation

Full documentation: [https://yunier-rojas.github.io/sombra-cli/](https://yunier-rojas.github.io/sombra-cli/)

Key topics:

- [Installation](https://yunier-rojas.github.io/sombra-cli/getting-started/installation/)
- [Creating Templates](https://yunier-rojas.github.io/sombra-cli/templates/start-a-template/)
- [Template Concepts](https://yunier-rojas.github.io/sombra-cli/templates/concepts/)
- [sombra.yaml Config](https://yunier-rojas.github.io/sombra-cli/cli/sombra-file/)
- [Command Reference](https://yunier-rojas.github.io/sombra-cli/cli/commands/)

---

## Contributing

Issues and pull requests are welcome. Start with the [contact page](https://yunier-rojas.github.io/sombra-cli/contact/)
or open an [issue](https://github.com/yunier-rojas/sombra-cli/issues).

MIT licensed. Made by [@yunier](https://www.linkedin.com/in/yunier-rojas-garc%C3%ADa/).
