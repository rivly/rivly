This page describes how Rivly is meant to be built, shipped and run.

---

## Target

One binary, one container image, one volume. Installing Rivly should mean running
a single container with the Docker socket mounted, and nothing else.

That target is what justifies most of the constraints elsewhere: SQLite instead
of a database server, a pure Go driver instead of CGO, go-git instead of the
`git` binary.

---

## Build

```bash
make build    # CGO_ENABLED=0 go build -ldflags="-w -s" -o bin/rivly
```

`CGO_ENABLED=0` is not optional. It is what keeps the binary static and
self-contained, and it is why `modernc.org/sqlite` is used. See
[Tech Stack](<Tech Stack.md>).

---

## Runtime state

Two paths must be persisted:

- the SQLite file at `RIVLY_DATABASE`;
- the data directory at `RIVLY_DATA`, which holds `secret.key` and the working
  directories of managed stacks.

Losing `secret.key` means losing every stored registry password and Git token.
See [Configuration](Configuration.md).

---

## The dashboard

`web/dist` is embedded with `go:embed` and served from the router's not-found
handler. A request that matches an embedded file gets that file; anything else
gets `index.html`, so client-side routes survive a page reload. Requests under
`/api/` are excluded, so an unknown endpoint still answers JSON rather than HTML.

Hashed assets under `/assets/` are cached for a year and marked immutable.
Everything else, `index.html` first of all, is served `no-cache`: a cached index
would pin a browser to an old build forever.

`make build-all` builds the dashboard and then the binary. `make build` alone
compiles against whatever is in `web/dist`, which is what keeps backend-only
development fast. When nothing is embedded, the server logs a warning at startup
and serves the API only.

A placeholder keeps `web/dist` present in a fresh clone, so the Go package
compiles before anyone has run a frontend build.

---

## Running the image

```bash
docker run -d \
  -p 8080:8080 \
  -v rivly:/data \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --group-add "$(stat -c %g /var/run/docker.sock)" \
  ghcr.io/rivly/rivly
```

The `--group-add` is not optional. The container runs as an unprivileged user, so
it cannot read a socket owned by `root:docker` without being given that group.
Skipping it produces an environment stuck in the down state, which is the first
thing a new user will hit.

`/data` holds the database, the encryption key and the working directories of
managed stacks. It is the only path that must be persisted.

The setup token is printed to the container logs at first start.

---

## Current state

The image builds and runs. Reaching the target still requires a release pipeline
that runs lint, test and govulncheck, then builds and publishes the image on a
tag.

---

## The image

`make image` builds it. Three stages: the dashboard with Bun, the binary with Go,
then a small Alpine runtime that receives the binary and the Compose plugin
copied from the official `docker/compose-bin` image. Pinning that image pins the
Compose version, so behaviour is reproducible across installs rather than
depending on what the host happens to have.

It is not distroless. Both binaries are static, so distroless was possible, but
the image keeps a shell so a self-hoster can debug it and so `HEALTHCHECK` works.
See [ADR-0016](<ADR/0016 - A minimal base image carrying the Compose plugin.md>).

The container runs as uid 10001 and owns nothing outside `/data`.
