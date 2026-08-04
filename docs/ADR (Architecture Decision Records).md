This page describes how architectural decisions are recorded in Rivly.

Structural decisions are documented so that their context, their justification
and their consequences survive the people who made them. A note tells you how
Rivly works today. An ADR tells you why it became that way.

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

## Format

Rivly uses the Nygard format: Context, Decision, Consequences, preceded by a
metadata table. See [0000 - Template](<ADR/0000 - Template.md>).

That format is deliberate. A 2026 empirical comparison of ADR templates found
Nygard scored higher than MADR on comprehension, usability and ease of adoption.
Heavier templates get written once and abandoned.

Keep an ADR short. One that nobody rereads has failed at its only job.

---

## Rules

An ADR is immutable once accepted. It is a record, not a living document.

To change a decision, write a new ADR that supersedes the old one, and set the
old one's status to Superseded with a link to its replacement. Never rewrite
history.

Numbers are sequential and never reused. The file name is
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

## Records

| ADR | Title | Status |
| --- | --- | --- |
| | | |
