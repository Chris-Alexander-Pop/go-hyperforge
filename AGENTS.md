# Agent Notes

## Cursor Cloud specific instructions

This repo is a **Go library SDK** (`go-hyperforge` / module `github.com/chris-alexander-pop/system-design-library`), not a runnable multi-service app. Default quality gates use in-memory adapters and do not require Docker.

### Commands

See root `Makefile` / `README.md` / `CONTRIBUTING.md` for the standard workflow (`make setup`, `make check`, `make test`, `make build`, `make lint`, `make up`/`down`).

Ensure `$(go env GOPATH)/bin` is on `PATH` before linting (needed for `golangci-lint`).

### Optional local infra

`make up` starts Postgres (`5432`), Redis (`6379`), and NATS+JetStream (`4222`/`8222`) via `compose.yml`. Useful for exercising real adapters; **not required** for `make test` / CI.

If Docker is installed in this VM but the daemon is down: start `dockerd` (fuse-overlayfs storage driver), and ensure the agent user can reach `/var/run/docker.sock`.

### Gotchas

- Pre-push hook (`.github/hooks/pre-push`, installed by `make setup`) runs fmt + golangci-lint + vet + build + tests; use `git push --no-verify` only if intentionally skipping.
- Cloud/integration adapter tests skip without credentials (`AWS_*`, `GOOGLE_APPLICATION_CREDENTIALS`, Azure vars, etc.).
- `services/`, `templates/`, and `verification/` are largely placeholders; product code lives under `pkg/`.
