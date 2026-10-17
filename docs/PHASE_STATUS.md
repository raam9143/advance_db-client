# Advance DB Client — migration status (17 October 2026)

This is a web-first rewrite foundation, **not yet a complete port** of t8y2/dbx.

| Phase | Status | Remaining work |
| --- | --- | --- |
| 1 SQL security | In progress | Full SQL parser, per-user RBAC, immutable write audit, encrypted credentials and broader engine/version integration coverage |
| 2 Database support | In progress | PostgreSQL, MySQL, SQLite baseline; MariaDB/CockroachDB protocol aliases only. ClickHouse read-only HTTPS adapter with protocol mock tests; Oracle, SQL Server, Mongo, Redis and most upstream drivers remain unported |
| 3 Advanced SQL workspace | In progress | Bind parameters, EXPLAIN, browser-local saved queries implemented. Transactions, multi-tab editing, autocomplete, cancellation and Playwright E2E pending |
| 4 Data movement | In progress | Bounded CSV/JSON export; streamed import, full backup, restore, migration/diff pending |
| 5 SSH/AI/MCP/CLI | In progress | SSH tunnels, cloud integrations, AI permissions and plugins; read-only CLI/MCP initial implementations added |
| 6 Production release | Pending | OIDC/SSO, MFA, account isolation, observability, SBOM, frontend dependency locking, disaster recovery validation |

## Test evidence
- **GitHub Actions is green:** [run 39028921850](https://github.com/raam9143/advance_db-client/actions/runs/39028921850) on commit `b2ce844`, after repository visibility changed to public.
- CI passed all four jobs: Go full `go test -race ./...` with PostgreSQL 16/MySQL 8.4 service containers, `go vet ./...`, `go build ./cmd/api`, frontend typecheck/production build, read-only ClickHouse protocol tests, and Node.js MCP tests.
- `backend/go.mod` and `backend/go.sum` are committed, with `go mod verify` and `go mod tidy -diff` enforcing reproducibility.
- This CI run is not proof of production readiness or successful connectivity to real ClickHouse, Oracle, SQL Server, Redis or MongoDB instances.
- Previous runner startup failures stopped after the repository became public and jobs were rerun. The precise prior account/runner restriction was not independently established.

## Important controls
- Writes require `ALLOW_WRITE_QUERIES=true`, a writable DB connection, and per-request `confirmWrite:true`.
- Database-level read-only roles are required; the built-in SQL keyword check is conservative, not a complete SQL parser.
- Exports are capped at 1,000 rows and can be truncated.
- PostgreSQL uses `$1` placeholders; MySQL and SQLite use `?`. Query parameters accept scalars only.
- Saved SQL and recent queries are browser-local and may contain sensitive SQL. Never store secrets in query text.

## Browser session isolation
New login cookies include a random nonce. The Next.js proxy sends HMAC-proven, SHA-256 scoped identities to the Go API, partitioning in-memory connections by login session. Direct bearer-API and CLI/MCP requests use a separate service namespace. This does not implement per-user RBAC, admin role separation, durable credential isolation, or enterprise SSO. Never expose the Go API token to untrusted clients.

## Dry-run schema difference
Authenticated `POST /v1/schema/diff` accepts two table metadata snapshots in `{"before":...,"after":...}` format returned by the `describe` endpoint. It reports deterministic column-level differences with destructive flags and `executable:false`. No SQL is generated or applied. It is not a tested production migration engine.

## ClickHouse read-only HTTPS adapter
Verified TLS 1.2+, Basic authentication, readonly=2, bounded JSON responses, named ClickHouse parameter placeholders ({p1:String}), table and column browsing. No writes, SQL transactions, native protocol, EXPLAIN, or independently validated live database behavior. Use least-privilege credentials.

## Opt-in SQLite snapshot download
An authenticated SQLite connection can create a database snapshot with `POST /v1/connections/{id}/backup` and `{"confirmBackup":true}`. Requires a dedicated absolute, owner-private `SQLITE_BACKUP_DIR` (0700). SQLite `VACUUM INTO` writes the snapshot in a generated private directory. Maximum 128 MiB; ten outstanding snapshots globally; session-isolated IDs expire after 30 minutes. Download with `GET /v1/backups/{id}` and delete with `DELETE /v1/backups/{id}`. **No restore endpoint exists**, and this is not a full disaster-recovery solution. Snapshots are disabled by default.
