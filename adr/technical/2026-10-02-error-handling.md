# Error Handling

## Problem

Error handling drifted: some errors were compared with `==`, some lost their
cause, and panics could appear outside startup. Reviews could not catch this
consistently.

## Decisions

- Wrap errors with context using `%w`.
- Compare errors with `errors.Is`/`errors.As`, never `==`.
- Do not use naked returns.
- Do not call `panic` outside `main` or package `init`.

The comparison and wrapping checks are best effort: ast-grep has no type
information, so it matches identifiers that look like errors.

## Consequences

- Errors keep a cause chain for callers.
