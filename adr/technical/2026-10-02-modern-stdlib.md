# Modern Standard Library

## Problem

Legacy APIs (`ioutil`, `interface{}`, `math/rand`) linger in code even though
the standard library has replacements.

## Decisions

- Use `any` instead of `interface{}`.
- Use `os.ReadFile` instead of `ioutil`.
- Use the `slices` and `maps` packages.
- Use `log/slog` instead of `logrus`.
- Use `math/rand/v2`.

Most of these are import level and already covered by `qa/import-core.yaml` and
`qa/import-rest.yaml`. The empty interface check is enforced by ast-grep.

## Consequences

- Code uses current standard library APIs.
