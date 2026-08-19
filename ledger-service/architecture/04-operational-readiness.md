# Epic 4 — Operational Readiness

Goal: make the completed ledger behavior safe to operate behind Kong and in
Kubernetes. These stories harden the service; they do not expand financial scope.

## LS-017 — Publish and validate the OpenAPI contract

**As an** API consumer  
**I want** a versioned, machine-readable ledger contract  
**So that** frontend and gateway integrations do not depend on implementation guesses

**Dependencies:** LS-007–LS-016

### Acceptance criteria

- [ ] An OpenAPI 3.1 document describes every v1 endpoint, filter, cursor, schema,
      status code, and standard error response.
- [ ] Money values are represented as decimal strings or another explicitly agreed
      lossless JSON representation with examples.
- [ ] Contract validation/lint runs in CI and detects incompatible undocumented
      changes.
- [ ] Request and response examples include BRL and USD, validation failures, empty
      collections, and pagination.
- [ ] The service version and API version are independent and documented.

---

## LS-018 — Add health, readiness, and build information endpoints

**As an** orchestrator  
**I want** distinct health signals  
**So that** unhealthy instances are restarted and unready instances receive no traffic

**Dependencies:** LS-002, LS-003

### Acceptance criteria

- [ ] `/healthz` reports only process liveness and does not fail because PostgreSQL is
      temporarily unavailable.
- [ ] `/readyz` checks PostgreSQL with a strict timeout and fails when the service
      cannot safely serve requests.
- [ ] Health endpoints are lightweight, do not leak configuration, and are excluded
      from ordinary access-log noise or sampled appropriately.
- [ ] A build-information endpoint or health metadata exposes version and commit SHA
      without exposing secrets.
- [ ] Docker and Kubernetes probes use the appropriate endpoints and timings.

---

## LS-019 — Integrate logs, metrics, and traces with Datadog

**As an** operator  
**I want** correlated telemetry for every request in Datadog  
**So that** failures and latency can be diagnosed across Kong, the service, and PostgreSQL

**Dependencies:** LS-002, LS-018

### Acceptance criteria

- [ ] Logs are structured JSON and include service, environment, severity, request
      ID, trace ID, route, method, status, duration, and user ID when known.
- [ ] Telemetry uses the Datadog unified service tags `env`, `service`, and `version`,
      with `service:ledger-service`, so logs, metrics, traces, and deployments can be
      correlated in one service view.
- [ ] Incoming request IDs and W3C trace context are preserved; missing values are
      generated and returned to callers where appropriate.
- [ ] The application uses OpenTelemetry APIs and exports OTLP telemetry to the local
      Datadog Agent; exporter failures never fail a business request and use bounded
      buffering, timeouts, and shutdown flushing.
- [ ] Distributed traces cover inbound HTTP handling and PostgreSQL calls without
      recording financial payloads, SQL parameter values, or high-cardinality resource
      names. Trace context propagates through Kong using W3C Trace Context.
- [ ] Runtime and custom metrics expose request rate, error rate, latency, in-flight
      requests, Go runtime health, and database-pool saturation with bounded-cardinality
      tags. Entry IDs, user IDs, descriptions, and raw URL/query values are prohibited
      as metric tags.
- [ ] Logs are collected by the Datadog Agent and automatically correlated with APM
      traces using trace and span IDs.
- [ ] Tokens, database URLs, descriptions, amounts, and raw request bodies are never
      logged by default.
- [ ] Panic recovery emits a correlated error and returns the standard `500` envelope
      without crashing the process.
- [ ] Telemetry filtering/redaction is covered by tests, including representative
      financial input and authorization headers.
- [ ] Sampling defaults are configurable by environment: full or high sampling is
      allowed locally, while non-local environments define an explicit ingestion and
      retention budget.

---

## LS-019A — Create Datadog dashboards and actionable monitors

**As an** operator  
**I want** a focused ledger-service dashboard and actionable monitors  
**So that** I can detect user-facing failures without manually searching raw telemetry

**Dependencies:** LS-019

### Acceptance criteria

- [ ] A version-controlled dashboard definition shows request rate, 4xx/5xx rate,
      p50/p95/p99 latency, PostgreSQL latency/errors, connection-pool saturation, pod
      restarts, CPU/memory, and service version.
- [ ] Dashboard filters use the unified `env`, `service`, and `version` tags and can
      distinguish local/staging/production without duplicating dashboards.
- [ ] Monitors exist for sustained 5xx rate, readiness failure, abnormal p95 latency,
      database connectivity/pool exhaustion, missing telemetry, and Kubernetes
      CrashLoopBackOff/restart behavior.
