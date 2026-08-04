This page describes how Rivly is tested and what "done" means.

---

## Definition of done

A change is done when the lint, the build and the tests of every area it touches
pass, and when that has been reported rather than assumed.

Documentation that the change invalidates is updated in the same change. An
implementation and a note are never allowed to diverge.

---

## Commands

```bash
make lint     # golangci-lint, must pass
make test     # go test
make build    # static binary

cd web
bun run lint   # oxlint, must pass
bun run build  # tsc -b + vite build
```

---

## Testing strategy

Tests exercise real dependencies wherever a real dependency is cheap, and use a
test double only at the boundary Rivly does not own.

In practice: server tests run a real `httptest` server against a real SQLite
database on a temporary path, with a fake Docker and a fake Compose runner. The
HTTP layer, the router, the middleware chain, the session store and the SQL are
all genuinely executed. Only the daemon is faked.

That is a deliberate choice over mocking every collaborator. A mock asserts that
code was called; this asserts that it works.

---

## Conventions

Prefer table-driven tests with named cases and `t.Run`, using field names in the
case struct so a reordering cannot silently change a test.

Call `t.Parallel()` on cases that do not share mutable state.

Every helper calls `t.Helper()`, so a failure points at the test rather than at
the helper.

Test concurrent and time-dependent code with `testing/synctest` rather than
sleeping. Timers, retries and polling loops become deterministic and instant.

Name a test after the behaviour it protects, not after the function it calls.

---

## Coverage

Coverage is a signal, not a target. The rule is simpler: any logic that would
silently corrupt state or leak a secret is tested, whatever the percentage says.

The areas that matter most are the ones with state and retries: the pollers, the
stack lifecycle, the credential stores, and everything under
[Security](Security.md).

---

## Linting

`golangci-lint` runs errcheck, govet, ineffassign, staticcheck and unused.

A deprecation flagged by the linter is fixed, never silenced. If a rule is wrong
for this project, disable it in the configuration with a reason, not inline.

---

## Dependencies

Adding a dependency requires approval. Before proposing one, verify the latest
stable release, confirm it works with the current stack, and justify the version.

Pin versions. No floating constraints, no unmaintained packages.

Run `govulncheck` before a release.

---

## Continuous integration

Not set up yet. Tracked in [Roadmap](Roadmap.md).

Until it exists, the validation commands above are run locally and their result
reported in the change.
