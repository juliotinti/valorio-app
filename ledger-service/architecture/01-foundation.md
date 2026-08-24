# Epic 1 — Service Foundation

Goal: turn the empty `ledger-service` directory into a repeatable engineering
baseline before adding business features.

## LS-001 — Scaffold the Go service repository

**As a** developer  
**I want** a conventional Go service structure and pinned toolchain  
**So that** all subsequent features have a clear home and reproducible build

**Dependencies:** None

### Acceptance criteria

- [x] A Go module is initialized using the repository's agreed module path and a
      supported, explicitly declared Go version.
- [x] `main` only performs composition and lifecycle management; business rules are
      not implemented in the entry point.
- [x] The service builds with `go build ./...` and tests with `go test ./...`.
- [x] `.gitignore`, `README.md`, and an example environment file are
      present; real secrets and local environment files are ignored.

### Verification

Run the documented bootstrap commands from a clean clone with only Go installed.

---

## LS-002 — Implement configuration and safe HTTP lifecycle

**As an** operator  
**I want** basic configuration validation and graceful startup/shutdown  
**So that** configuration errors fail fast and deployments stop safely

**Dependencies:** LS-001

### Acceptance criteria

- [x] Configuration is read from environment variables and includes HTTP address,
      log level, environment, shutdown timeout, service name, and service version.
- [x] Optional values have documented, safe local defaults.
- [x] Required values are validated at startup with actionable errors that do not
      expose secret values.
- [x] The HTTP server defines explicit read-header, read, write, and idle timeouts.
- [x] The process handles `SIGINT` and `SIGTERM`.
- [x] On shutdown, the server stops accepting new requests and gives in-flight
      requests a bounded period to finish.
- [x] Unit tests cover valid, missing, malformed, and unsafe configuration.

---

## LS-003 — Provision PostgreSQL for local development

**As a** developer  
**I want** a disposable local PostgreSQL instance  
**So that** development uses the same database engine as integration and deployment

**Dependencies:** LS-001

### Acceptance criteria

- [ ] Docker starts PostgreSQL with a health check, named volume, and a
      dedicated `ledger_db` database and least-privilege application user.
- [ ] Create a kubernetes yaml for ledger PostgreSQL.
- [ ] Credentials are configurable and development defaults are clearly marked as
      non-production values.
- [ ] The service waits for a usable connection with bounded retries and exits when
      the database remains unavailable.
- [ ] Connection-pool limits and lifetimes are configurable with conservative
      defaults appropriate for a small service.
- [ ] A documented command starts, verifies, and stops the database without deleting
      its volume by default.
- [ ] `DATABASE_URL` is introduced and validated as part of the PostgreSQL
      integration.
- [ ] The database connection pool is initialized during startup and closed
      during graceful shutdown.

---

## LS-004 — Establish versioned database migrations

**As a** developer  
**I want** repeatable, service-owned database migrations  
**So that** schema changes are reviewed and deployed predictably

**Dependencies:** LS-003

### Acceptance criteria

- [ ] `golang-migrate` or an equivalent agreed tool applies migrations from the
      service-owned `migrations` directory.
- [ ] Migration `000001` creates `categories` and `entries` with UUID primary keys,
      timestamps, ownership, foreign keys, and database constraints for entry type,
      currency, and positive amount.
- [ ] The initial schema supports nullable subcategory and institution fields and an
      optional card invoice month.
- [ ] Indexes support entry listing by owner/date and monthly aggregation by
      owner/category/date without speculative indexes.
- [ ] Up and down migrations are verified locally while the project is pre-release;
      production guidance states that later destructive changes use expand/contract.
- [ ] Applying all migrations twice is safe in the migration tool's normal workflow,
      and applying them from an empty database succeeds.
- [ ] Applied migrations are never edited after release; corrections use a new
      migration.

---

## LS-005 — Add continuous integration quality gates

**As a** maintainer  
**I want** every ledger-service change validated automatically  
**So that** defects and migration failures cannot silently enter the main branch

**Dependencies:** LS-001, LS-004

### Acceptance criteria

- [ ] A path-filtered GitHub Actions workflow runs for ledger-service pull requests
      and relevant shared workflow changes.
- [ ] CI runs formatting checks, `go vet`, unit tests and `go build`.
- [ ] The workflow builds the Docker image without publishing it on pull requests.
- [ ] Jobs use dependency/build caching without caching secrets or test state.
- [ ] Permissions default to `contents: read`; package write permission exists only
      in a future image-publishing job.
- [ ] Failures in any required gate block merge, and CI cancellation prevents stale
      runs from consuming resources for superseded commits.

---

## LS-005A — Integrate SonarQube Cloud code quality analysis

**As a** maintainer<br>
**I want** automated SonarQube Cloud analysis for the ledger-service<br>
**So that** code quality and security issues are detected before changes are merged

**Dependencies:** LS-005

### Acceptance criteria

- [ ] The repository is connected to a SonarQube Cloud organization and has a stable,
      documented project key for `ledger-service`.
- [ ] A version-controlled `sonar-project.properties` file defines the project base,
      source and test locations, Go coverage report path, and narrowly justified
      exclusions.
- [ ] CI generates Go test execution and coverage reports before analysis; coverage is
      imported with `sonar.go.coverage.reportPaths` and test execution data with
      `sonar.go.tests.reportPaths`.
- [ ] SonarQube Cloud analyzes bugs, vulnerabilities, security hotspots, code smells,
      duplicated code, and coverage for changed Go code.
- [ ] Pull requests that change `ledger-service` or a relevant shared dependency run
      SonarQube Cloud analysis and receive analysis status or decoration in GitHub.
- [ ] The SonarQube Cloud Quality Gate is a required check and blocks merging when it
      fails or when analysis does not complete successfully.
- [ ] The initial Quality Gate evaluates new code, avoiding arbitrary coverage debt
      from unrelated, generated, or pre-existing content.
- [ ] Generated files, vendored dependencies, migrations, and fixtures are excluded
      only when the exclusion is explicit and documented; production Go code is not
      hidden from analysis to improve metrics.
- [ ] `SONAR_TOKEN` is stored only as a GitHub Actions secret. The organization key,
      project key, and SonarQube Cloud URL are non-secret configuration and no
      credentials are committed or printed in logs.
- [ ] The CI workflow uses pinned or reviewed SonarSource actions/scanner versions and
      grants only the permissions needed for checkout and pull-request analysis.
- [ ] The README documents how to run tests and generate the local coverage report;
      developers are not required to run a local SonarQube server.

### Verification

Open a pull request with a ledger-service change and verify that test results and Go
coverage reach SonarQube Cloud, the analysis appears on the pull request, and a failing
Quality Gate prevents merge.

---

## LS-006 — Package the service as a secure container

**As an** operator  
**I want** a small, immutable service image  
**So that** the same artifact can run locally and in Kubernetes

**Dependencies:** LS-001, LS-002

### Acceptance criteria

- [ ] A multi-stage Dockerfile caches module download separately from source
      compilation and produces a statically linked Linux binary.
- [ ] The runtime image is distroless or equivalently minimal, pinned to a reviewed
      version or digest.
- [ ] The process runs as a non-root user and the runtime image contains no compiler,
      package manager, or shell.
- [ ] The binary embeds build version, commit SHA, and build date without changing
      application behavior.
- [ ] `.dockerignore` excludes Git data, local secrets, documentation artifacts, and
      build output not needed by the image.
- [ ] The container starts with read-only-root-filesystem compatibility and stops
      cleanly on `SIGTERM`.
- [ ] CI can build the image from a clean checkout and a smoke test verifies process
      startup with its required dependencies.