- [ ] Each monitor defines evaluation window, threshold, recovery condition, no-data
      behavior, severity, owner, and a concise runbook link; low-volume traffic is
      handled without noisy percentage-only alerts.
- [ ] Synthetic or API checks validate `/healthz` and the externally routed readiness
      path where appropriate; business write checks never create uncontrolled
      financial records.
- [ ] A monthly usage review covers indexed log volume, custom metric cardinality,
      trace ingestion/retention, and synthetic test frequency, with alerts or budgets
      configured when supported by the Datadog plan.
- [ ] Dashboard and monitor definitions contain no API/application keys and can be
      applied reproducibly through the chosen infrastructure-as-code mechanism.

---

## LS-020 — Integrate Keycloak authentication and resource authorization

**As a** ledger owner  
**I want** my data protected by validated identity and roles  
**So that** no other user can read or mutate my financial records

**Dependencies:** LS-007–LS-019A and the shared Go `authkit` delivery

### Acceptance criteria

- [ ] Every business endpoint validates a bearer JWT locally using the configured
      Keycloak issuer, audience, signature, expiration, and not-before claims.
- [ ] Owner identity comes exclusively from the token `sub`; client-supplied owner
      IDs are ignored or rejected.
- [ ] Read endpoints allow authenticated `owner` and `viewer` roles; write endpoints
      require `owner`.
- [ ] Missing/invalid credentials return `401`; valid identity without permission
      returns `403`; both use the standard error envelope.
- [ ] JWKS caching and refresh tolerate a temporary Keycloak outage after a valid key
      has been obtained and correctly handle key rotation.
- [ ] `DEFAULT_USER_ID` remains possible only in the explicit local profile and can
      never be activated by a production default.
- [ ] Tests cover owner isolation, viewer restrictions, invalid issuer/audience,
      expiration, and key rotation using a local test issuer rather than live
      Keycloak.

---

## LS-021 — Deploy the ledger service to local Kubernetes

**As a** developer learning operations  
**I want** to run the ledger service in a local Kubernetes cluster  
**So that** its deployment behavior matches the target platform

**Dependencies:** LS-006, LS-018, LS-019

### Acceptance criteria

- [ ] Customize base manifests define a Deployment, ClusterIP Service, ConfigMap,
      Secret references, resource requests/limits, and probes.
- [ ] The pod runs as non-root with a read-only root filesystem, dropped Linux
      capabilities, and no privilege escalation.
- [ ] A pre-rollout migration Job applies schema changes exactly once and blocks the
      rollout on failure; migrations do not run concurrently in every application pod.
- [ ] RollingUpdate uses readiness gating and `maxUnavailable: 0` where cluster
      capacity allows.
- [ ] The local overlay deploys immutable/local image tags and connects to the
      expected PostgreSQL instance without embedding credentials in Git.
- [ ] The Datadog Agent runs as the standard local-cluster DaemonSet (or Datadog
      Operator equivalent), accepts OTLP telemetry, collects container logs and
      Kubernetes metrics, and injects `env`, `service`, and `version` tags.
- [ ] Datadog API and application keys come only from an uncommitted local secret or
      external secret provider; neither key appears in manifests, images, logs, or
      example values.
- [ ] Kong routes `/api/v1/ledger/*`, preserves request/trace headers, and applies the
      agreed CORS, timeout, and rate-limit behavior.
- [ ] A smoke test exercises health, category creation, entry creation, listing, and
      summary through Kong.

---

## LS-022 — Publish immutable images and define a controlled release

**As a** maintainer  
**I want** traceable image publication and deployment controls  
**So that** every running version can be reproduced and rolled back safely

**Dependencies:** LS-005, LS-006, LS-017, LS-021

### Acceptance criteria

- [ ] A successful merge to `main` publishes the affected ledger image to GHCR using
      the commit SHA; deployment manifests never rely on `latest`.
- [ ] A successful deployment sends a Datadog deployment marker containing
      environment, service, version/commit SHA, repository, and deployment result so
      regressions can be correlated with releases.
- [ ] The publish job produces provenance/SBOM metadata and scans the final image for
      known high-severity vulnerabilities according to an explicit initial policy.
- [ ] Package write permission exists only in the publish job and authentication uses
      short-lived platform credentials where available.
- [ ] Deployment is manually triggered through protected GitHub Environments and
      prevents concurrent releases to the same environment.
- [ ] Migration failure stops deployment before application rollout.
- [ ] Rollback documentation distinguishes application rollback from forward-only
      database recovery and describes the expand/contract rule for schema changes.
- [ ] A post-deploy smoke check runs through Kong and automatically marks the release
      failed if core health or read behavior is unavailable.
- [ ] The release verifies that logs, traces, and core metrics for the new service
      version arrive in Datadog within a documented bounded interval.
