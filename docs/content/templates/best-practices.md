---
title: Best Practices
weight: 3
aliases:
  - /sombra-templates/best-practices/
---

## Best Practices for Creating Templates

Follow these practices when you design a `.sombra/template.yaml`.

---

### 1. Curate the base part

List only the files that belong in a new project. Everything else stays in the source
repository and is never copied.

Prefer explicit directories and files over broad patterns. `include` does not accept
globs, so `parts.base.include` is the review surface: if a file is not listed, it cannot
leak into a generated project.

---

### 2. Default to seed for code the consumer will edit

Use `mode: seed` for commands, domain types, composition roots and module files. Use
the default `sync` for generated or shared scaffolding such as `Makefile`, `mk/` and
CI configuration.

A `seed` file is never overwritten. That protects the consumer's work.
A `sync` file is updated only while it is unmodified, so edits still surface as
conflicts instead of being lost.

---

### 3. Order rewrites from specific to general

Rules run in order. Put the full module path first, then the repository slug, then the
project name.

```yaml
rewrites:
  - find: "github.com/acme/app"
    replace: "{{ .module }}"
    in: [all]
  - find: "app"
    as: word
    replace: "{{ .project }}"
    in: [all]
```

Use `as: word` for short identifiers. It avoids changing derived names by accident.

---

### 4. Scope rewrites to files

A global rule is convenient, but a scoped rule is safer. Use `files` to limit a rule
to the files where the text means the same thing.

```yaml
- files: [/internal/core/entities/app.go]
  find: "app.go"
  replace: "entities.go"
  in: [name]
```

---

### 5. Use overlays for whole-file differences

When a file must be much smaller in generated projects, keep the full file in the
repository and put the reduced version under `.sombra/overlay/`.

```text
.sombra/overlay/README.md
```

Overlays keep the diff obvious and avoid fragile regexes.

---

### 6. Use directives for small regions

Wrap template-only code in `sombra:skip` and `sombra:end` markers and enable
`directives: true` on the owning entry.

Keep the markers out of pages that are copied and that document the marker syntax.
Those files need `directives` disabled or an overlay.

---

### 7. Move optional trees into parts

CI, docs and QA are good part candidates. Parts keep the base small and make the
consumer's choice explicit. Every definition still needs a `base` part, but it may be
empty.

Name parts after the capability, not the directory. Write a description. Consumers see
it in `local init` and `local update` errors.

Parts are flat. To compose several templates, select parts from each in the target's
`sombra.yaml`.

---

### 8. Test both a fresh and an existing project

Test twice:

1. An empty directory. Every file is created.
2. A directory with an edited `sync` file. The update reports a conflict.

Also test a dropped part with and without `--prune`.

---

### 9. Commit the definition and the lock

Commit `.sombra/template.yaml` in the template repository. Commit `.sombra/lock.yaml`
in each consumer project. The lock is how updates distinguish an untouched file from
an edited one.

Do not edit the lock by hand.

---

### 10. Tag releases

Use semantic tags (`v1.0.0`, `v1.1.0`). Consumers can pin a tag with
`--tag`, or use the latest tag by default.

---

### 11. Keep the source repository working

A template repository is also a normal project. Do not break it to satisfy the
template definition. Use overlays and directives to keep template-only content out of
the generated result.

---

For a walkthrough, see [Start a Template](start-a-template.md). For the full
reference, see [Template Concepts](concepts.md).
