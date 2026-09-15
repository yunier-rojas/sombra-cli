---
title: About
type: page
description: Learn about Yunier and the Sombra project for automating project setup and structure.
---


[//]: # (sombra:skip)

## About the Creator

Hi, I’m **Yunier**, a software developer focused on automation and tooling.  
Sombra started as a side project to avoid setting up the same project by hand.

## About the Project

**Sombra CLI** is an open-source command-line tool that turns existing code into reusable templates.  
It generates new projects from a template without changing the original repository.

The project started as an internal tool and is now open source under the MIT License.  
Sombra is used to:

- Set up boilerplate code
- Manage many similar services (for example, in microservice architectures)
- Share a base across projects or clients

## Philosophy

Most scaffolding tools keep the template in a separate repository. Files use placeholders and template logic, so the template becomes a second copy of a project that someone maintains by hand. Nothing links it to the real project, so the two diverge over time.

Sombra has no separate template. The template is the project itself plus a `.sombra/default.yaml` that records the changes to apply: copy the files, drop what is not needed, and rename what remains. That is the routine developers already follow by hand, written down and versioned.

- The template stays current because it is the production code.
- The source repository keeps working; nothing is rewritten for the template.
- Sombra renames and removes. It does not create files that are absent from the repository, so conditional file sets and per-variable loops are out of scope.


[//]: # (sombra:end)

## Want to Learn More?

Check out the [Contact](contact.md) page to reach out, or explore the [User Guide](user-guide/index.md) to get started.
