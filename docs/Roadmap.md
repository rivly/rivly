This page describes what is planned and what is deliberately deferred.

What already works is in [Features](Features.md).

---

## Before a first release

These are the gaps between what Rivly is and what it claims to be. The binary and
the image are done; see [Packaging](Packaging.md).

- **Published image.** A release pipeline that tags, builds and publishes it.
- **Environment management.** Create, edit and delete environments, so a remote
  host can be added without editing the database.

---

## Next

- **Swarm.** Services, nodes, tasks and scaling. Today only the Swarm flag and
  the node count are read, which does not honour the promise made in
  [Vision](Vision.md).
- **Roles.** The `role` column exists and nothing enforces it. Multi-user is not
  safe until it does. See [Security](Security.md).
- **Password reset.** The `tokens` table exists for it, and the interface already
  has a page saying it is coming.
- **Dark mode.** The token layer is ready for a second palette.

---

## Later

- Remote environments over TLS, with client certificates.
- Container health and restart history.
- Stack rollback to a previous commit for Git stacks.
- Notifications on stack failure.
- Audit log of who did what.

---

## Deliberately deferred

- Metrics history. Rivly shows current statistics; storing a time series is a
  different product.
- Any hosted component.
- Support for orchestrators other than Docker and Swarm.
