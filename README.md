# Sombra CLI

**Sombra** is an open-source command-line tool that helps you automate project scaffolding by turning production-ready repositories into reusable, version-controlled templates.

Built for developers, consultants, and teams who want to:

* ⚡ Quickly start new projects with consistent setup
* 🧱 Reuse real, tested code without changing it
* 🔄 Keep projects up to date with shared boilerplate

---

## ✨ Features

* ✅ Use any Git repository as a template source
* ⚙️ Define flexible rules using Go templates + YAML
* ♻️ Reuse code without modifying production files
* 🔍 Match and transform paths, filenames, and content
* 🏷 Semantic versioning with Git tags

---

## 📦 Install

### Option 1: via Go

```bash
go install github.com/yunier-rojas/sombra-cli/cmd/sombra@latest
```

### Option 2: Prebuilt Binaries

Download from [GitHub Releases](https://github.com/yunier-rojas/sombra-cli/releases)

### Option 3: Build from source

```bash
git clone https://github.com/yunier-rojas/sombra-cli.git
cd sombra-cli
make build WHAT=sombra
```

See [Installation Guide](https://yunier-rojas.github.io/sombra-cli/user-guide/installation.html) for more details.

---

## 🚀 Quick Start

### 1. Create a Template

Convert a production repo into a template:

```bash
sombra template init ./my-app
```

This creates `.sombra/default.yaml`.

### 2. Register a Template

In a new repo, register the template:

```bash
sombra local init https://github.com/your-org/your-template
```

This prompts for the variables the template declares and writes `sombra.yaml`:

```yaml
templates:
  - uri: https://github.com/your-org/your-template
    vars:
      project: New API
```

### 3. Apply and Update a Project

```bash
sombra local update --tag v1.0.0 --method copy https://github.com/your-org/your-template
```

---

## 📖 Documentation

Full docs available at 👉 [https://yunier-rojas.github.io/sombra-cli/](https://yunier-rojas.github.io/sombra-cli/)

Key topics:

* [Installation](https://yunier-rojas.github.io/sombra-cli/user-guide/installation.html)
* [Creating Templates](https://yunier-rojas.github.io/sombra-cli/sombra-templates/start-a-template.html)
* [sombra.yaml Config](https://yunier-rojas.github.io/sombra-cli/user-guide/sombra-file.html)
* [Command Reference](https://yunier-rojas.github.io/sombra-cli/user-guide/commands.html)

---

## 🤝 Contributing

Issues and PRs welcome! Start with the [Contact page](https://yunier-rojas.github.io/sombra-cli/contact.html) or open an [Issue](https://github.com/yunier-rojas/sombra-cli/issues).

MIT licensed. Made with ❤️ by [@yunier](https://www.linkedin.com/in/yunier-rojas-garc%C3%ADa/)
