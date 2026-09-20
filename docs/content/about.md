---
title: About
description: What sombra-cli is, why it exists, and how the pieces fit together.
weight: 1
---

[//]: # (sombra:skip)

## The Problem

Starting a new project usually means copying an existing one by hand: rename the module, update identifiers, delete what does not apply. The result is easy to get wrong, and later changes to the source never reach the copy.

The common alternatives have limits:

- **Scaffolding generators**: the template is a second repository with placeholders and template logic. It must be maintained by hand and drifts from the real code.
- **Copy and paste**: quick for the first project, but fixes, CI changes, and tooling updates never propagate.
- **Rewriting the source as a template**: breaks the original repository as production code.

The gap is most visible for teams that want scaffolding to be:

- Based on code that actually runs in production
- Updatable after the project is generated
- Versioned and reviewable
- Not another repository to maintain by hand

## The Idea

`sombra-cli` removes the separate template. A template is an existing repository plus one file:

- **The project itself**: real code that builds, runs, and is maintained as usual.
- **`.sombra/default.yaml`**: the transformation rules. Rename paths, filenames, and content; drop template-only blocks; copy the rest.

Generating a project clones the template at a tagged version, applies the rules, and writes the result. The template repository is never modified.

The CLI covers both sides of the workflow:

- **Use a template** (`sombra local init`, `sombra local update`): register it in a project and refresh it when the template changes.
- **Author a template** (`sombra template init`): derive a starting definition from an existing project.

## Why Build It?

`sombra-cli` keeps scaffolding close to the code and avoids a second artifact to maintain:

- Keep the template current because it is production code.
- Update generated projects instead of forking them, with the applied version recorded.
- Make every transformation explicit and reviewable in one file.
- Stay open and portable: MIT licensed, with no platform to host, no database, and no service to run.

## Principles

- **The template is the project**: there is no separate template format to keep in sync.
- **Transform, do not generate**: `sombra` renames, removes, and copies files that exist. It does not create files that are absent from the repository.
- **Explicit over implicit**: rules live in `.sombra/default.yaml`, state lives in `sombra.yaml`.
- **Versioned by Git**: templates are consumed by tag, and each project records the version it was generated from.
- **Readable and auditable**: plain Go, YAML, and Go templates.

[//]: # (sombra:end)

## Learn More

- Getting Started: [Installation and first project](/getting-started/)
- Templates: [Authoring a template](/templates/)
- CLI: [Commands and configuration](/cli/)
