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

`CGO_ENABLED=0` is not optional. It is what allows a static binary in a
distroless image, and it is why `modernc.org/sqlite` is used. See
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

## Current state

The dashboard is not embedded yet. The Go binary serves the API only, and the
frontend is built and served separately during development.

There is no Dockerfile and no published image.

Reaching the target requires:

1. building `web/` and embedding `dist/` with `go:embed`;
2. serving it with an SPA fallback, and a real Content-Security-Policy rather
   than the current frame-ancestors-only header;
3. a distroless image;
4. a release pipeline that runs lint, test and govulncheck before publishing.

---

## The compose problem

Managed stacks are deployed by invoking the Compose CLI as a subprocess. A
distroless image contains no shell and no Compose binary, so the two goals are
currently incompatible.

The options are to ship a base image that includes the Compose plugin, to
implement compose deployment through the SDK, or to drop managed stacks from the
minimal image. None has been decided, and the decision belongs in an ADR.
