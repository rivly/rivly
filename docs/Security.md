This page describes the security model of Rivly and the rules that protect it.

---

## Threat model

Rivly holds a socket to a Docker daemon. Anyone who can create a container with a
host bind mount can read and write the host filesystem as root.

There is therefore no meaningful boundary between "access to Rivly" and "root on
the host". Every control below exists to protect that single fact.

The assumed deployment is a self-hosted instance behind TLS, reachable only by
its operator. Rivly is not designed to be exposed to the public internet.

---

## Authentication

The first account is created through a one-time setup flow, guarded by a setup
token generated at startup and printed to the logs. Without that token the
instance cannot be claimed, so an instance discovered before its owner reaches it
cannot be stolen.

Passwords are hashed with argon2id. The parameters are set in one place and are
never chosen per call site.

Authentication failures are generic and constant-time. A login against an unknown
email still performs a hash comparison against a decoy, so response timing does
not reveal whether an account exists.

Sessions are server side, stored in SQLite. The cookie is `HttpOnly`,
`SameSite=Lax`, and is upgraded to `Secure` when the request arrives over TLS or
through a proxy that declares it.

Changing a password destroys every other session for that account and renews the
current one.

---

## Authorization

Every account is an administrator today. The `role` column exists and setup
writes `admin`, but no endpoint checks it: authorization is currently "has a
session".

This is honest for a single-operator tool and unacceptable for a multi-user one.
Role enforcement is tracked in [Roadmap](Roadmap.md).

---

## Rate limiting

Setup, login and password change are limited per client IP. Nothing else is
limited, because every other endpoint requires a session.

---

## Secrets

Registry passwords and Git tokens are encrypted with AES-256-GCM before storage.
The key is generated on first run and kept in the data directory with owner-only
permissions.

Secrets are never returned by the API. Updating a credential without supplying a
new secret keeps the stored one.

Never log a secret. That includes a repository URL, which can carry a token in
its userinfo.

---

## Transport

Every response carries `X-Content-Type-Options: nosniff`,
`X-Frame-Options: DENY`, `Referrer-Policy: no-referrer` and a Content-Security
Policy that allows scripts, styles, fonts and connections from the origin only.

Scripts are never allowed inline and never evaluated. Styles do allow
`'unsafe-inline'`, because the headless widget and terminal libraries position
elements through inline styles at runtime.

Unsafe methods are protected against cross-origin requests. Additional origins
must be declared explicitly through `RIVLY_TRUSTED_ORIGINS`.

Request bodies are bounded, and unknown JSON fields are rejected rather than
ignored.

---

## Rules

- Never roll your own hashing, session handling or cookie logic. `internal/auth`
  already exists.
- Never weaken authentication, authorization, cryptography or input validation
  without an ADR.
- Never return an internal error to a client. Log the cause once, return a
  generic message.
- Validate every name that reaches the filesystem or a subprocess against an
  explicit pattern, never by escaping.
- Prefer secure defaults, including when they are less convenient.
