# Valorio Ledger Service

The Ledger Service is the budgeting and cash-flow component of Valorio. It replaces
the manual BRL/USD budgeting workflow currently maintained in spreadsheets with a
reliable API for recording income and expenses, organizing financial entries, and
producing monthly insights.

> See the [implementation backlog](architecture/README.md) for
> the current delivery sequence.

## Responsibilities

The service owns the following capabilities:

- Create, edit, retrieve, list, and soft-delete income and expense entries.
- Create, rename, list, and archive user-defined categories.
- Filter entries by date, category, institution, currency, and entry type.
- Calculate monthly income, expenses, and net cash flow.
- Group monthly results by category and institution.
- Compare category spending between two months.
- Keep each owner's financial records isolated.

## Domain boundaries

The Ledger Service records budgeting facts. It does not manage investments, exchange
rates, or currency holdings.

- BRL and USD entries are stored and reported independently.
- The service never converts currencies or combines BRL and USD totals.
- Buying USD as an asset belongs to `fx-service`.
- Portfolio transactions and holdings belong to `portfolio-service`.
- Cross-domain dashboards belong to `dashboard-bff` when composition is required.

Keeping these concerns separate avoids distributed transactions for ordinary ledger
operations and allows this service to remain the source of truth for budgeting data.

## Core business rules

- Every entry is either `income` or `expense`.
- Amounts are always positive decimal values. The entry type determines their effect
  on the net total.
- Supported currencies in v1 are `BRL` and `USD`.
- Financial calculations must not use binary floating-point values.
- A new entry must reference an active category owned by the same user.
- Archived categories remain attached to historical entries but cannot be selected
  for new or recategorized entries.
- Deleted entries are soft-deleted for auditability and do not appear in ordinary
  reads, summaries, or comparisons.
- Monthly results are calculated on read from PostgreSQL. They are not cached or
  stored in a summary table in v1.
- Percentage change is `null` when the base value is zero; the API never returns
  `NaN` or infinity.

## Planned API

All business endpoints use JSON over HTTP under the `/api/v1/ledger` prefix.

### Categories

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/v1/ledger/categories` | Create a category |
| `GET` | `/api/v1/ledger/categories` | List categories |
| `PUT` | `/api/v1/ledger/categories/{id}` | Update a category |
| `DELETE` | `/api/v1/ledger/categories/{id}` | Archive a category |

### Entries

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/v1/ledger/entries` | Create an income or expense entry |
| `GET` | `/api/v1/ledger/entries/{id}` | Retrieve one entry |
| `GET` | `/api/v1/ledger/entries` | List and filter entries using cursor pagination |
| `PUT` | `/api/v1/ledger/entries/{id}` | Replace an entry's editable data |
| `DELETE` | `/api/v1/ledger/entries/{id}` | Soft-delete an entry |

