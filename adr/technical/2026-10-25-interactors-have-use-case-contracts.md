# Each interactor has a use case contract

## Problem

Callers construct concrete interactor structs, so the operations a use case offers are
only visible in its implementation. There is no small contract to fake or depend on
apart from the code that runs it.

## Decisions

- Every `XInteractor` struct in `internal/core/usecases` has a case interface in the
  same file, declared before the struct.
- The interface declares the interactor's exported methods with their Go
  documentation.
- The interactor implements the interface and asserts it:
  `var _ XCase = (*XInteractor)(nil)`.
- Private methods, fields and helper types stay in the struct. The interface holds no
  state.
- Ports are constructor parameters, not members of the use case interface.
- `make lint-usecases` fails when an interactor has no case interface, the interface is
  empty or misplaced, the assertion is missing, or an exported method is not declared.

## Consequences

Callers and tests depend on the contract without constructing the implementation.
Adding an operation means updating the interface and the implementation. The compiler
checks the assertion; the custom check keeps the exported methods in sync. The cost is
one interface per interactor and a small AST check.
