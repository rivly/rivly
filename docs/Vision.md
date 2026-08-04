## Why Rivly

Running containers on your own hardware is normal again. The tooling has not
followed.

The options are a terminal, a dashboard that stopped at single-host management,
or a control plane designed for a cluster you do not have. Teams end up running
several tools and switching between them depending on how big the thing in front
of them is.

**Rivly manages a single host and a full Swarm the same way, from one interface
you host yourself.** No accounts, no telemetry, no lock-in, and no paid tier
holding a feature hostage.

---

## Principles

- **Self-hosted with no strings.** Rivly runs on your hardware, talks to your
  daemon, and phones nobody. There is no cloud component and there will not be
  one.
- **One interface, whatever the scale.** A single host and a cluster are the same
  product, not two.
- **Docker is the source of truth.** Rivly reads the daemon rather than keeping a
  parallel model of reality that can drift from it.
- **Open source, entirely.** MIT, with no feature withheld for a commercial
  edition.
- **Installable in one step.** One container, one volume. Complexity in the
  install is a defect.
- **Honest about danger.** Rivly holds a Docker socket. It says so, and it is
  built accordingly.

---

## What Rivly refuses to be

- A hosted service that manages your infrastructure from someone else's server.
- An abstraction that hides Docker instead of exposing it.
- A tool that needs a database, a queue and a cache to show a list of containers.
- A free tier.
