# CI/CD Documentation

This directory contains GitHub Actions workflows for continuous integration and deployment.

## Workflows

### 1. CI Pipeline (`ci.yml`)

Runs on every push to `main`/`develop` branches and on all pull requests.

**Jobs:**

- **Lint** - Code quality checks using golangci-lint
- **Build** - Compile verification for all packages
- **Unit Tests** - Domain layer tests (no database required)
- **E2E Tests** - Integration tests with Spanner emulator
- **Proto Check** - Verify proto files are up-to-date
- **Security** - Security scanning with Gosec
- **Vulnerability Check** - Dependency vulnerability scanning with govulncheck

**Duration:** ~5-7 minutes

### 2. Release Pipeline (`release.yml`)

Runs when version tags are pushed (e.g., `v1.0.0`, `v1.2.3-rc1`).

**Steps:**

1. Run full test suite
2. Build binaries for multiple platforms:
   - Linux (amd64, arm64)
   - macOS (amd64, arm64)
3. Generate checksums
4. Create GitHub release with:
   - Binaries
   - Changelog
   - Checksums
5. (Optional) Build and push Docker images

**Artifacts:** Server and migration binaries for 4 platforms

### 3. Dependabot (`dependabot.yml`)

Automated dependency updates running weekly on Mondays at 9:00 AM.

**Updates:**
- Go modules
- GitHub Actions
- Docker images

## Environment Variables

### Required for E2E Tests
- `SPANNER_EMULATOR_HOST: localhost:9010`

### Optional
- `GO_VERSION: '1.23'` - Go version to use

## Infrastructure

### Docker Compose
The CI pipeline uses the existing `docker-compose.yml` to start the Spanner emulator:
- **Image:** `gcr.io/cloud-spanner-emulator/emulator:latest`
- **Ports:** 9010 (gRPC), 9020 (HTTP)
- **Commands:** `docker compose up -d` / `docker compose down -v`

This ensures **100% parity** between local development and CI environments.

## Running CI Locally

### Using Make (Recommended)

```bash
# Run complete CI pipeline
make ci-all

# Individual steps
make ci-lint        # Linting
make ci-build       # Build verification
make ci-test        # Unit tests
make ci-e2e         # E2E tests (requires Spanner emulator)
```

### Using GitHub Actions Locally (with `act`)

```bash
# Install act (https://github.com/nektos/act)
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Run CI workflow locally
act push

# Run specific job
act -j lint
act -j unit-tests
act -j e2e-tests
```

### Manual Steps

```bash
# 1. Start Spanner emulator
docker compose up -d

# 2. Run migrations
make migrate

# 3. Run tests
go test ./tests/e2e -v

# 4. Run linting
golangci-lint run ./...
```

## CI Status Checks

The following checks must pass before merging:

✅ **Linting** - Code style and quality
✅ **Build** - Compilation succeeds
✅ **Unit Tests** - Domain logic tests pass
✅ **E2E Tests** - Integration tests pass with Spanner
✅ **Proto Check** - Generated files are up-to-date
✅ **Security** - No security issues found
✅ **Vulnerability Check** - Dependencies are safe

## Troubleshooting

### E2E Tests Failing with "Cannot write timestamps in future"

This is a known issue with Spanner emulator clock skew. The tests include workarounds:
- Use `time.Now().Add(-5 * time.Minute)` for discount dates
- Ensure optimistic locking versions are fetched immediately before operations

### Proto Check Failing

Regenerate proto files locally and commit:

```bash
make proto
git add proto/
git commit -m "chore: regenerate proto files"
```

### Linter Issues

Fix automatically where possible:

```bash
make fmt              # Format code
golangci-lint run --fix ./...
```

### Go Module Issues

Clean and re-download:

```bash
go mod tidy
go mod download
```

## Performance Optimization

### Caching

The CI uses GitHub Actions caching for:
- Go modules (`~/go/pkg/mod`)
- Build cache

Cache is invalidated when `go.sum` changes.

### Parallelization

Jobs run in parallel where possible:
- `lint`, `build`, `unit-tests` run simultaneously
- `e2e-tests` runs separately (requires Spanner)

## Security

### Secrets

No secrets are required for CI. The pipeline uses:
- Public repositories only
- Spanner emulator (no credentials)
- GitHub token (automatically provided)

### SARIF Upload

Security scan results are uploaded in SARIF format for GitHub Security tab integration.

## Release Process

1. **Version Tag**: Create and push a version tag
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **Automatic**: GitHub Actions creates release with:
   - Binaries for all platforms
   - Changelog from git commits
   - Checksums file

3. **Manual**: Edit release notes if needed

## Badges

Add these to your README:

```markdown
[![CI](https://github.com/light-bringer/procat-service/workflows/CI/badge.svg)](https://github.com/light-bringer/procat-service/actions/workflows/ci.yml)
[![Release](https://github.com/light-bringer/procat-service/workflows/Release/badge.svg)](https://github.com/light-bringer/procat-service/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/light-bringer/procat-service)](https://goreportcard.com/report/github.com/light-bringer/procat-service)
[![codecov](https://codecov.io/gh/light-bringer/procat-service/branch/main/graph/badge.svg)](https://codecov.io/gh/light-bringer/procat-service)
```

## Contact

For CI/CD issues, open an issue or contact the maintainers.
