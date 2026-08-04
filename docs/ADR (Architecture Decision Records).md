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

The first fifteen were written retroactively in August 2026, from the commit
history. Each is dated to the commit that introduced the decision, not to the day
it was written down.

| ADR | Title | Date | Status |
| --- | --- | --- | --- |
| 0001 | [A single self-hosted binary](<ADR/0001 - A single self-hosted binary.md>) | 2026-07-13 | Accepted |
| 0002 | [SQLite as the only datastore](<ADR/0002 - SQLite as the only datastore.md>) | 2026-07-13 | Accepted |
| 0003 | [A pure Go build with CGO disabled](<ADR/0003 - A pure Go build with CGO disabled.md>) | 2026-07-13 | Accepted |
| 0004 | [sqlc instead of an ORM](<ADR/0004 - sqlc instead of an ORM.md>) | 2026-07-13 | Accepted |
| 0005 | [The official Docker SDK under moby](<ADR/0005 - The official Docker SDK under moby.md>) | 2026-07-13 | Accepted |
| 0006 | [Docker as the source of truth](<ADR/0006 - Docker as the source of truth.md>) | 2026-07-13 | Accepted |
| 0007 | [Cookie sessions instead of tokens](<ADR/0007 - Cookie sessions instead of tokens.md>) | 2026-07-13 | Accepted |
| 0008 | [A hand-built interface on headless primitives](<ADR/0008 - A hand-built interface on headless primitives.md>) | 2026-07-13 | Accepted |
| 0009 | [A poller alongside Docker event watchers](<ADR/0009 - A poller alongside Docker event watchers.md>) | 2026-07-14 | Accepted |
| 0010 | [Server-Sent Events for streams, WebSocket only for exec](<ADR/0010 - Server-Sent Events for streams, WebSocket only for exec.md>) | 2026-07-14 | Accepted |
| 0011 | [Deploying managed stacks through the Compose CLI](<ADR/0011 - Deploying managed stacks through the Compose CLI.md>) | 2026-07-14 | Accepted |
| 0012 | [Discovering stacks from compose labels](<ADR/0012 - Discovering stacks from compose labels.md>) | 2026-07-14 | Accepted |
| 0013 | [Encrypted credential storage with a local key](<ADR/0013 - Encrypted credential storage with a local key.md>) | 2026-07-14 | Accepted |
| 0014 | [A setup token to claim the instance](<ADR/0014 - A setup token to claim the instance.md>) | 2026-07-15 | Accepted |
| 0015 | [A single SQLite connection](<ADR/0015 - A single SQLite connection.md>) | 2026-07-15 | Accepted |
