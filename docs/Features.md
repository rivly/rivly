This page lists what Rivly does today. Anything not listed here is not built.

Planned work lives in [Roadmap](Roadmap.md).

---

## Environments

- A `local` environment is created at first start from `DOCKER_HOST`.
- System information is shown per environment: Docker version, OS, architecture,
  kernel, CPU count, memory, container and image counts, Swarm state.
- An unreachable environment is marked down and still renders its last known
  snapshot.

---

## Containers

- List with state, image, stack, ports, IP and creation date.
- Detail view with command, restart policy, ports, networks, mounts, environment
  variables and labels.
- Actions: start, stop, restart, pause, unpause, kill, remove. Available in bulk.
- Create a container: image, name, command, environment variables, port
  mappings, mounts, network and restart policy. The image is pulled if missing.
- Live logs, with a configurable tail.
- Live statistics: CPU, memory, network and block IO, process count.
- Interactive terminal in the container.

---

## Images

- List with tags, size, creation date and whether the image is in use.
- Detail view with digests, architecture, entrypoint, command, environment,
  exposed ports, labels and the containers using it.
- Pull an image, with live progress.
- Remove, in bulk. Prune, either dangling only or every unused image.

---

## Volumes and networks

- List volumes with driver, mountpoint, owning stack and usage.
- List networks with driver, scope, owning stack and usage.
- Detail views listing the containers attached to each.
- Create a volume with a driver. Create a network with a driver and a subnet.
- Remove, in bulk. Predefined networks are protected.

---

## Stacks

- **External stacks** are discovered from compose labels, so a project deployed
  outside Rivly is visible and controllable.
- **Managed stacks** are deployed by Rivly from a compose file written in the
  editor or uploaded, with environment variables.
- **Git stacks** are deployed from a repository, with a ref and a path to the
  compose file, authenticated with a stored credential.
- Automatic updates: Rivly watches the remote commit and redeploys when it moves,
  at a configurable interval. A stopped stack is not restarted.
- Actions: start, stop, restart, remove. Removing a managed stack tears it down
  and forgets it.
- Every stack shows who deployed it and when. Automatic updates are attributed to
  GitOps.

---

## Credentials

- **Registries**: name, server, username and password, encrypted at rest. Used
  automatically when pulling an image from that registry. A connection test is
  available.
- **Git credentials**: name, username and token, encrypted at rest. Selected per
  Git stack.

---

## Account

- One-time instance claiming, protected by a setup token.
- Sign in and sign out.
- Change display name and password. Changing a password signs out every other
  session.

---

## Realtime

- A single event stream keeps the interface in sync with the host, driven by both
  Docker events and a periodic poll.
