This page describes the design principles of the Rivly HTTP API.

The API exists to serve the Rivly dashboard. It is not a public integration
surface yet, and no stability guarantee is offered outside a release.

---

## Principles

- JSON in, JSON out. No form encoding, no query parameters carrying state.
- Resources are nested under the environment they belong to, because every Docker
  object is meaningless without a daemon.
- A handler parses, delegates and encodes. Logic lives in a package, not in a
  handler.
- An error response carries a message a user can read, never an internal detail.

---

## Versioning

Business endpoints live under `/api/v1/`.

Operational endpoints such as `/api/health` are unversioned, because a probe
should not have to follow an API version.

A breaking change means a new prefix, not a mutation of the existing one.

---

## Authentication

Authentication is a session cookie, `HttpOnly`, `SameSite=Lax`, `Path=/`, marked
`Secure` automatically when the request arrives over TLS or through a proxy that
says so.

There is no bearer token and no API key. A browser is the only intended client,
which is what makes a cookie the right choice: it cannot be read by JavaScript.

Instance claiming is a one-time flow. Until an account exists, `GET /setup`
reports that setup is needed and `POST /setup` creates the first administrator,
guarded by a setup token printed in the logs at startup.

Streaming endpoints authenticate by loading the session from the cookie directly,
because they run outside the session middleware.

---

## Cross-origin protection

The server applies Go's cross-origin protection to every unsafe method, so a
browser on another origin cannot write to the API. Additional origins are
declared through `RIVLY_TRUSTED_ORIGINS`.

In development the dashboard proxies `/api` through Vite so requests stay
same-origin. Calling the backend directly from `:5173` will be rejected.

---

## Resources

| Prefix | Covers |
| --- | --- |
| `/setup`, `/login`, `/logout`, `/me` | Account and session |
| `/environments` | Docker endpoints and their system info |
| `/environments/{id}/containers` | List, create, inspect, act |
| `/environments/{id}/images` | List, inspect, pull, prune, act |
| `/environments/{id}/volumes` | List, create, inspect, act |
| `/environments/{id}/networks` | List, create, inspect, act |
| `/environments/{id}/stacks` | List, deploy, inspect, act |
| `/registries`, `/git-credentials` | Credential stores |

---

## Conventions

Reads are `GET` and return an array or an object. A list is always an array,
never an object wrapping one, and it is empty rather than null.

Writes are `POST`. Updates on credential stores are `PUT`, deletes are `DELETE`
returning `204`.

Bulk operations on Docker objects are a single `POST` to an `actions` endpoint,
carrying an action and a list of ids. They return one result per id, each with
its own success flag, because a partial failure is the normal case.

Timestamps are unix seconds. Sizes are bytes. Neither is formatted server side.

---

## Errors

An error response is `{"error": "message"}` with a meaningful status.

| Status | Meaning |
| --- | --- |
| `400` | The request is malformed or fails validation |
| `401` | No valid session |
| `403` | The setup token is wrong |
| `404` | The environment or resource does not exist |
| `409` | The instance is already claimed, or the name is taken |
| `413` | The body exceeds the limit |
| `422` | Compose or Git rejected the input, with their output as the message |
| `429` | Rate limit on an authentication endpoint |
| `502` | The daemon is unreachable or refused the operation |

Authentication failures are deliberately generic and constant-time, so they never
reveal whether an account exists.

---

## Streaming

Logs, stats and image pulls are Server-Sent Events. Each stream opens with a
comment, sends a periodic heartbeat, and closes with an `end` event.

The container terminal is a WebSocket. Binary frames carry terminal bytes, text
frames carry control messages such as a resize.

The event feed is a single SSE stream on `/api/v1/events` carrying typed events,
currently `environment.updated`.
