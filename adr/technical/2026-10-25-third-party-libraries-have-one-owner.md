# Third-party libraries have one owner
## Problem

A third-party library shared by several framework units makes every adapter know its
types, versions, lifecycle and failure modes. `yaml.v3` and `doublestar` already cross
framework boundaries.

## Decisions

- A framework unit is the first package below `internal/frameworks`; its subpackages
  belong to that unit.
- A third-party library has exactly one owning package. `cmd` may own CLI libraries,
  and `runtime` may own composition libraries.
- If another unit needs the capability, define a port in `usecases`, implement it in
  the owner, and inject it from `runtime`.
- If two implementations genuinely need the same library, create a new framework unit
  for that capability.
- The import allowlist controls imports, not ownership. A listed external import still
  needs one owner.
- Existing shared uses live in `.sombra/third-party-owners.txt`. New shared uses fail.
  The list may only shrink.
- `make lint-third-party` derives owners from `go list` and enforces the baseline.

## Consequences

Upgrades, vulnerability fixes and runtime changes have one place to land. A port can be
faked in a test. The cost is wiring, and genuine sharing needs a new framework unit. The
migration list stays visible and can only get smaller.
