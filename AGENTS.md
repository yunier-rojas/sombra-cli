# AGENTS.md

Guidance for agentic coding tools working in the **sombra-cli** repository.

## Project Overview

Sombra is a Go CLI that turns Git repositories into reusable, version-controlled
project templates. Users define rules in YAML using Go templates, and the CLI
scaffolds and updates projects from those templates.

- Module: `github.com/yunier-rojas/sombra-cli`
- Go version: 1.26 (see `go.mod`)
- Entrypoint: `cmd/sombra/main.go`

## Commands

All commands are driven by the top-level `Makefile`, which includes `mk/*.mk`.

| Command                                              | Purpose                                                            |
|------------------------------------------------------|--------------------------------------------------------------------|
| `make build WHAT=sombra`                             | Build binary into `build/`                                         |
| `make test`                                          | Run all tests with coverage (`go test --cover -parallel=1 -v ...`) |
| `make qa`                                            | Format `internal/` and `cmd/` with `go fmt`                        |
| `make imports`                                       | Enforce import-layer rules (see Architecture)                      |
| `make build-docs`                                    | Build MkDocs site (requires Python/pip)                            |
| `go test ./internal/core/usecases/lib_cvs_test.go`   | Run a single test file                                             |
| `go test -run TestName ./internal/core/usecases/...` | Run a single test by name                                          |

### Required before finishing

Run these and ensure they pass:

```bash
make qa        # formatting
make test      # tests
make imports   # import rules
```

CI (`.github/workflows/ci.yaml`) runs `make test` and `make imports`, plus
`make build-docs`.

## Architecture

Clean/hexagonal layering. Dependencies point inward.

```
cmd/sombra              CLI parsing (go-arg). Only allowed to import
                        internal/runtime and the logger framework.

internal/core/entities  Pure domain types + helpers. No external deps
                        except fmt/strings.
internal/core/usecases  Business logic (interactors) + ports. No framework
                        imports; only entities.
internal/frameworks/*   Concrete adapters: analysers, cvs/git, files, logger,
                        sombra, templates, vars, versions.
internal/runtime        Wiring/composition root. Constructs interactors with
                        framework implementations and exposes use cases.
```

### Import rules are enforced

`qa/.import.yaml` defines per-folder allowlists and is validated by
`make imports`. Do not introduce imports outside the allowed list for a folder.
Notably:

- `core/usecases` may **not** import frameworks; if a use case needs I/O,
  define a port (interface) and implement it in a framework.
- `core/entities` stays dependency-free (only `fmt`, `strings`).
- `internal/runtime` is the only place that wires core + frameworks together.

### Conventions

- Package layering: when adding a feature, add the interactor in
  `internal/core/usecases`, the adapter in the relevant `internal/frameworks/*`
  package, and wire it in `internal/runtime`.
- Follow existing naming: use cases are `...Interactor`, services in frameworks
  are `...Service`, engines `...Engine`.
- Tests live next to the code as `*_test.go` in `internal/core/usecases`.
- Error handling uses `github.com/cockroachdb/errors`.
- Logging uses `github.com/rs/zerolog` via `internal/frameworks/logger`; do not
  log directly in core packages.
- YAML parsing uses `gopkg.in/yaml.v3`; Go templating uses
  `github.com/Masterminds/sprig/v3`.
- Do **not** add comments unless the surrounding code already documents that way
  or the user asks.
- Ask before adding any external dependency.

### Formatting

- Go files and Makefiles use tabs; YAML/JSON use 2 spaces (`.editorconfig`).
- Use `make qa` (`go fmt`) before committing; do not hand-format.

## Docs

- User-facing docs live in `docs/` and are built with MkDocs (`mkdocs.yml`).
- Update them when changing CLI commands or the `sombra.yaml` format.
- Encouraging, accessible, clear, and empathetic.
- Use short sentences, active voice, concise language.
- Forbidden to use fluff, clichés, and corporate jargon.
- Keeps docs up to date with the code.
- Document features and behaviors, use facts, examples but not opinions.
- State limitations and caveats.

## Errors
- Wrap with context and %w: `fmt.Errorf("load user %s: %w", id, err)`.
- Compare with errors.Is / errors.As, never ==.
- Handle once: add context and return. Do not log and return the same error.
- No naked returns. No panic outside main() and package init.

## Style the linter cannot enforce
- Interfaces are declared by the CONSUMER, in the consumer's package, and are small. One or two methods. Return concrete types.
- Use the current stdlib: os.ReadFile not ioutil, any not interface{}, slices/maps packages, log/slog not logrus, math/rand/v2.
