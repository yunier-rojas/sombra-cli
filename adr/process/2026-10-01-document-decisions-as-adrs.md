# Document Decisions as ADRs

## Problem

Decisions are made every day. Without a record, the reason behind a choice is
lost, and new contributors repeat old debates. We need a cheap, durable way to
capture decisions and their reasoning. ADRs are a database of active decisions.
If a decision is no longer relevant, rewrite it or delete it.

## Decisions

Every decision, ours and the agent's, is a Markdown file in `adr/<type>/`.

- Type is `product`, `process` or `technical`.
- Named `[yyyy-mm-dd]-short-title.md`.
- Do not add the date in the ADR title
- Body: Title, Problem, Decisions, Consequences.
- Under 30 lines and 500 words. Prefer the simplest option that works.
- ast-grep rules and enforcement scripts must specify the ADR they implement.
- Never write ast-grep rules inside the ADR.
- Do not write the rule names in the ADRs.

## Consequences

- Low writing cost, so it actually gets done.
- `adr/` is the first place to look before proposing a change.

