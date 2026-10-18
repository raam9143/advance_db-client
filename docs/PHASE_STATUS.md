# Advance DB Client — migration status (17 October 2026)

This is a web-first rewrite foundation, **not yet a complete port** of t8y2/dbx.

| Phase | Status | Verified functionality and remaining blockers |
| --- | --- | --- |
| 1 SQL security | In progress | Session namespaces, OIDC admin/viewer, encrypted operator vault, guarded writes and a stricter lexical read-only guard. Still need dialect-aware parser semantics, reviewed dual-approval writes, per-resource RBAC, off-host immutable audit and expert security review. |
| 2 Database adapters | In progress | Live PostgreSQL/MySQL/SQLite/Redis/MongoDB/SQL Server CI; ClickHouse 25.3 real-query and typed-parameter tests through test-only verified HTTPS proxy. Oracle TCPS real integration, vendor-specific MariaDB/Cockroach compatibility and advanced cluster/Atlas behavior remain pending. |
| 3 SQL workspace | In progress | Bind parameters, EXPLAIN, cancellation, in-memory multi-tabs, browser-local explicit saves, CSV/JSON downloads and authenticated Playwright E2E. Still need server-encrypted per-user saved queries, syntax completion, transaction controls, virtualization and advanced charts. |
| 4 Data movement | In progress | Bounded CSV/JSON/MongoDB page export, read-only CSV streamed via Go/Next.js capped at 50k rows/32MiB, private SQLite snapshot and offline rollback-preserving restore, non-executable schema diff. Still need per-engine full backup/restore, imports, approval workflow, progress/resume, DR validation. |
| 5 SSH/cloud, AI, MCP | In progress | Host-key-pinned SSH-local Postgres/MySQL, consent-gated AI SQL review (mock-tested), read-only MCP and CLI. Remaining: cloud connectors, vendor tunnels, external AI provider acceptance, plugin sandbox/signatures, jobs/schedules. |
| 6 Production security and release | In progress | OIDC/PKCE + MFA evidence, Redis sessions/OTP replay, admin/viewer roles, TOTP option, AES-GCM vault rotation, local audit integrity + exclusive lock, production policy, explicit DB egress allowlist, Prometheus signals, SBOM/Trivy and release qualifications. Still need real IdP/live MFA/offboarding, off-host immutable audit, KMS/multi-replica rotation, branch protection, operating alert delivery, penetration review and demonstrated DR/HA. |

