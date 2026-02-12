# Repository Guidelines

## Project Structure & Module Organization
This service is a Go gRPC application with DDD/Clean Architecture boundaries.

- `cmd/server/`: application entrypoint.
- `internal/app/product/`: core product logic.
- `internal/app/product/domain/`: pure business rules and entities.
- `internal/app/product/usecases/` and `internal/app/product/queries/`: command/query handlers.
- `internal/app/product/repo/`, `internal/models/`, `internal/transport/grpc/`: persistence, DB mappings, and transport adapters.
- `proto/product/v1/`: protobuf contracts and generated gRPC code.
- `migrations/`: Spanner schema migrations.
- `tests/integration/`, `tests/e2e/`, `tests/testutil/`: integration, end-to-end, and test helpers.

## Build, Test, and Development Commands
- `make deps`: download and tidy Go modules.
- `make tools`: install local dev tools (`protoc` plugins, `grpcurl`, `gotestsum`).
- `make proto`: regenerate protobuf stubs.
- `make docker-up && make migrate && make run`: start emulator, apply schema, run service.
- `make build`: build binary to `bin/server`.
- `make test-unit`: fast domain tests (no DB).
- `make test-integration` / `make test-e2e`: Spanner emulator-backed suites.
- `make test-all`: run all suites.
- `make check`: run formatting, vet, and lints.

## Coding Style & Naming Conventions
- Use standard Go formatting (`make fmt` / `go fmt ./...`); do not hand-format.
- Run static checks before pushing: `make vet` and `make lint`.
- Keep package names short, lowercase, and descriptive.
- Follow existing file patterns: `interactor.go`, `query.go`, `*_repo.go`, `*_test.go`.
- Keep domain layer dependency-free (no transport/DB imports in `domain/`).

## Testing Guidelines
- Frameworks: Go `testing` + `testify` (`require`/`assert`).
- Test files must end in `*_test.go`; test funcs should be `TestXxx`.
- Integration tests live in `tests/integration/` and use the `integration` tag where applicable.
- Prefer `make test-all` before opening a PR; generate coverage via `make test-coverage`.

## Commit & Pull Request Guidelines
- Follow existing history style: imperative, concise subjects (for example, `Add gRPC endpoint integration tests`).
- Keep commits focused by concern (domain, transport, tests, tooling).
- PRs should include:
  - what changed and why,
  - how it was tested (exact commands),
  - related issue/task links,
  - notes for schema/proto changes (`make proto`, migration impact).

## Security & Configuration Tips
- Use `.env.example` as the baseline; never commit secrets.
- Use `.env.test` and test docker compose for isolated test runs.
- Set `SPANNER_EMULATOR_HOST` through existing `make` targets instead of ad hoc shell state.
