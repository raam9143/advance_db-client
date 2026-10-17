# Advance DB Client — migration status (17 October 2026)

This is a web-first rewrite foundation, **not yet a complete port** of t8y2/dbx.

| Phase | Status | Remaining work |
| --- | --- | --- |
| 1 SQL security | In progress | Full parser, integration CI, per-user RBAC, immutable write audit, encrypted credentials |
| 2 Database support | In progress | PostgreSQL, MySQL, SQLite baseline; MariaDB/CockroachDB protocol aliases only. Oracle, SQL Server, ClickHouse, Mongo, Redis and 100+ other drivers not ported |
| 3 Advanced SQL workspace | In progress | Bind parameters, EXPLAIN, browser-local saved queries implemented. Transactions, multi-tab editing, autocomplete, cancellation and Playwright E2E pending |
| 4 Data movement | In progress | Bounded CSV/JSON export; streamed import, full backup, restore, migration/diff pending |
| 5 SSH/AI/MCP/CLI | In progress | SSH tunnels, cloud integrations, AI permissions and plugins; read-only CLI/MCP initial implementations added |
| 6 Production release | Pending | OIDC/SSO, MFA, account isolation, observability, SBOM, stable dependencies, disaster recovery validation |

## Test evidence
- Locally passed: `cd backend && go test -race ./internal/server && go vet ./internal/server`.
- Full `go test ./...` requires driver dependencies unavailable in the offline execution environment; the Next.js build also requires package installation.
- GitHub Actions run 39015371552 failed **before steps started** and has no job logs. The root cause remains unconfirmed.

## Important controls
- Writes require `ALLOW_WRITE_QUERIES=true`, a writable DB connection, and per-request `confirmWrite:true`.
- Database-level read-only roles are required; the built-in SQL keyword check is conservative, not a complete SQL parser.
- Exports are capped at 1,000 rows and can be truncated.
- PostgreSQL uses `$1` placeholders; MySQL and SQLite use `?`. Query parameters accept scalars only.
- Saved SQL and recent queries are browser-local and may contain sensitive SQL. Never store secrets in query text.