## Test evidence
- **Latest fully successful baseline:** [run 39054426876](https://github.com/raam9143/advance_db-client/actions/runs/39054426876) after SSH, local audit, SQLite restore, and TOTP JavaScript syntax correction. Later AI review and UI feature pushes must also pass CI before being called verified.
- CI passed the five-job workflow: Go full `go test -race ./...` with PostgreSQL 16/MySQL 8.4/MongoDB 7.0/Redis 7.4 containers plus separate SQL Server 2022 job, `go vet ./...`, `go build ./cmd/api`, frontend typecheck/production build, read-only ClickHouse protocol tests, and Node.js MCP tests.
- `backend/go.mod` and `backend/go.sum` are committed, with `go mod verify` and `go mod tidy -diff` enforcing reproducibility.
- This CI run is not proof of production readiness or successful connectivity to real ClickHouse or Oracle TCPS instances; Azure/Atlas/replica-set/HA compatibility remains unverified.
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

## Query cancellation
The web SQL editor now supports a Cancel query button. The browser abort signal is forwarded by the Next.js BFF to the Go API, which already executes SQL within a bounded, cancellable request context. Cancellation is best-effort and vendor/driver-dependent: a database may finish or commit a write before cancellation. This is **not** explicit transaction rollback, and writes require separate confirmation as before.

## SQL Server and Oracle (TCPS) adapters — under validation
New `sqlserver` and `oracle` engines use Microsoft go-mssqldb v1.8.2 (Go 1.23-compatible) and go-ora/v2 v2.9.0 (pure Go; no Instant Client). Both enforce encrypted, verified TLS connection strings by default and support table enumeration, column/primary-key/index metadata, dialect-specific paging, and scalar bind parameters. Oracle supports service name or SID (via `serviceMode`). SQL Server and Oracle EXPLAIN is deliberately unavailable until a vetted, engine-specific implementation exists. Read-only checks must be enforced by restricted database logins; driver-specific read-only transactions may not be supported. No Oracle/SQL Server live database compatibility test has been run yet.

## SQL Server and Oracle implementation evidence
- SQL Server 2022 live integration **passed** in [GitHub Actions run 39041968918](https://github.com/raam9143/advance_db-client/actions/runs/39041968918). Verified: connection setup (TLS-encrypted loopback CI exception), schema, primary-key metadata, table paging, bind parameters, create/insert/drop on disposable test data. Default remote TLS certificate verification remains enforced.
- Oracle go-ora/v2 driver, service/SID handling, TCPS certificate enforcement, metadata and 12c+ pagination; compile/parameter tests pass. Real Oracle TCPS test is opt-in and not part of public CI. See [enterprise driver notes](SQLSERVER_ORACLE.md).

## MongoDB read-only milestone
Implemented standalone MongoDB Go v1 driver (v1.17.6), schema/collection browsing, bounded Extended JSON document pagination and optional scalar equality filters. The API blocks operators, aggregations, mutations and SQL endpoint misuse. TLS verification is mandatory remotely, with opt-in loopback-only plaintext for disposable local CI. Browser UI and local MCP stdio read tools added. Go CI provisions a MongoDB 7.0 service and runs an explicit live `TestMongoLiveReadonly`. Atlas/replica set, writes, import/export, change streams and MongoDB backup/restore remain out of scope. See [MongoDB integration](MONGODB.md).

## Consent-gated AI SQL review
Opt-in external OpenAI Responses API review added. SQL single-statement checks and string-literal redaction precede outbound requests; no rows/credentials forwarded, no SQL execution, and no generated query can be applied automatically. Mock HTTP tests are in CI. Requires a live API key and schema-sharing approval for production validation. See [AI review](AI_REVIEW.md).

## Phase 6 hardening — 18 October 2026
- Found and eliminated frontend runtime vulnerability findings by replacing npm-bearing runtime with the non-root, Debian 13 distroless Node image. The resulting baseline [security workflow 39089048555](https://github.com/raam9143/advance_db-client/actions/runs/39089048555) passed npm audit, Go vulnerabilities, SBOM and both container-image scans. Later vault and telemetry commits require their own new scan results before acceptance.
- Added encrypted operator-managed AES-256-GCM profile store and offline key rotation (`backend/cmd/adb-vault`), plus admin-only read-only `credentialRef` connection resolution. Tests cover missing/incorrect keys, record tampering, permission checks and read-only/vs-viewer roles. See [Vault](CREDENTIAL_VAULT.md).
- Added shared Redis local-admin lockout and credential-store fail-closed errors, instead of trusting spoofable `X-Forwarded-For` headers for throttling. Production requires `SESSION_REDIS_REQUIRED=true` and externally managed TLS Redis.
- Added restricted Prometheus exposition and operator alert rules; no distributed tracing or real alert delivery exercise has been completed. See [Observability](OBSERVABILITY.md).
- Remaining Phase 6 blockers: production IdP authentication/MFA offboarding tests, externally reviewed RBAC/threat model and credential custody, distributed load/failover tests, immutable/off-host audit ingestion, secrets rotation across replicas, branch protections/review requirements, real alerting and DR validation. **Do not close Phase 6 solely on green CI.**

## Production qualification safeguards — 18 October 2026
Added opt-in fail-closed `ADB_PRODUCTION=true` runtime profile for backend and frontend, including mandatory verified-TLS sessions/SSO/MFA `amr`, distinct IdP groups, strict public Origin and API key strength, audit + vault startup checks, and prohibition on development plaintext/write exceptions. Added audit file exclusive writer locking and health fail-closed after logging failures. Added non-root read-only Compose reference, `production-qualification-evidence` manual workflow with exact-commit CI/review ruleset/evidence checks and test suite. See [production guide](PRODUCTION_DEPLOYMENT.md). **Independent proof is still required** for external IdP, HSM/KMS, durable off-host audits, live alerts, load/DR/rollback and review-enforcing GitHub rules. Issue #6 remains open until independently accepted.

## 18 October implementation increments
- **SQL read-only parser:** added conservative token scanner rejecting CTE mutations, dollar-quoted SQL, stacked statements, side-effecting functions, dangerous metadata/sequence constructs; SQLite EXPLAIN QUERY PLAN retained. It is intentionally **not** a complete vendor SQL grammar and does not replace DB permissions.
- **ClickHouse:** real ClickHouse 25.3 service is now used in CI through a test-only, certificate-validated HTTPS reverse proxy; parameter queries and metadata tested. This is not a native-server TLS production certificate test.
- **Production egress:** `DB_ENDPOINT_ALLOWLIST` required in `ADB_PRODUCTION=true`, verified before connection creation; exact host/port allowlist denies wildcards and link-local endpoints. Production VPC/firewall controls remain required to prevent DNS rebinding/network pivots.
- **Workspace tabs:** up to eight accessible, independently editable in-memory SQL tabs with browser E2E coverage; tabs deliberately do not persist automatically.
- **Streamed exports:** new read-only `POST /v1/connections/{id}/stream` CSV endpoint and web button, server-side limits 50,000 rows/32 MiB/60s, formula neutralization, CSV integration and browser proxy tests. Browser Blob download may still allocate the full 32 MiB.
- **Unfinished issues:** #1–#6 remain open; the current code increments do not satisfy all original acceptance criteria or confer production certification.
