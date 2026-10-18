# Advance DB Client

**Web-first database workspace · Go API · TypeScript · Node.js/Next.js**

A clean-room, incremental migration **foundation** inspired by [DBX](https://github.com/t8y2/dbx), not a finished or feature-equivalent port of DBX. This is a distinct codebase: the original Rust + Tauri backend, Vue desktop application, and 100+ engine integrations have **not** been automatically translated.

### Current functionality
- Opt-in SSH-local, pinned-host-key database tunnels for PostgreSQL/MySQL protocols, with bastion/destination allowlists; see [SSH tunnel guide](docs/SSH_TUNNEL.md).
- Local keyed integrity-chain API audit logging (opt-in, single-process); see [security limitations](docs/SECURITY.md).
- Offline, operator-run SQLite restoration from validated snapshots with retained originals; see [SQLite restore](docs/SQLITE_RESTORE.md).
- Opt-in external AI SQL review with explicit consent and literal redaction; advice is non-executable; see [AI review](docs/AI_REVIEW.md).


- Go 1.23 `database/sql` API: connect/disconnect to PostgreSQL, MySQL, and SQLite; schema explorer; execute queries; bounded CSV export.
- Next.js 15 + React 19 + TypeScript frontend with responsive database workspace, SQL editor, schema sidebar, result table and local query history.
- Node.js server-for-frontend API routes; database credentials and API token remain server-side.
- Single-admin password login with signed, HttpOnly, SameSite cookie; opt-in TOTP authenticator code; CSRF Origin checks and local login throttling. Not enterprise SSO/RBAC.
- Time-limited in-memory connection handles (2h idle); 30s query time limit, 1,000-row response cap, request-size limits.
- Conservative SQL detection and driver-level read-only transactions for query statements. **Use database read-only roles for actual enforcement.**
- Go unit tests for authentication, statement guard, and DSN handling.

### Architecture

```text
Browser / React + TypeScript / Next.js App Router
        | browser cookies (HttpOnly, signed)
        v
Next.js Node API proxy (authenticated, same-origin)
        | Bearer API_TOKEN (never exposed to browser)
        v
Go net/http API (127.0.0.1:8080, or private Docker network)
        | database/sql + drivers
        +--> PostgreSQL (pgx)
        +--> MySQL (go-sql-driver)
        +--> SQLite (modernc.org/sqlite)
```

## Local development

Requirements: Go 1.23+, Node.js 22+, npm, **network access for initial dependency download**, and a database server if testing a remote engine.

```sh
cp .env.example .env
# Edit with STRONG, UNIQUE values for APP_PASSWORD, APP_SESSION_SECRET, API_TOKEN.
# Use the same API_TOKEN for both services; do not commit .env.

# terminal 1
cd backend
API_TOKEN='your-api-token' SQLITE_ALLOWED_DIR="$PWD/../data" go run ./cmd/api

# terminal 2
cd frontend
npm install
APP_PASSWORD='your-strong-access-password' \
APP_SESSION_SECRET='at-least-32-random-characters-change-me' \
API_TOKEN='your-api-token' \
GO_API_URL=http://127.0.0.1:8080 npm run dev
```

Open `http://localhost:3000`. Example passwords above are placeholders; create your own (`openssl rand -hex 32`). For SQLite files, create `data/` and place an existing `.db` file there. SQLite `:memory:` may be used without a data directory, but disappears after disconnection/restart. The `SQLITE_ALLOWED_DIR` requirement prevents opening arbitrary host file paths. In production, put Next.js behind HTTPS and use a private network between Next.js and Go.

### Docker Compose

```sh
cp .env.example .env
# edit .env secrets
mkdir -p data
docker compose up --build -d
# http://localhost:3000  (localhost testing only; use TLS for public access)
```

The Go API is internal to the Compose network, not published on a host port. The only exposed service is the Next.js web interface. See `docs/SECURITY.md` before production use.

## Supported engines

| Engine | Status | Verification |
|---|---|---|
| PostgreSQL 16 | SQL queries, schema, table paging and exports | Live CI integration |
| MySQL 8.4 | SQL queries, schema, table paging and exports | Live CI integration |
| SQLite | SQL queries, schema, table paging, exports and opt-in private snapshot | Live CI integration |
| MariaDB | MySQL protocol adapter | Dedicated vendor tests pending |
| CockroachDB | PostgreSQL protocol adapter | Dedicated vendor tests pending |
| ClickHouse | Verified HTTPS, bounded read-only SQL and metadata | Protocol mock tests; live server pending |
| Redis 7.4 | Separate read-only RESP2 key inspector; SCAN, GET, TYPE, TTL, HGETALL, LRANGE | Live CI integration |
| Microsoft SQL Server 2022 | SQL, metadata, bind parameters, dialect pagination | Live integration CI passed |
| Oracle (TCPS) | SQL, metadata, Service Name/SID, bind parameters, Oracle paging | Driver/build tests passed; optional live Oracle test not run |
| MongoDB 7.0 | Read-only collection listing, bounded documents, scalar equality filter | CI integration; MongoDB Atlas/replica-set compatibility pending |
| Remaining DBX engines | Not implemented | Pending |

MongoDB similarly uses a separate document-inspection API, not SQL emulation. Bounded document queries are available through the web UI and read-only MCP tools. See [MongoDB details](docs/MONGODB.md). Remote MongoDB connections require validated TLS; local plaintext requires an explicitly enabled loopback-only development setting.

Redis does **not** use SQL emulation or enable modifying commands. By default, Redis requires TLS; loopback plaintext requires an explicit server opt-in (`REDIS_ALLOW_PLAINTEXT_LOOPBACK=true`).

SQL editor capabilities include bind parameters, EXPLAIN, saved browser-local SQL, JSON/CSV downloads and best-effort cancellation. SQL writes require the operator to enable the server policy and confirm each statement. Read-only accounts and a production identity system are still required.

SQLite snapshot creation is **disabled by default**. To enable, configure an absolute, owner-private (0700) `SQLITE_BACKUP_DIR`; the server creates a bounded, temporary, session-owned backup using SQLite VACUUM INTO. Maximum 128 MiB, ten outstanding backups, 30-minute expiry. Restores are not implemented. See [Redis integration](docs/REDIS_INTEGRATION.md), [phase status](docs/PHASE_STATUS.md) and [security notes](docs/SECURITY.md).

## Go HTTP API

`GET /health` is unprotected; all `/v1/*` endpoints require `Authorization: Bearer <API_TOKEN>`:

| Route | Method | Purpose |
|---|---|---|
| `/v1/connections` | GET | List live connections (no secrets) |
| `/v1/connections` | POST | Create connection using `{name,engine,host,port,database,user,password,sslMode,readOnly}` |
| `/v1/connections/{id}` | DELETE | Disconnect |
| `/v1/connections/{id}/schema` | GET | List tables |
| `/v1/connections/{id}/query` | POST | Execute `{sql,parameters,timeoutSeconds,confirmWrite}` |
| `/v1/connections/{id}/export?format=csv\|json` | POST | Export up to 1,000 selected rows |\n| `/v1/connections/{id}/redis` | POST | Redis read-only inspector commands |\n| `/v1/connections/{id}/backup` | POST | Opt-in SQLite snapshot with explicit confirmation |\n| `/v1/backups/{id}` | GET / DELETE | Download or remove a session-owned SQLite snapshot |

For SQL Server and Oracle deployment prerequisites, TLS requirements, and optional Oracle integration-test configuration, see [Enterprise driver notes](docs/SQLSERVER_ORACLE.md).

For SQL statements producing rows the query response has `{columns,rows,rowCount,truncated,durationMs}`. Nonquery responses use `affected`. Query errors are intentionally generic so connection details are not leaked to clients.

## Tests

```sh
(cd backend && go test ./internal/server)
(cd backend && go test ./...)
(cd frontend && npm install && npm run typecheck && npm run build)
```

GitHub Actions now runs Go race/integration tests against PostgreSQL 16, MySQL 8.4, Redis 7.4, MongoDB 7.0 and Microsoft SQL Server 2022 service containers, frontend typecheck/build, MCP tests and ClickHouse protocol mocks. See the latest [CI workflow](https://github.com/raam9143/advance_db-client/actions/workflows/ci.yml). Live Oracle, SQL Server, MongoDB, and real ClickHouse validation remain pending.

## Source and license

Upstream reference: `t8y2/dbx`, `main` commit `99eca47cc33b7a3a71fd57460e61dba2d30f7504` (2026-10-17). Upstream license: Apache-2.0. This project includes the Apache-2.0 license and attributes DBX in `NOTICE`. This new implementation is not associated with, endorsed by, or a distribution of the upstream DBX binaries.

Read [`docs/MIGRATION.md`](docs/MIGRATION.md) for the exact scope and the remaining conversion milestones. See [`docs/SECURITY.md`](docs/SECURITY.md) for deployment constraints.

## GitHub repository

Source, issues and CI: [raam9143/advance_db-client](https://github.com/raam9143/advance_db-client). The repository is public, with `main` as the default branch. Releases are not yet production-qualified.

### Enhanced database inspection (migration in progress)

Select a table to inspect column types, primary keys, indexes, and preview rows using bounded server-side pagination (250 maximum per request). The database API has two new GET routes: `/v1/connections/{id}/describe?schema=&table=` and `/v1/connections/{id}/table?schema=&table=&limit=50&offset=0`.

**New protocol aliases:** MariaDB uses the MySQL protocol; CockroachDB uses the PostgreSQL protocol. Vendor-specific compatibility and live tests are still outstanding.

**Write safety:** SQL writes are *disabled* unless the backend environment sets `ALLOW_WRITE_QUERIES=true`, and each write requires a connection opened without the read-only flag. SQL text is not logged; SHA-256 statement fingerprints and success/failure are logged. This logging is not a persistent audit trail.

**Tests:** `cd backend && go test ./internal/server` runs dependency-free unit tests. `go test -race ./...` runs driver-backed SQLite tests; the PostgreSQL and MySQL integration cases require disposable test databases selected by `TEST_POSTGRES_HOST` and `TEST_MYSQL_HOST`, provisioned by CI. Run `cd frontend && npm install && npm run typecheck && npm run build` to verify the Next.js build.

### Phase 6 operator resources
- [Encrypted credentials and key rotation](docs/CREDENTIAL_VAULT.md)
- [Prometheus metrics and SLO alert rules](docs/OBSERVABILITY.md)
- [Enterprise release and recovery runbook](docs/PHASE6_RUNBOOK.md)

### Locked-down production profile
See [Production Deployment and Qualification](docs/PRODUCTION_DEPLOYMENT.md). `ADB_PRODUCTION=true` enables fail-closed Go/Next runtime policy checks and the [single-node production Compose example](deploy/compose.production.yml). The [manual production qualification workflow](.github/workflows/production-qualification.yml) requires independent evidence, active branch rules and successful exact-commit quality/security/release checks; it does **not** award a security certificate.
