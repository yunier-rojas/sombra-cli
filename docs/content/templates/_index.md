---
title: Templates
type: docs
cascade:
  type: docs
aliases:
  - /sombra-templates/
---

## Overview

Templates turn real code into reusable, versioned project generators. The source
repository stays a normal project.

This section explains how to author, structure and publish a template.

---

## What Is a Template?

A template is a git repository with a definition file at `.sombra/template.yaml`. The
definition selects files, assigns ownership modes, rewrites the project identity and
declares a required `base` part and optional parts.

A target project stores its template choices in `sombra.yaml` and records what Sombra
wrote in `.sombra/lock.yaml`.

---

## Who Is This For?

This guide is for developers, consultants and teams who want to:

- share boilerplate across projects or clients;
- keep a consistent structure and tooling setup;
- reuse existing code without duplication;
- update shared scaffolding without overwriting local edits.

---

## Get Started

- [Start a Template](start-a-template.md): convert a real project into a template.
- [Concepts](concepts.md): selection, modes, rewrites, overlays, directives, parts and
  the lock file.
- [Best Practices](best-practices.md): keep a template maintainable.

To use a template instead, see [Use sombra-cli as a Template](../getting-started/use-sombra-as-template.md).
