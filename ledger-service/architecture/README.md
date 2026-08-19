# Ledger Service — User Story Backlog

This directory contains the implementation backlog for the Valorio `ledger-service`.
The stories are ordered from an empty repository to a production-ready service. They
are intentionally small enough to be delivered independently while preserving a
working main branch.

## Product goal

Replace the personal BRL/USD budgeting spreadsheet with a reliable ledger API that
records income and expenses, organizes them by category, and produces monthly
financial views without mixing currencies.

## Assumptions and constraints

- Go is the service runtime and PostgreSQL is the source of truth.
- The service owns the `ledger_db` schema and its migrations.
- REST endpoints use the `/api/v1/ledger` prefix.
- Amounts are positive decimal values; `income` or `expense` determines their sign.
- BRL and USD are recorded and reported independently. Currency conversion belongs
  to the future `fx-service` integration.
- The first development phase uses a configured `DEFAULT_USER_ID` in local mode only.
  Keycloak/JWT integration is delivered later and must replace that fallback outside
  local development.
- Financial records and summaries are not cached in v1. PostgreSQL is fast enough at
  the expected scale and correctness is more valuable than cache complexity.
- Datadog is the observability backend for logs, metrics, traces, dashboards, and
  monitors. The application emits OpenTelemetry-compatible telemetry to the local
  Datadog Agent so domain code remains independent from the vendor backend.
- The expected scale is one or two users and tens of thousands, not millions, of
  entries. No sharding, read replicas, or event streaming is required for the first
  release.

## Backlog map

| Order | Epic | Outcome | Stories |
|---:|---|---|---|
| 1 | [Foundation](01-foundation.md) | A buildable, testable, containerized service with CI and a migrated database | LS-001–LS-006 |
| 2 | [Core ledger](02-core-ledger.md) | Categories and ledger entries can be managed safely | LS-007–LS-013 |
| 3 | [Insights](03-insights.md) | Monthly totals and comparisons are available | LS-014–LS-016 |
| 4 | [Operational readiness](04-operational-readiness.md) | The service is observable, secure, deployable, and documented | LS-017–LS-022 |
