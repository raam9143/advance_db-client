# DBX → Advance DB Client: staged migration plan

## Upstream baseline

Reference: https://github.com/t8y2/dbx at `99eca47cc33b7a3a71fd57460e61dba2d30f7504` (2026-10-17).
Upstream DBX is a sizeable Rust workspace with Vue + TypeScript + Vite desktop UI, Tauri shell, Rust web server, CLI and MCP. `crates/ARCHITECTURE.md` describes its separation of application orchestration (`dbx-core`), database adapters (`dbx-drivers`), SQL dialects (`dbx-sql`), types (`dbx-types`), formats, AI integration, plugins, and platform services. It advertises 100+ engines; this starter DOES NOT implement them all.

## Target mapping

| DBX subsystem | New target module | Current disposition |
|---|---|---|
| Rust `dbx-core/connection` | Go `backend/internal/server` | Basic session lifecycle only |
| Rust `dbx-drivers` | Go `database/sql` + driver imports | PostgreSQL, MySQL, SQLite only |
| Rust `dbx-core/query` | Go HTTP query handler | Execute, timeout, row cap; no paging/transactions UI |
| Rust `dbx-core/schema` | Go schema inspection | Table names only |
| Rust `dbx-formats` | Go `encoding/csv` | CSV export (1,000-row cap) |
| Desktop `apps/desktop/src` (Vue) | Next.js React TypeScript | New responsive web-first UI, not component-level rewrite |
| `dbx-web` web service | Next Node BFF + Go service | Basic auth, API proxy, connection sessions |
| `dbx-sql`, dialect metadata | Go SQL guard, TypeScript editor | Limited; no real dialect parser / formatting |
| `dbx-core/persistence` | Browser localStorage (SQL history) | History only; no encrypted server storage |
| AI & agent workflows | Future Go/Node adapters | Not implemented |
| Tauri, CLI, MCP, agents, plugins, tunneling | Future modular services | Not implemented |
| Import/export, migrations, backups | Go jobs + workers (planned) | Not implemented |

## Prioritized engineering phases

1. **Complete the verified SQL vertical slice:** Postgres/MySQL/SQLite integration tests against test containers; database credentials vault, schema columns/indexes/constraints, pagination, strict SQL parser, audited writes and role-based security.
2. **Add database adapters:** Oracle, ClickHouse, SQL Server, MariaDB, CockroachDB, Redis, MongoDB; preserve upstream dialect semantics and use protocol-specific services for non-SQL engines.
3. **Full query experience:** tabs, syntax highlighting, autocomplete, parameter binding, explain plans, cancel requests, transactions, charting, saved connections and result virtualization.
4. **Data operations:** import/export streams, encrypted backups, restore, migrations, comparison and workflow jobs with approvals and progress.
5. **Parity services:** SSH tunnels, cloud connectors, AI assistant, MCP, CLI, plugins, schedules and optional desktop distribution.
6. **Production hardening:** multi-user OIDC/MFA, RBAC, persistent encrypted storage, audit trails, per-role quotas, SSO, CSRF/trusted proxy validation, TLS, integration tests, metrics, benchmarking, release automation.

## Known limitations / non-goals of this starter

- This is a new implementation inspired by the public source: **not** mechanically translated line-by-line and **not feature parity** with DBX.
- Database drivers require dependency downloads; no live database connection tests could be run in the environment used to create this bundle.
- UI dependencies require npm installation; TypeScript/Next build was not run during offline authoring.
- Browser history resides in localStorage, contains SQL text, and is not encrypted; do not paste passwords or PII into query text.
- The backend stores live `*sql.DB` handles in Go memory only; restart loses sessions. The UI has no persistent credential store.
- SQL read-only detection is heuristic and not a substitute for database privileges, even with read-only transactions.
- Backend SQLite file opening requires a pre-existing file within SQLITE_ALLOWED_DIR. Connections cannot access arbitrary filesystem paths.
- CSV export intentionally uses the same bounded 1,000-row query result and should not be used as a full database dump.
- Single administrative login only: not suitable for multi-user shared production without further security work.

## Acceptance criteria before declaring full conversion

Ported features must be mapped 1:1 to upstream contract tests; database combinations tested; UI workflows screenshotted and compared; desktop/CLI/MCP accounted for; security reviewed; integration CI green; license notices preserved. Do not mark any unsupported driver or feature as migrated.
