This page describes how architectural decisions are recorded in Rivly.

Structural decisions are documented so that their context, their justification
and their consequences survive the people who made them. A note tells you how
Rivly works today. An ADR tells you why it works that way, and what was rejected.

---

## When to write an ADR

Write one when a decision durably shapes the platform:

- adopting, replacing or dropping a technology;
- changing the architecture or a package boundary;
- changing a security or cryptography principle;
- changing the API contract or the database schema;
- accepting a trade-off that a future reader would otherwise undo by mistake.

Do not write one for a bug fix, a refactor with no external effect, or a choice
that is trivially reversible.

---

## Rules

An ADR is immutable once accepted. It is a record, not a living document.

To change a decision, write a new ADR that supersedes the old one, and update the
status of the old one to point at its replacement. Never rewrite history.

The number is sequential and never reused. The file name is
`NNNN - Short title.md`.

---

## Status values

| Status | Meaning |
| --- | --- |
| Proposed | Written, not yet agreed |
| Accepted | In force |
| Superseded | Replaced by a later ADR, which must be linked |
| Deprecated | No longer applies, with no replacement |

---

## Structure

Every ADR uses [0000 - Template](<ADR/0000 - Template.md>): a metadata table,
then Context, Decision, Rationale, Consequences and Alternatives considered.

Keep it short. An ADR that nobody rereads has failed at its only job.

---

## Records

| ADR | Title | Status |
| --- | --- | --- |
| | | |
