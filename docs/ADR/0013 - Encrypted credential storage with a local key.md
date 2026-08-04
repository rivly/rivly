| Field | Value |
| --- | --- |
| **Identifier** | ADR-0013 |
| **Date** | 2026-07-14 |
| **Status** | Accepted |

---

## Context

Pulling from a private registry and cloning a private repository both require a
credential that Rivly must replay later, without a user present. Hashing is
therefore not an option: the plaintext is needed.

[ADR-0001](<0001 - A single self-hosted binary.md>) rules out a secrets manager as a companion service, and asking a
self-hosting user to run Vault to store one registry password is absurd.

Storing them in plain text is also not acceptable. A leaked database file should
not hand over every registry and repository the instance can reach.

---

## Decision

Registry passwords and Git tokens are encrypted with AES-256-GCM before they
reach a column. The nonce is generated per encryption and prepended to the
ciphertext.

The key is 32 random bytes, generated on first run and written to `secret.key` in
the data directory with owner-only permissions.

Secrets are never returned by the API. Updating a credential without supplying a
new secret keeps the stored one, which is what lets a form show a credential
without ever having received it.

---

## Consequences

A leaked database file alone is not enough to recover credentials. The key file
is needed too.

The key file must be persisted and backed up alongside the database. Losing it
means losing every stored credential with no recovery path, which is stated in
[Configuration](../Configuration.md).

An attacker who reaches the filesystem gets both, so this protects against a
leaked backup or a stolen volume, not against host compromise. Given that Rivly
holds a Docker socket, host compromise is already total and encrypting harder
would not change that.

Key rotation is not implemented.

One credential path escapes this: a Git repository URL can carry a token in its
userinfo, and that URL is stored in a plain column. That is a defect, not a
decision.
