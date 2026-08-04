This page describes how Rivly talks to a Docker daemon, and what it deliberately
never does.

---

## Client

`internal/docker` is the only package that opens a connection to a daemon. No
other package imports the SDK.

A `Manager` caches one client per environment, keyed by environment id. When the
stored URL of an environment changes, the cached client is discarded and rebuilt,
so a renamed host never keeps talking to the old socket.

Read calls carry a short timeout. Action calls carry a longer one. Streaming
calls carry none, because their lifetime is the client's.

---

## Reachability

An environment is up if the daemon answers, down otherwise. There is no
intermediate state.

When a daemon is unreachable, Rivly falls back to the last snapshot stored in
SQLite, so the environment page still shows what the host looked like the last
time it answered, marked as down.

---

## Discovery

Rivly reads compose labels to reconstruct a project view that Docker itself does
not expose.

| Label | Used for |
| --- | --- |
| `com.docker.compose.project` | Grouping containers, volumes and networks into a stack |
| `com.docker.compose.service` | Counting distinct services in a stack |
| `com.docker.compose.project.working_dir` | Showing where an external stack was deployed from |

A stack is running when every container is running, stopped when none is, and
partial otherwise.

---

## Streams

| Stream | Transport | Notes |
| --- | --- | --- |
| Container logs | SSE | Demultiplexed with stdcopy unless the container has a TTY, then read raw |
| Container stats | SSE | CPU and memory deltas computed from consecutive samples |
| Image pull | SSE | Progress messages forwarded as they arrive |
| Exec | WebSocket | Bridged onto a Docker exec session with a TTY |
| Daemon events | Internal | Filtered, debounced, republished on the event hub |

Daemon events are filtered before they reach the hub. Exec lifecycle and health
check events are noise for a dashboard and are dropped.

---

## Registry authentication

Image pulls resolve credentials by parsing the image reference, extracting its
registry domain, and looking up a stored registry for that domain.

When one exists, its password is decrypted and encoded into the auth header the
SDK expects. When none exists, the pull proceeds anonymously.

---

## Compose

Managed stacks are not deployed through the SDK. Rivly materialises a compose
file and an env file under the data directory, then invokes the Compose CLI with
`DOCKER_HOST` pointed at the target environment.

Git stacks work the same way, except the compose file comes from a clone rather
than from the database.

---

## What Rivly never does

- It never writes to the daemon outside an explicit user action. The poller and
  the watchers are read-only.
- It never keeps a daemon connection open per browser session. Clients are
  pooled per environment, not per user.
- It never stores container or image state. That data is read live and thrown
  away, apart from the environment snapshot.
- It never falls back to `github.com/docker/docker`.
