This page lists every environment variable Rivly reads.

All configuration comes from the environment. There is no configuration file, no
flags, and no hardcoded path or secret. An invalid value fails at startup rather
than being silently ignored.

---

## Variables

| Variable | Default | Effect |
| --- | --- | --- |
| `RIVLY_ADDR` | `:8080` | Address the HTTP server listens on |
| `RIVLY_DATABASE` | `rivly.db` | Path to the SQLite file |
| `RIVLY_DATA` | `data` | Directory for the encryption key and stack working directories |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | Endpoint used to seed the local environment on first run |
| `RIVLY_POLL_INTERVAL` | `5s` | How often each environment is polled. Any Go duration |
| `RIVLY_COMPOSE_BIN` | detected | Compose command used to deploy managed stacks |
| `RIVLY_TRUSTED_ORIGINS` | empty | Comma-separated origins allowed to send unsafe requests, as `scheme://host` |
| `RIVLY_SETUP_TOKEN` | generated | Token required to claim the instance |
| `RIVLY_LOG_LEVEL` | `info` | Log verbosity: `debug`, `info`, `warn` or `error` |

---

## Notes

`DOCKER_HOST` is read once, to seed the `local` environment when the database is
empty. Changing it later has no effect on an existing environment, whose URL
lives in the database.

`RIVLY_DATA` holds `secret.key`, which decrypts every stored credential. Losing
it means losing every registry password and Git token. It must be persisted and
backed up.

`RIVLY_COMPOSE_BIN` is detected at startup when it is not set. Rivly probes
`docker compose` first, then `docker-compose`, and logs which one it resolved.

Setting it accepts a full command, not just an executable, so `docker compose`,
`docker-compose` and an absolute path to the plugin binary are all valid. The
executable must exist, otherwise the override is refused and reported at startup
rather than at the first deploy.

When nothing resolves, Rivly still starts and every other feature works, but
deploying a managed stack fails with an explicit error.

`RIVLY_TRUSTED_ORIGINS` is only needed when the dashboard is served from a
different origin than the API. An invalid origin aborts startup.

`RIVLY_LOG_LEVEL` is applied as soon as the configuration is read, so a failure
to parse the configuration itself is still reported at the default level.

`RIVLY_SETUP_TOKEN` pins the setup token instead of generating a new one, which
is what makes an automated first-run provisioning possible. It is ignored once an
account exists.
