# ADR-0007 - Cookie sessions instead of tokens

| Field | Value |
| --- | --- |
| **Identifier** | ADR-0007 |
| **Date** | 2026-07-13 |
| **Status** | Accepted |

---

## Context

Rivly has exactly one client: its own dashboard, running in a browser. There is
no mobile app, no third-party integration, and no service-to-service call.

A JWT stored in the browser has to live somewhere JavaScript can reach, which
makes it readable by any script that gets injected. It also cannot be revoked
before it expires, which matters for a tool that holds a Docker socket.

---

## Decision

Authentication is a server-side session, stored in SQLite through scs, referenced
by a cookie that is `HttpOnly`, `SameSite=Lax`, scoped to `/`, and marked
`Secure` when the request arrives over TLS or through a proxy that declares it.

Passwords are hashed with argon2id, with parameters defined in one place.

Authentication failures are generic and constant-time. A login against an unknown
email still performs a hash comparison against a decoy, so timing does not reveal
whether an account exists.

Writes are protected against cross-origin requests by the standard library's
cross-origin protection rather than by a CSRF token, with extra origins declared
through `RIVLY_TRUSTED_ORIGINS`.

Changing a password destroys every other session for that account.

---

## Consequences

A session can be revoked immediately, which is what makes "sign out everywhere"
and the password-change behaviour possible.

The token is unreachable from JavaScript, so an injected script cannot steal it.

The dashboard must be same-origin with the API, which is why development proxies
`/api` through Vite instead of calling `:8080` directly.

Streaming endpoints run outside the session middleware and have to load the
session from the cookie themselves.

Exposing a machine-usable API later means adding a second mechanism, not reusing
this one.
