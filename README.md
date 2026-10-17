# Advance DB Client

**Web-first database workspace · Go API · TypeScript · Node.js/Next.js**

A clean-room, incremental migration **foundation** inspired by [DBX](https://github.com/t8y2/dbx), not a finished or feature-equivalent port of DBX. This is a distinct codebase: the original Rust + Tauri backend, Vue desktop application, and 100+ engine integrations have **not** been automatically translated.

### Current functionality

- Go 1.23 `database/sql` API: connect/disconnect to PostgreSQL, MySQL, and SQLite; schema explorer; execute queries; bounded CSV export.
- Next.js 15 + React 19 + TypeScript frontend with responsive database workspace, SQL editor, schema sidebar, result table and local query history.
- Node.js server-for-frontend API routes; database credentials and API token remain server-side.
- Password login with signed, HttpOnly, SameSite session cookie; CSRF Origin checks on mutations and simple login throttling.
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

| Engine | Connection | Tables | Query | CSV | Status |
|---|---|---|---|---|---|
| PostgreSQL | Yes | Yes | Yes | Yes | Implemented, live DB untested here |
| MySQL / compatible | Yes | Yes | Yes | Yes | Implemented, live DB untested here |
| SQLite | Yes | Yes | Yes | Yes | Implemented, live DB untested here |
| Oracle, SQL Server, Redis, MongoDB, ClickHouse and other DBX engines | No | No | No | No | Future migration work |

For now, non-SELECT statements are executed as commands and show only affected rows. Read-only mode is selected by default. No schema mutations, backup/restore, AI assistant, SSH tunneling, pagination, or multi-user RBAC are claimed.

## Go HTTP API

`GET /health` is unprotected; all `/v1/*` endpoints require `Authorization: Bearer <API_TOKEN>`:

| Route | Method | Purpose |
|---|---|---|
| `/v1/connections` | GET | List live connections (no secrets) |
| `/v1/connections` | POST | Create connection using `{name,engine,host,port,database,user,password,sslMode,readOnly}` |
| `/v1/connections/{id}` | DELETE | Disconnect |
| `/v1/connections/{id}/schema` | GET | List tables |
| `/v1/connections/{id}/query` | POST | Execute `{sql,timeoutSeconds}` |
| `/v1/connections/{id}/export` | POST | Export up to 1,000 selected rows as CSV |

For SQL statements producing rows the query response has `{columns,rows,rowCount,truncated,durationMs}`. Nonquery responses use `affected`. Query errors are intentionally generic so connection details are not leaked to clients.

## Tests

```sh
(cd backend && go test ./internal/server)
(cd backend && go test ./...)
(cd frontend && npm install && npm run typecheck && npm run build)
```

In an offline environment only the core Go tests can currently be executed without external dependencies. `go test ./...` and the Next.js build **require downloads**; do not treat them as verified until they run in CI.

## Source and license

Upstream reference: `t8y2/dbx`, `main` commit `99eca47cc33b7a3a71fd57460e61dba2d30f7504` (2026-10-17). Upstream license: Apache-2.0. This project includes the Apache-2.0 license and attributes DBX in `NOTICE`. This new implementation is not associated with, endorsed by, or a distribution of the upstream DBX binaries.

Read [`docs/MIGRATION.md`](docs/MIGRATION.md) for the exact scope and the remaining conversion milestones. See [`docs/SECURITY.md`](docs/SECURITY.md) for deployment constraints.

## Publishing on GitHub

This workspace is ready to initialize as `raam9143/advance_db-client`. If an authenticated GitHub CLI is installed with permission to create repositories:

```sh
bash scripts/publish.sh
```

The script **creates a new private repository** under `raam9143`, makes a local commit, and pushes `main`. It refuses to overwrite an existing local Git repository or remote repository. Review the code before publishing.

### Enhanced database inspection (migration in progress)

Select a table to inspect column types, primary keys, indexes, and preview rows using bounded server-side pagination (250 maximum per request). The database API has two new GET routes: `/v1/connections/{id}/describe?schema=&table=` and `/v1/connections/{id}/table?schema=&table=&limit=50&offset=0`.

**New protocol aliases:** MariaDB uses the MySQL protocol; CockroachDB uses the PostgreSQL protocol. Vendor-specific compatibility and live tests are still outstanding.

**Write safety:** SQL writes are *disabled* unless the backend environment sets `ALLOW_WRITE_QUERIES=true`, and each write requires a connection opened without the read-only flag. SQL text is not logged; SHA-256 statement fingerprints and success/failure are logged. This logging is not a persistent audit trail.

**Tests:** `cd backend && go test ./internal/server` runs dependency-free unit tests. `go test -race ./...` runs driver-backed SQLite tests; the PostgreSQL and MySQL integration cases require disposable test databases selected by `TEST_POSTGRES_HOST` and `TEST_MYSQL_HOST`, provisioned by CI. Run `cd frontend && npm install && npm run typecheck && npm run build` to verify the Next.js build.
