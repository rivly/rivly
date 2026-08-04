This page states what Rivly must achieve. It is the reference against which a
feature is judged worth building.

---

## Product objectives

- Give a complete view of a Docker host: containers, images, volumes, networks
  and stacks, with their logs and their health.
- Make a single host and a Swarm cluster the same experience.
- Deploy and keep stacks in sync from a Git repository.
- Replace the day-to-day use of a terminal for operating containers, without
  pretending the terminal is unnecessary.

---

## Technical objectives

- Ship as one binary and one container image, with no companion service.
- Keep the install to a single container with the Docker socket mounted.
- Read live state from the daemon rather than maintaining a parallel model.
- Reflect a change on the host in the interface within a second.
- Stay responsive when a daemon is unreachable, using the last known snapshot.
- Keep the codebase small enough that a newcomer can read it in an afternoon.

---

## Operational objectives

- Survive a restart with no state loss beyond what Docker itself loses.
- Make backup obvious: one database file and one data directory.
- Fail loudly at startup on an invalid configuration, never silently.
- Never require a migration step that a user has to perform by hand.

---

## Non-objectives

Stated so they are not proposed again.

- Managing anything other than Docker. No Kubernetes, no Podman, no VMs.
- Hosting a control plane on behalf of users.
- Building a monitoring product. Rivly shows health, it does not store metrics
  history.
- Supporting deployments where Rivly is exposed to the public internet.
