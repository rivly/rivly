| Field | Value |
| --- | --- |
| **Identifier** | ADR-0010 |
| **Date** | 2026-07-14 |
| **Status** | Accepted |

---

## Context

Rivly pushes four kinds of live data to the browser: state changes, container
logs, container statistics, and image pull progress. It also needs an interactive
terminal.

The first four are strictly server to client. The terminal is genuinely
bidirectional: keystrokes go up, output comes down, and resize events go up.

Using WebSocket for everything would work, and would mean writing reconnection,
heartbeat and backpressure logic for streams that never need to receive anything.

---

## Decision

One-way streams use Server-Sent Events. The browser's `EventSource` handles
reconnection natively.

- `/api/v1/events` carries typed state-change events for the whole app.
- Logs, statistics and image pulls open their own short-lived stream, scoped to
  the component that needs them.

Each stream opens with a comment, sends a periodic heartbeat, and closes with an
`end` event.

The container terminal uses a WebSocket, bridged onto a Docker exec session.
Binary frames carry terminal bytes, text frames carry control messages such as
resize.

Streaming endpoints authenticate by loading the session from the cookie directly,
because they run outside the session middleware.

---

## Consequences

Reconnection for the common case is free, handled by the browser.

SSE is plain HTTP, so it survives proxies that mishandle WebSocket upgrades. The
heartbeat and the `X-Accel-Buffering` header exist to survive proxies that buffer.

Any `http.ResponseWriter` wrapper added to the middleware chain must forward
`Flush` and `Hijack`, or every stream breaks at once. This has already happened
once.

Browsers cap concurrent connections per origin, which bounds how many streams a
page can hold open. Opening one stream per row in a table is therefore not an
option.
