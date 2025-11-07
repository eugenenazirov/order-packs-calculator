# Repository Guidelines

## Overview & Layout
- Code entrypoint is `cmd/server/main.go`; it wires config loading, HTTP server bootstrap, and graceful shutdown tests (`cmd/server/main_shutdown_test.go`).
- Domain packages sit under `internal/`:
  - `internal/calculator`: DP-based packing engine with benchmarks.
  - `internal/api`: routers, handlers, middleware (rate limiters, recovery, logging).
  - `internal/storage`: in-memory pack-size store behind an interface for future adapters.
  - `internal/config`: layered config loader (flags, YAML, env, defaults).
- UI assets live in `web/` (static HTML/CSS/JS). Long-form references and diagrams live in `docs/` (`docs/api.md`, `docs/algorithm.md`).
- Tests mirror sources (e.g., `internal/calculator/calculator_test.go`), with extra suites in `test/`. Operational assets (`Dockerfile`, `docker-compose.yml`, `Makefile`) sit at repo root.

## Tooling & Commands
- Target Go ≥ 1.25.1. Install dev tooling via `make tools`, which fetches `golangci-lint` into `./bin`.
- Core workflow commands (see `Makefile`):
  - `make run` / `make build` for local execution or binaries.
  - `make test`, `make test-race`, `make coverage` (writes `coverage.out`) for validation.
  - `make lint` (depends on `make tools`) and `make fmt` (runs `golangci-lint --fix`) keep code formatted/imports ordered.
  - `make bench` targets `internal/calculator`.
  - `make tidy` and `make clean` manage modules and artifacts.
  - Container workflows rely on `make compose-up` / `make compose-down` (wrapper around `docker compose up --build` / `down`).
- The service listens on `http://localhost:8080`, serving both `/api/*` and the static UI.

## Coding Style & Quality Bar
- All Go code must stay gofmt/goimports clean; rely on `make fmt` for consistent formatting and `golangci-lint` for lint parity with CI.
- Follow idiomatic naming: exported identifiers use PascalCase, unexported helpers camelCase, avoid ALL_CAPS except true constants. Keep HTTP handlers thin—delegate validation/calculation/storage to their packages.
- Use structured logging (zap) helpers already wired in middleware; never log raw request bodies or secrets.
- When editing the front end, stick to vanilla HTML/CSS/JS conventions already present; if React (or other tooling) is introduced later, enforce Prettier defaults.

## Configuration & Runtime Expectations
- Configuration precedence: CLI flags → YAML (`config.yaml`) → environment variables → hard-coded defaults. `config.yaml.example` documents every field (ports, pack sizes, timeouts, rate limits).
- Key CLI flags/environment variables: `--config`, `--port`/`PORT`, `--pack-sizes`/`PACK_SIZES`, `--rate-limit-rps`/`RATE_LIMIT_RPS`, `--rate-limit-burst`/`RATE_LIMIT_BURST`. Zero-disable semantics apply to rate limits.
- Requests hit `/api/calculate` (POST), `/api/pack-sizes` (GET/PUT), and `/api/health` (GET). All payloads require items > 0 and 1–10 positive pack sizes; reject invalid input with `400`, impossible combos with `422`.
- Rate limiting uses a global token bucket tuned via the config above; keep it enabled unless a benchmark explicitly disables it.

## Testing Strategy
- Maintain ≥80% coverage, emphasizing `internal/calculator` edge cases such as `[23, 31, 53] → 500000`.
- Co-locate unit tests with implementations; use `testdata/` when large fixtures are needed. Name integration tests after the exercised endpoint (e.g., `TestCalculatePacks_ExactMatch`).
- Run `make test-race` before landing concurrency-sensitive changes, and `make bench` when touching algorithmic code paths.
- Add benchmarks (`BenchmarkCalculatePacks_*`) alongside algorithm tweaks to guard regressions.

## Security, Deployment & Ops
- Validate every inbound payload server-side (items > 0, ≤10 pack sizes, all positive). Reject malformed JSON before touching business logic.
- Middleware already enforces request IDs, panic recovery, and optional structured access logs—leave them enabled in prod.
- Docker image builds via the provided multi-stage `Dockerfile` and must run as a non-root user; do not add root-only operations. Health checks target `/api/health`.
- Expose new toggles through `internal/config` and document them in `README.md` plus `config.yaml.example`.

## Git & Review Hygiene
- Follow Conventional Commits (`feat(calculator): memoize large order paths`). Keep commits reviewable and focused.
- Every PR must include: change summary, linked issue/spec clause, verification evidence (`make test`, `make lint`, or targeted equivalents), and UI screenshots when visual output changes.
- Avoid bundling formatting-only changes with feature work; run `make fmt`/`make lint` before pushing.
