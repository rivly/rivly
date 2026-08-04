| Field | Value |
| --- | --- |
| **Identifier** | ADR-0014 |
| **Date** | 2026-07-15 |
| **Status** | Accepted |

---

## Context

A fresh Rivly instance has no accounts, so the first-run flow has to let someone
create one without being authenticated.

Left open, that flow is a takeover primitive. Between the moment the container
starts and the moment its owner opens the browser, anyone who reaches the port
can claim it, and claiming it means a Docker socket on the host. On a home
network with a forwarded port, that window is long enough to lose.

Binding setup to localhost was considered and rejected: the common install is a
container behind a reverse proxy, where every request already looks remote.

---

## Decision

Claiming an instance requires a setup token.

At startup, when no account exists, Rivly generates 32 random bytes and logs the
token. The operator copies it from the container logs into the setup screen.

`RIVLY_SETUP_TOKEN` pins the token instead, which is what makes automated
provisioning possible.

The token is compared in constant time. An empty configured token never matches,
so a misconfiguration fails closed rather than leaving setup open.

Once an account exists the endpoint returns a conflict before the token is even
examined, and the token is dropped from memory.

---

## Consequences

An unclaimed instance cannot be stolen by whoever finds it first. Possession of
the logs is required, and reaching the logs already implies host access.

The first-run experience costs one extra step, which is a deliberate trade
against a total compromise.

A user who cannot read the container logs cannot complete setup. This is the
main friction, and it is why the token is logged at info level with an explicit
message rather than buried.

Restarting an unclaimed instance generates a new token, invalidating the previous
one.
