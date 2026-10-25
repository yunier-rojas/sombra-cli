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
| `make qa`                                            | Format, lint, imports, ast-grep and ADR checks                     |
| `make imports`                                       | Enforce import-layer rules (see Architecture)                      |
| `make lint-sg`                                       | Run ast-grep rules from `qa/sg/rules`                              |
| `make lint-usecases`                                 | Check interactor use case contracts                                |
| `make lint-third-party`                              | Check third-party library ownership                                |
| `make lint-adr`                                      | Check ADR names, length and required sections                      |
| `make build-docs`                                    | Build Hugo site (requires Hugo extended + Go)                      |
| `make serve-docs`                                    | Serve docs locally at `http://localhost:1313/`                     |
| `go test ./internal/core/usecases/lib_cvs_test.go`   | Run a single test file                                             |
| `go test -run TestName ./internal/core/usecases/...` | Run a single test by name                                          |

### Required before finishing

Run these and ensure they pass:

```bash
make qa        # formatting, lint, imports, ast-grep and ADRs
make test      # tests
```

CI (`.github/workflows/ci.yaml`) runs `make test`, `make imports`, `make lint-sg`,
`make lint-usecases`, `make lint-third-party`, `make lint-adr`, plus
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

`qa/import-core.yaml` and `qa/import-rest.yaml` define per-folder allowlists and
are validated by `make imports`. Do not introduce imports outside the allowed
list for a folder. Notably:

- `core/usecases` may **not** import frameworks; if a use case needs I/O,
  define a port (interface) and implement it in a framework.
- `core/entities` and `core/usecases` may **not** import `os`, `io/fs`,
  `path/filepath` or `os/exec`. Paths cross ports as `entities.File` or `string`.
- Adapters translate platform failures into `entities.Failure` values before they
  cross a port. The core relies on the failure kind, not on an `os` error.
- `core/entities` stays dependency-free (only `fmt`, `strings`).
- `internal/runtime` is the only place that wires core + frameworks together.

### Decisions are recorded

- Every decision goes in `adr/<type>/` as `yyyy-mm-dd-short-title.md`, where type is
  `product`, `process` or `technical`.
- Each ADR has a title, Problem, Decisions and Consequences, stays under 30
  lines and 500 words, and does not repeat the date in the title.
- Every ast-grep rule names the ADR that created it under `metadata.adr`.
- Run `make lint-adr` and `make lint-sg` after changing an ADR or a rule.

### Conventions

- Package layering: when adding a feature, add the interactor in
  `internal/core/usecases`, the adapter in the relevant `internal/frameworks/*`
  package, and wire it in `internal/runtime`.
- Follow existing naming: use cases are `...Interactor`, services in frameworks
  are `...Service`, engines `...Engine`.
- Ports take roots and slash-relative paths. Adapters resolve them into platform
  paths and translate platform failures.
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

## Working on guidance

- Write guidance for future work, not a history of the change. AGENTS.md holds
  standing rules that apply to every future change; rationale and past context belong
  in an ADR.
- When a rule can be enforced, add the enforcement. Prefer the compiler, import
  allowlists and ast-grep. When those cannot express it, a small script under `qa/`
  wired into `make qa` and CI is acceptable, as long as it fails closed.

## Docs

- User-facing docs live in `docs/` and are built with Hugo + Hextra
  (`docs/hugo.toml`, content in `docs/content/`, theme pinned in `docs/go.mod`).
- Update them when changing CLI commands or the `sombra.yaml` format.
- Encouraging, accessible, clear, and empathetic.
- Use short sentences, active voice, concise language.
- Forbidden to use fluff, clichés, and corporate jargon.
- Keeps docs up to date with the code.
- Document features and behaviors, use facts, examples but not opinions.
- State limitations and caveats.
- Heavily reduce the usage of emojis.
- Forbid the use of — in the docs
- Never write the literal `sombra:skip` or `sombra:end` tokens in prose of
  template-copied pages (`_index.md`, `about.md`, `contact.md`); the copy engine
  treats any line containing them as a block directive.

## Errors
- Wrap with context and %w: `fmt.Errorf("load user %s: %w", id, err)`.
- Compare with errors.Is / errors.As, never ==.
- Handle once: add context and return. Do not log and return the same error.
- No naked returns. No panic outside main() and package init.

## Style the linter cannot enforce
- Interfaces are declared by the CONSUMER, in the consumer's package, and are small. One or two methods. Return concrete types.
- Use the current stdlib: os.ReadFile not ioutil, any not interface{}, slices/maps packages, log/slog not logrus, math/rand/v2.
