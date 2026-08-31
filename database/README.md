# Local Kubernetes resources

Local infrastructure is deployed to a developer-owned Kubernetes cluster. The
reference workflow uses `kind`, but any cluster with a default `StorageClass` works.

## Ledger PostgreSQL

The local overlay creates a single-replica PostgreSQL `StatefulSet`, a `ClusterIP`
service, and a persistent volume claim. The credentials in
`overlays/local/database/ledger-postgres/credentials.env` are deliberately insecure
development defaults. Replace them before the first deployment when different local
values are required. Never use them in a shared or production environment.

Create or start PostgreSQL and wait until it is ready:

```powershell
kubectl apply -k database/overlays/local
kubectl rollout status statefulset/ledger-postgres --timeout=90s
```

Verify a usable connection with the least-privilege application account:

```powershell
kubectl exec statefulset/ledger-postgres -- sh -ec 'PGPASSWORD="$LEDGER_APP_PASSWORD" psql --username="$LEDGER_APP_USER" --dbname=ledger_db --command="select 1"'
```

Stop PostgreSQL without deleting its volume:

```powershell
kubectl scale statefulset/ledger-postgres --replicas=0
```

Start it again and wait for readiness:

```powershell
kubectl scale statefulset/ledger-postgres --replicas=1
kubectl rollout status statefulset/ledger-postgres --timeout=90s
```

To connect from a ledger service running outside the cluster, keep this command
running in a separate terminal:

```powershell
kubectl port-forward service/ledger-postgres 5432:5432
```

The matching development connection URL is documented in
`ledger-service/.env.example`. A ledger service running inside the same namespace
uses `ledger-postgres:5432` instead of `localhost:5432`.

Deleting the local database is intentionally separate and destructive. This is the
only routine command here that removes the stored data:

```powershell
kubectl delete statefulset ledger-postgres
kubectl delete persistentvolumeclaim data-ledger-postgres-0
```

PostgreSQL initialization scripts only run for an empty volume. Changing the
bootstrap credentials after the database has already been initialized therefore
requires intentionally deleting the local persistent volume claim and applying the
overlay again.
