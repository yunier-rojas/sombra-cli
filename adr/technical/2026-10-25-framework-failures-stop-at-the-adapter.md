# Framework Failures Stop at the Adapter
## Problem

The update use case builds absolute paths with `path/filepath`, and adapters return
raw `os`, `os/exec`, YAML and git failures. The core can only tell a missing template
from a permission failure by parsing adapter text.

## Decisions

- A domain failure is an `entities.Failure` with a kind: `not-found`, `permission`,
  `invalid`, `unavailable` or `storage`. Adapters translate platform failures before
  they cross a port.
- `entities` and `usecases` do not import `os`, `io/fs`, `path/filepath` or
  `os/exec`. A path crosses a port as `entities.File` or `string`.
- Ports stay small contracts: interfaces and neutral values, with no filesystem walk,
  process execution or OS path type.
- Interactors rely on domain failures. A framework failure does not decide whether an
  update or init fails.
- `FileStorePort` takes a root and a slash-relative path. `Remove` cleans empty
  directories inside the adapter.
- The import allowlist and ast-grep rules enforce the boundary. When those cannot
  express a rule, a small script under `qa/` runs from `make qa` and CI and fails closed.

## Consequences

A path is data at the boundary and a filesystem path only inside an adapter. The core
can distinguish not found, permission, invalid input, unavailable source and storage
failure without inspecting an `os` error. A fake store can use string paths. The cost
is a conversion at each adapter edge and a little duplicated platform knowledge.
