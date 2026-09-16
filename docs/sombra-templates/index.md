---
title: Sombra Templates
---

## Overview

Sombra templates turn real code into reusable, versioned project generators. You do not modify the original repository.

This section explains how to author, structure, and publish a template that other people can use with the `sombra` CLI.

---

## What Is a Sombra Template?

A **Sombra template** is a Git repository that contains:

- Real project files (application, service, etc.)
- A `.sombra/default.yaml` file defining transformation rules

With a `sombra.yaml`, the CLI generates new projects by replacing values in filenames, content, and directory paths using Go templates.

---

## Who Is This For?

This guide is for developers, consultants, and teams who want to:

- Share boilerplate across projects or clients
- Keep a consistent project structure and tooling setup across services
- Reuse existing code without duplication

---

## Get Started

Ready to build a template?

- [Start a Template](start-a-template.md) — Convert a real project into a Sombra template
- [Concepts](concepts.md) — Templates, patterns, and mappings
- [Best Practices](best-practices.md) — Keep a template maintainable

To use a template instead, see the [User Guide](../user-guide/index.md).
