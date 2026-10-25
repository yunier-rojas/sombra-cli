# Code Shape

## Problem

Wide interfaces and inconsistent names made the layering hard to follow: ports
grew without limit, adapters were named after implementation details, and use
cases were not always identifiable.

## Decisions

- Interfaces declare at most three methods directly and are declared by the consumer.
  Composites may embed smaller interfaces.
- Use cases are `...Interactor`.
- Every exported framework adapter is `...Service`; engines are `...Engine`.

## Consequences

- The files adapter implements several small ports instead of one broad
  interface.
- Adapter types are named `...Service`.
