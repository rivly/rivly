# ADR-0001 - A single self-hosted binary

| Field | Value |
| --- | --- |
| **Identifier** | ADR-0001 |
| **Date** | 2026-07-13 |
| **Status** | Accepted |

---

## Context

Rivly targets people who run containers on their own hardware. That audience
abandons a tool at the install step, not at the feature comparison.

The existing options either require a compose file with several services, or hand
management to a hosted control plane. Both are a reason not to try the tool.

---

## Decision

Rivly ships as one binary and one container image, with no companion service.

Everything the product needs at runtime is either inside the binary or inside a
single mounted volume. A user installs it by running one container with the
Docker socket mounted.

This is a constraint, not a preference. Any proposal that adds a service to the
runtime is rejected by default, and must be justified against this ADR.

---

## Consequences

The datastore has to be embedded, which leads to [ADR-0002](<0002 - SQLite as the only datastore.md>).

The binary has to be static, which leads to [ADR-0003](<0003 - A pure Go build with CGO disabled.md>) and rules out
whole categories of library.

There is no queue and no cache, so background work runs as goroutines in the same
process and dies with it. Anything long-running must be resumable.

Horizontal scaling is foreclosed. A second Rivly instance pointed at the same
volume is not supported and will corrupt state.

The frontend must be embedded rather than served separately, which is not done
yet and is tracked in [Packaging](../Packaging.md).