### Insights

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/api/v1/ledger/monthly-summary?month=YYYY-MM` | Get monthly totals and breakdowns |
| `GET` | `/api/v1/ledger/monthly-comparison?base=YYYY-MM&compare=YYYY-MM` | Compare category spending between two months |

The final request and response schemas will be published as an OpenAPI 3.1 contract.
Money will use a lossless JSON representation, and all failures will follow a stable
error envelope with machine-readable error codes.

## Technology and architecture

The planned implementation uses:

- **Go** for the service runtime.
- **PostgreSQL** as the source of truth, with a dedicated `ledger_db` database.
- **Versioned SQL migrations** owned by this service and applied with
  `golang-migrate` or an agreed equivalent.
- **Docker** with a multi-stage, non-root, minimal runtime image.
- **Kubernetes** for local and future environment deployments.
- **Kong Gateway** for external routing, CORS, timeouts, and rate limiting.
- **Keycloak** for identity, with JWT validation provided by the shared Go
  authentication library.
- **OpenTelemetry and Datadog** for logs, metrics, distributed traces, dashboards,
  monitors, and deployment correlation.
- **SonarQube Cloud** for static analysis, security findings, code-quality metrics,
  pull-request feedback, and the required Quality Gate.
- **GitHub Actions** for build, test, migration validation, image publication, and
  controlled releases.

Application code will be organized so that HTTP transport, use cases, domain rules,
and PostgreSQL persistence remain separate. The executable entry point is responsible
only for configuration, dependency composition, and process lifecycle.

## Data ownership and migrations

The service exclusively owns its schema. Other Valorio services must not query or
modify `ledger_db` directly.

The initial migration will create `categories` and `entries` with UUID identifiers,
ownership, timestamps, foreign keys, financial constraints, and indexes for the
primary listing and monthly aggregation paths. Once a migration has been applied in a
shared environment, it is immutable; corrections are delivered through a new
migration. Production-breaking schema changes must follow the expand/contract pattern.

## Authentication and authorization

During the earliest local development phase, the service may use a configured
`DEFAULT_USER_ID`. This fallback is permitted only in the explicit local environment.

The production model uses Keycloak bearer tokens:

- The token `sub` claim identifies the record owner.
- Authenticated `owner` and `viewer` roles may read data.
- Only the `owner` role may create, update, archive, or delete data.
- Requests for another owner's resources return `404` to avoid revealing their
  existence.
- Missing or invalid credentials return `401`; insufficient permission returns `403`.

## Observability

The application will emit OpenTelemetry-compatible telemetry to a local Datadog Agent.
Datadog is the operational backend for logs, metrics, traces, dashboards, and monitors.

Telemetry uses the unified tags:

```text
env:<environment>
service:ledger-service
version:<release-or-commit>
```

Logs and traces must never record authorization tokens, database credentials, entry
descriptions, amounts, raw request bodies, or SQL parameter values. Metrics use
bounded-cardinality tags; user IDs and entry IDs are not metric tags. Telemetry export
must not cause a business request to fail when Datadog is unavailable.

## Health and lifecycle

The service will expose separate health signals:

- `/healthz` reports whether the process is alive.
- `/readyz` reports whether required dependencies, including PostgreSQL, are usable.

Startup validates configuration before accepting traffic. On `SIGINT` or `SIGTERM`,
the service stops accepting new requests, gives in-flight requests a bounded period to
finish, flushes telemetry, and closes its PostgreSQL connection pool.

### Configuration

Configuration is read from environment variables at startup. `SERVICE_NAME` and
`SERVICE_VERSION` are required. All other values are optional and use the safe local
defaults below when unset; every default can be overridden through its environment
variable.

|          Variable          | Default  |                      Validation                            |
|----------------------------|----------|------------------------------------------------------------|
| `SERVICE_NAME`             | required | Single-line identifier, at most 128 characters             |
| `SERVICE_VERSION`          | required | Single-line identifier, at most 128 characters             |
| `HTTP_ADDRESS`             | `localhost:8080`  | `host:port`, with port 1–65535                    |
| `LOG_LEVEL`                | `info`   | `debug`, `info`, `warn`, or `error`                        |
| `ENVIRONMENT`              | `local`  | `local`, `development`, `test`, `staging`, or `production` |
| `SHUTDOWN_TIMEOUT`         | `50s`    | Greater than zero and no more than 5 minutes               |
| `HTTP_READ_HEADER_TIMEOUT` | `5s`     | Greater than zero and no more than 5 minutes               |
| `HTTP_READ_TIMEOUT`        | `15s`    | Greater than zero and no more than 5 minutes               |
| `HTTP_WRITE_TIMEOUT`       | `15s`    | Greater than zero and no more than 5 minutes               |
| `HTTP_IDLE_TIMEOUT`        | `60s`    | Greater than zero and no more than 5 minutes               |

Duration values use Go duration syntax, such as `500ms`, `10s`, or `1m`. See
`.env.example` for a local template. The service reports invalid variable names and
expected formats without echoing their values.

## Testing strategy

- Unit tests cover configuration, domain validation, money rules, and calculations.
- Repository integration tests run against real PostgreSQL and validate migrations,
  constraints, owner isolation, and queries.
- HTTP tests cover successful responses and error contracts.
- CI produces native Go test and coverage reports and imports them into SonarQube
  Cloud; SonarQube reports coverage but does not replace test execution.
- The SonarQube Cloud Quality Gate evaluates new code and must pass before changes can
  be merged or a release image can be published.
- External dependencies such as Keycloak and Datadog are replaced with controlled test
  doubles or local test endpoints; tests do not send data to real external accounts.

## Delivery roadmap

Implementation is split into four epics:

1. [Service Foundation](architecture/01-foundation.md) — Go scaffold, lifecycle,
   PostgreSQL, migrations, CI, SonarQube Cloud, and container image.
2. [Core Ledger](architecture/02-core-ledger.md) — categories and entry management.
3. [Ledger Insights](architecture/03-insights.md) — monthly summaries, breakdowns, and
   comparisons.
4. [Operational Readiness](architecture/04-operational-readiness.md) — OpenAPI,
   health checks, Datadog, Keycloak, Kubernetes, and controlled releases.

The first meaningful vertical slice includes category creation, entry creation, entry
retrieval, PostgreSQL persistence, migrations, and integration tests. An in-memory-only
prototype does not complete that milestone.

## Authors <img src="https://content.linkedin.com/content/dam/me/business/en-us/amp/brand-site/v2/bg/LI-Bug.svg.original.svg" width="25" height="25" alt="LinkedIn" />

- [Julio Tinti](https://www.linkedin.com/in/juliotinti/)

## License

This project is licensed under the MIT License. See the [LICENSE](../LICENSE) file for
details.
