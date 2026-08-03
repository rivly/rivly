# AGENTS.md

## Scope

These instructions apply to the entire repository. A closer `AGENTS.md`, if one
exists, takes precedence over this one.

---

## Project

Rivly is an open-source dashboard for Docker, from a single host to a full Swarm.

It is self-hosted and MIT licensed. No accounts, no telemetry, no lock-in.

The intended packaging is a single binary: a Go backend serving the React
frontend embedded at build time.

---

## Repository structure

- `cmd/` entrypoints, limited to wiring and lifecycle.
- `internal/` the Go backend, packages grouped by responsibility.
- `web/` the React dashboard.
- `bruno/` a Bruno collection of API requests, kept in sync with the endpoints.
- `brand/` logo and brand assets.

---

## Commands

```bash
make run      # API on :8080
make build    # static binary -> bin/rivly
make test     # go test
make lint     # golangci-lint, must pass

cd web
bun run dev    # dashboard on :5173
bun run build  # tsc -b + vite build
bun run lint   # oxlint, must pass
```

Run the lint, the build and the tests of every area a change touches.

Report what was executed, what passed, and what could not run and why. Never
declare a task complete without that report.

---

## Working workflow

For every non-trivial task:

1. Explain the approach.
2. List the files that will be created or modified.
3. Explain the technical decisions and their trade-offs.
4. Wait for approval before writing code.

Never refactor or change the architecture without explicit approval.

Keep changes scoped to what was asked. Report anything else you spot instead of
fixing it.

---

## Verification

Do not answer from memory when something can be checked. Verification beats
prior knowledge, always.

Before a technical decision, read the code that is already there. If the answer
is in the repository, read the repository instead of assuming.

Verify anything that changes over time against its official source: library
versions, image versions, compatibility, deprecations, security advisories.

Generated code, templates and framework defaults are starting points, not
decisions. Review them before keeping them.

When you verify something, say what you checked, where, and what you concluded.

Label facts, observations, assumptions and recommendations for what they are.
Never present an assumption as a fact.

---

## Dependencies

Wait for approval before adding a dependency.

To propose one: name the latest stable release, confirm it works with the
current stack, and justify the version. Pin it. No floating or unbounded
constraints, no unmaintained packages.

---

## Self review

Before presenting a result, look for your own mistakes: wrong assumptions,
inherited template defaults, technical debt, risks you did not mention.

If a previous assumption turns out to be wrong, say so, explain the impact, and
give the corrected version.

---

## Security

Rivly drives a Docker daemon, so a defect here is a host-level defect.

Never weaken authentication, authorization, cryptography, secret handling or
input validation without explicit approval. Prefer secure defaults.

Never log a secret, and never return one in an API response.

---

## Git

Never run git. No commits, no branches, no tags, no history rewriting.

When the work is ready, hand over one single-line Conventional Commit message
and stop.

---

## Communication

Ask instead of guessing when requirements are ambiguous.

When several approaches are possible, give the alternatives, their trade-offs,
and your recommendation.

Never silently change architecture, documented behaviour or project conventions.

Always explain why, not only what.
