| Field | Value |
| --- | --- |
| **Identifier** | ADR-0004 |
| **Date** | 2026-07-13 |
| **Status** | Accepted |

---

## Context

Rivly's data access is a small set of well-known queries against a schema that
changes rarely. There is no dynamic query building, no user-defined filtering,
and no polymorphic loading.

An ORM solves problems Rivly does not have, at the cost of a runtime layer
between the code and the SQL, and of a query language that has to be learned
alongside SQL rather than instead of it.

Hand-written `database/sql` code solves it too, but at the cost of scanning
boilerplate that is tedious to review and easy to get wrong when a column is
added.

---

## Decision

SQL is written by hand in `internal/database/queries/*.sql`. `sqlc generate`
produces the Go code in `internal/database/db/`.

Generated code is never edited. Changing a query means editing the SQL and
regenerating.

---

## Consequences

The SQL in the repository is the SQL that runs, which makes review and
explanation straightforward.

A schema change that breaks a query fails at generation time rather than at
runtime.

`sqlc` becomes a required development tool, and its output must be committed so
the build does not depend on it.

Dynamic queries are awkward by design. If one is ever genuinely needed, it is
written by hand next to the generated code, not by loosening this rule.
