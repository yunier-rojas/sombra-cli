---
title: Template Concepts
---

## Overview

This page explains the internal structure and logic of **Sombra Templates**, including how files are transformed using
Go templates, YAML definitions, and mappings.

A Sombra template is composed of:

- Real source files (code, configs, etc.)
- A `.sombra/default.yaml` file with transformation rules

The CLI uses this configuration to generate new projects with modified paths, filenames, and content based on input
variables.

---

## Directory Structure

Sombra templates are Git repositories that contain a `.sombra` directory with a `default.yaml` file:

```

.
└── .sombra/
└── default.yaml

```

Only one definition file is currently supported.

---

## Go Templates + YAML

Sombra uses [Go’s templating engine](https://pkg.go.dev/text/template) to apply variable substitutions. It also
includes [Sprig](https://masterminds.github.io/sprig/) functions for string, list, and math utilities.

All variables come from the target repo’s `sombra.yaml`.

---

## Template Structure

### `vars`

A list of variable names that are expected to be defined by the user when applying the template:

```yaml
vars:
    - project
    - author
```

---

### `patterns`

Each pattern block applies transformation rules to files matching a glob-style pattern.

```yaml
patterns:
    -   pattern: "*"
        abstract: true
        default:
            project-template: "{{ .project | kebabcase }}"
```

---

### Pattern Matching Categories

Each pattern supports three transformation scopes:

* `path`: Folder and subdirectory names
* `name`: Filenames
* `content`: File contents

The `default` section applies to all three.

---

## Pattern Field Reference

| Field              | Description                                          |
|--------------------|------------------------------------------------------|
| `pattern`          | Required. Glob to match files                        |
| `abstract`         | If true, this rule applies only as a base pattern    |
| `verbatim`         | If true, copies matched files without content edits  |
| `copy_only`        | If true, only the `copy` update method applies it    |
| `block_directives` | If true, evaluates in-file `sombra:` markers         |
| `delete`           | If true, matched target paths are tombstoned         |
| `replace`          | Replaces matched file content with a `.sombra/` file |
| `when`             | If false, skips the whole pattern (Go template)      |
| `default`          | General-purpose replacements                         |
| `path`             | Folder path replacements                             |
| `name`             | Filename replacements                                |
| `content`          | Content-specific replacements                        |
| `except`           | Files to exclude from the rule                       |

### Tombstones (`delete`)

A `delete: true` pattern does not copy anything. It declares paths in the **target**
project that the template no longer manages, so renames and removals can be cleaned up:

```yaml
-   pattern: "/legacy/**"
    delete: true
```

Tombstones are only acted on by `sombra local update`. Matching paths are reported by
default and removed only when `--prune` is passed. Support files such as `.git`,
`.sombra` and editor caches are never touched.

---

## How Replacements Are Applied

When a file matches multiple patterns:

1. All relevant mappings are collected
2. Mappings are sorted by key length (longest first)
3. Each key-value pair is applied sequentially to:

    * Path
    * Name
    * Content

This ensures deterministic and consistent replacements.

---

## Replacement Directives

By default a mapping key is matched literally. Prefix the key with a directive to opt
into a different behavior:

| Prefix     | Behavior                                               |
|------------|--------------------------------------------------------|
| *(none)*   | Literal substring replacement                          |
| `re:`      | [RE2](https://github.com/google/re2/wiki/Syntax) regex |
| `rex:`     | Regex with a per-match Go template (`[[ ]]`)           |
| `word:`    | Literal replacement only for whole tokens              |
| `ci:`      | Case-insensitive literal replacement                   |
| `word-ci:` | Case-insensitive whole-token replacement               |
| `file:`    | Replace the whole file content                         |

Keys with an unrecognized prefix (for example `https://example.com`) stay literal.

### `re:` regular expressions

```yaml
"re:description = .*\n": ""
```

This removes lines that start with `description = ...`. The value supports capture
references such as `$1`, including named groups via `${name}`.

### `rex:` regex with per-match templates

`rex:` is like `re:`, but the replacement is a Go template evaluated once per match,
with the submatches exposed and [Sprig](https://masterminds.github.io/sprig/) functions
available. Because the template definition is itself rendered with `{{ }}`, `rex:`
replacements use `[[ ]]` delimiters:

```yaml
"rex:(?P<name>\\w+)": "[[ upper .Match.name ]]"
```

| Expression               | Meaning                  |
|--------------------------|--------------------------|
| `[[ .Value ]]`           | The whole match          |
| `[[ .Match.name ]]`      | Named submatch           |
| `[[ index .Match "1" ]]` | Numbered submatch        |
| `[[ index .Groups 1 ]]`  | Numbered submatch (list) |

For simple substitutions `re:` is enough: Go's regexp expansion already understands
`${name}` for named groups.

### `word:` whole tokens

`word:` replaces the literal only when it is not part of a larger token. Token
characters are ASCII letters, digits, `_`, `.` and `-`, which keeps derived names
intact:

```yaml
"word:sombra": "my-app"
```

| Input            | Result           |
|------------------|------------------|
| `run sombra now` | `run my-app now` |
| `` `sombra` ``   | `` `my-app` ``   |
| `sombra-cli`     | unchanged        |
| `sombra.yaml`    | unchanged        |
| `.sombra`        | unchanged        |
| `sombrahq`       | unchanged        |

Because `.` and `-` count as token characters, a trailing `sombra.` at the end of a
sentence is also left untouched.

### `ci:` and `word-ci:`

`ci:` is a case-insensitive literal replacement and is *not* token aware. Combine it
with `word:` using `word-ci:` to get both behaviors:

```yaml
"word-ci:sombra": "my-app"
```

This replaces `Sombra`, `SOMBRA` and `sombra` when they stand alone, while leaving
`Sombra-cli` untouched.

### `json:`, `yaml:`, `ini:` structured edits

Structured directives update a single value addressed by a path instead of matching
text. The path is a dot separated list of object keys and array indexes:

```yaml
content:
    "json:scripts.test": "go test ./..."
    "yaml:jobs.build.runs-on": "ubuntu-latest"
    "ini:core.editor": "vim"
```

- `json:` replaces the value in place and leaves the rest of the file byte for byte
  identical.
- `yaml:` preserves comments and key order but re-encodes the document, so quoting and
  spacing may be normalized.
- `ini:` rewrites the matching `key = value` (or `key: value`) line; an inline comment
  on that line is not preserved.

A path that cannot be resolved is an error. JSON values are interpreted as JSON when
possible (`true`, `42`, `[...]`), otherwise as strings.

### `file:` replace the whole file

`file:` ignores the existing content and sets the file to the mapping value, expanded
with variables like any other value. A multi-line YAML block scalar is the usual way to
write it:

```yaml
-   pattern: "/.gitignore"
    content:
        "file:": |
            /.cache/
            /build/
```

Everything after `file:` is ignored, so `file:` and `file:anything` behave the same.

### `replace`: replace file content with a `.sombra` file

`replace` is a pattern field that discards the matched file's content and uses the
content of a file from the template's `.sombra` directory instead. The value is a path
relative to `.sombra` and may not escape that directory. It is the recommended way to
keep a real file in the template for reference while shipping only a minimal boilerplate
to new projects:

```yaml
-   pattern: "/README.md"
    replace: snippets/README.md
```

The example above renders the content of `.sombra/snippets/README.md` into the target
`README.md`, ignoring the repository's own `README.md`.

The referenced file is rendered with Go templates and Sprig functions using the project
variables before it is used, so `{{ .project_name }}` is expanded. When the pattern is
not `verbatim`, any `content` mappings on the same pattern are then applied on top of
the replacement.

`replace` is best used with a specific `pattern` so the reference file is skipped and is
not exposed as source code. When several matching patterns define `replace`, a
non-abstract pattern overrides an `abstract` one. `replace` applies to the `copy` update
path only; the `--method diff` path applies string mappings alone.

### `when`: conditional patterns

`when` enables or disables a whole pattern, including all of its mappings. It is a Go
template expression that is evaluated with the project variables before the definition
is parsed, so it must render to `true` or `false`:

```yaml
-   pattern: "/.github/**"
    when: '{{ eq .include_ci "true" }}'
    content:
        "word:sombra": "{{ .project_name }}"
```

When the expression is false the pattern is skipped entirely: it does not contribute
mappings and cannot mark a file as included. This is useful for optional components
(CI, docs, licenses) that would otherwise need a large `except` denylist.

`when` is optional; omitting it keeps the pattern enabled as before. The `sombra init`
command reads the definition before it is rendered, so patterns whose `when` is still an
unevaluated expression are treated as enabled at that stage.

### `copy_only`: copy-method-only patterns

`copy_only: true` ties a pattern to the `copy` update method. Files matched only by
`copy_only` patterns are created and updated by `sombra local update --method copy`
(the default) and are ignored by `--method diff`. On the copy path every replacement
behaves as usual: `path`, `name`, `content`, `block_directives` and `replace` all apply.

Use it for files that should be seeded once and then left untouched by incremental
updates:

```yaml
-   pattern: "/cmd/**"
    copy_only: true
```

### Binary files

Binary files are detected automatically by scanning the first bytes for a NUL byte. They
are always copied byte-for-byte, even without `verbatim`, so an active `re:` mapping
cannot corrupt a PNG or archive. Binary detection applies to the `copy` update path;
`--method diff` only ever sees text hunks.

## In-file Block Directives

Copied files are not rendered with Go templates, so content to omit is expressed
with line markers. They are comment agnostic: the marker can appear inside any line
comment. Block directives are **opt-in**: set `block_directives: true` on the pattern
that owns the file, otherwise `sombra:` text is treated as ordinary content and copied
untouched.

```yaml
-   pattern: "config/**/*.yaml"
    block_directives: true
```

```text
# sombra:skip
jobs:
  build: ...
# sombra:end
```

The `sombra:skip` / `sombra:end` markers and every line between them are removed from
the output.

Blocks can be nested. An unbalanced `sombra:skip`/`sombra:end` pair fails the file and
reports the line number; the copy flow prefixes that error with the file name. Block,
structured and `file:` directives apply to whole files only: the `--method diff` update
path applies string mappings alone.

---

For a step-by-step guide, continue to [Start a Template](start-a-template.md).
