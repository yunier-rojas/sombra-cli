# Keep a core clean from technology

## Problem

Technology spreads into the core. Serialization, I/O, HTTP, randomness,
concurrency primitives, and third-party libraries creep into the use case and
entity packages because it is the shortest path to a feature. Each addition
couples the domain to an implementation choice, makes tests heavier, and
quietly widens the allowed dependency set until "core" means nothing.

## Decisions

- Core packages depend only on a small, fixed set of standard-library
  primitives and other core packages.
- All technology lives in framework adapters, reached through small ports that
  the core consumer declares.
- A feature that appears to need a new import in core is a signal to add a port
  and implement it in an adapter, not to widen the allowlist.
- The allowlist is architecture. Changing it is a decision, not an exception.

## Consequences

- The domain stays portable and testable without I/O or frameworks.
- Adapters own operational concerns and can be replaced without touching core.
- Some features take one more indirection than the shortest path.
