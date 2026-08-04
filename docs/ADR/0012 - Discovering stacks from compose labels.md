| Field | Value |
| --- | --- |
| **Identifier** | ADR-0012 |
| **Date** | 2026-07-14 |
| **Status** | Accepted |

---

## Context

Docker has no concept of a stack. A compose project exists only as a label that
Compose writes onto the containers, volumes and networks it creates.

Anyone installing Rivly already has compose projects running, deployed from a
terminal. A dashboard that only shows what it deployed itself is useless on day
one, and pushes users to redeploy everything through it just to see it.

---

## Decision

Rivly reconstructs stacks by grouping Docker objects on their compose labels:
`com.docker.compose.project` for the grouping, `com.docker.compose.service` to
count distinct services, `com.docker.compose.project.working_dir` to show where
an external stack came from.

Two kinds of stack result, and the distinction is surfaced in the interface.

**External** stacks are discovered and controllable, but their compose file is
unknown to Rivly. Lifecycle actions apply to their containers directly.

**Managed** stacks have a row in the database with their compose file, their
environment variables, their author and their history.

Listing merges both views. A discovered project that also has a row is reported
as managed.

---

## Consequences

Rivly is useful immediately on an existing host, which is the point.

A stack adopted from outside cannot be edited, because Rivly never saw its
compose file. It can only be started, stopped, restarted or removed.

Removing a managed stack tears it down through Compose and deletes its row.
Removing an external stack acts on its containers, and leaves whatever created it
untouched.

Anything that writes compose labels by hand will appear as a stack. This is
acceptable: the label is the only definition of a project that exists.

Swarm services carry different labels and are not covered by this mechanism.
