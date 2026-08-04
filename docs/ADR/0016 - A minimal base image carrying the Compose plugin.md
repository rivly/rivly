| Field | Value |
| --- | --- |
| **Identifier** | ADR-0016 |
| **Date** | 2026-08-04 |
| **Status** | Accepted |

> **Amended 2026-08-04.** The context below claims a distroless image has no
> Compose binary to shell out to. That is wrong: the Compose binary is static
> and can be copied into a distroless image. The decision stands on the two
> reasons given under Consequences, a shell for debugging and a working
> container healthcheck, not on the impossibility stated here.

---

## Context

[ADR-0011](<0011 - Deploying managed stacks through the Compose CLI.md>) deploys managed stacks by invoking the Compose CLI as a subprocess.
[ADR-0003](<0003 - A pure Go build with CGO disabled.md>) targets a distroless image, which contains no shell and no Compose
binary. The two are incompatible, and that contradiction has blocked the
container image, and therefore the first installable release.

Three options were considered.

Reimplementing compose through the SDK removes the dependency entirely, and was
already rejected by ADR-0011: the specification is large and moving, and a
partial implementation would mis-deploy files that work everywhere else.

Shipping distroless and letting managed stacks fail keeps the image minimal, but
removes a headline feature from the default install. A dashboard that cannot
deploy a stack out of the box is not the product described in
[Vision](../Vision.md).

Shipping a base image that carries the Compose plugin costs image size and
reintroduces a shell.

---

## Decision

The production image is built on a minimal base that includes the Docker Compose
plugin. It is not distroless.

`CGO_ENABLED=0` stays in force. The binary remains static and self-contained, and
every other constraint from ADR-0003 is unchanged: no CGO dependency, pure Go
SQLite, go-git rather than the `git` binary.

The single artifact promised by [ADR-0001](<0001 - A single self-hosted binary.md>) is the image, not the base layer. What
matters to a user is one container to run, not what that container is built from.

---

## Consequences

The image is larger than distroless and contains a shell, which widens the attack
surface for anyone who obtains code execution inside the container. That
container already holds the Docker socket, so it is not the weakest link, but the
image must still run as a non-root user and carry nothing beyond the binary, the
plugin and what the healthcheck needs.

The shell is what makes the container debuggable by someone who self-hosts it and
has nobody to escalate to. It is also what lets `HEALTHCHECK` work without giving
the binary a second job.

Running as a non-root user means the Docker socket is not readable by default.
Users must pass the socket group explicitly, which is documented in
[Packaging](../Packaging.md).

Compose is now a versioned part of the image rather than an ambient dependency of
the host, so its behaviour is reproducible across installs.

`RIVLY_COMPOSE_BIN` keeps its meaning for people running the binary outside the
image.

Base image updates become a security concern of the project rather than of the
user, so the release pipeline has to rebuild on base image advisories.

The `go:embed` work and the Dockerfile are unblocked.
