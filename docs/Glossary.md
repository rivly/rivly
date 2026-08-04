This page defines the terms used across the Rivly documentation and codebase.

---

## Action

A verb applied to one or more Docker objects, such as start or remove. Actions
are always submitted in bulk and return one result per object, because a partial
failure is the normal case.

## ADR

Architecture Decision Record. A dated, immutable note recording why a structural
decision was made. See [ADR (Architecture Decision Records)](<ADR (Architecture Decision Records).md>).

## Environment

A Docker endpoint Rivly can talk to: a name, a kind and a URL. Every container,
image, volume, network and stack belongs to exactly one environment.

## External stack

A compose project discovered from container labels, deployed outside Rivly.
Visible and controllable, but its compose file is unknown to Rivly.

## Fingerprint

A hash of an environment's observable state, used by the poller to decide whether
anything actually changed before publishing an event.

## GitOps

Deploying from a Git repository and keeping the deployment in sync with it.
Automatic stack updates are attributed to the author name `GitOps`.

## Git stack

A managed stack whose compose file comes from a cloned repository rather than
from the database.

## Managed stack

A compose project Rivly deployed and remembers. It has a row in the database, an
owner, a history, and a compose file Rivly can show and edit.

## Poller

The background loop that queries every environment on a fixed interval,
regardless of daemon events. It guarantees the interface converges even when a
daemon is unreachable or its event stream is broken.

## Snapshot

The last successful system information of an environment, stored as JSON. It is
what lets a down environment still render something useful.

## Stack

A compose project: a set of containers sharing a `com.docker.compose.project`
label. Rivly distinguishes external and managed stacks.

## Setup token

A one-time secret generated at first start and printed to the logs, required to
create the first account. It prevents an unclaimed instance from being taken over
by whoever finds it first.

## Watcher

A background loop subscribed to the Docker event stream of one environment. It
gives sub-second reactivity, and is complemented by the poller.
