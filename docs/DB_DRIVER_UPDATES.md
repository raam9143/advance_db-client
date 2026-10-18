# Database driver auto-update and forward compatibility

Database support is **tested compatibility, not a promise to support versions that have not shipped**. This project proposes dependency updates automatically, but it never silently updates a production instance, database schema, or driver at runtime.

## Automated dependency updates

`.github/dependabot.yml` checks `backend/go.mod` every day (04:00 UTC), and opens individually reviewable pull requests for Go dependencies, including:

| Engine | Implementation | Update route |
| --- | --- | --- |
| PostgreSQL / CockroachDB | `github.com/jackc/pgx/v5` | Go module PR; new major module paths need separate migration work |
| MySQL / MariaDB | `github.com/go-sql-driver/mysql` | Go module PR |
| Microsoft SQL Server | `github.com/microsoft/go-mssqldb` | Go module PR |
| Oracle (TCPS) | `github.com/sijms/go-ora/v2` | Go module PR |
| MongoDB | `go.mongodb.org/mongo-driver` | Go module PR; future major module paths need manual code adaptation |
| SQLite | `modernc.org/sqlite` | Go module PR |
| Redis | Bounded built-in RESP implementation | No Go Redis driver to update; tested against new Redis servers |
| ClickHouse | Bounded built-in HTTPS implementation | No Go ClickHouse driver to update; tested against new ClickHouse servers |

The frontend npm lockfile, backend/frontend Dockerfiles and GitHub Actions are also checked weekly. `go.mod`, `go.sum`, and the frontend lockfile remain reproducible and must be committed by the update PR.

**No auto-merge** is configured. Major releases, driver module path migrations, new database authentication defaults, TLS and protocol changes all require a deliberate compatibility assessment. Apply `go mod tidy`, read vendor release notes and migration advisories, and review security findings before approving any PR. No runtime driver download or execution is allowed.

## Weekly rolling-server compatibility canary

`.github/workflows/database-forward-compatibility.yml` runs each Sunday at **03:17 UTC**, on manual dispatch, and for pull requests changing driver/module or integration-test files. All tests run against disposable CI instances listening only on the runner's loopback interface.

| Engine | Rolling image | Live coverage |
| --- | --- | --- |
| PostgreSQL | `postgres:latest` | Connect, schema, PK/index metadata, SQL, bind values, pagination, export |
| MySQL | `mysql:latest` | Same SQL foundation |
| MariaDB | `mariadb:lts` | Live alias compatibility and SQL foundation |
| CockroachDB | `cockroachdb/cockroach:latest` | Live alias compatibility and SQL foundation |
| MongoDB | `mongo:latest` | Read-only collection and document operations |
| Redis | `redis:latest` | Read-only SCAN, write rejection |
| ClickHouse | `clickhouse/clickhouse-server:latest` | Parameterized reads behind a verified test TLS proxy |
| SQL Server | `mcr.microsoft.com/mssql/server:2025-latest` | Connect, schema, bind parameters, pagination, write protection |

These tags move without source changes. The canary records the pulled image digest in Actions logs and fails visibly if new server behavior breaks a supported operation. It runs with disposable CI-only credentials, uses no corporate databases, and **does not deploy image tags to production**.

The normal `quality-gates` workflow continues to test pinned stable database versions. A new rolling-image failure is a compatibility finding, not evidence that an existing production version has failed. Never disable security tests or weaken TLS to clear a failing matrix job.

## Update acceptance checklist

1. Dependabot proposes a Go driver version update. Review the changelog, license, transitive dependencies, Go toolchain requirements and CVE/security advisories.
2. The pull request must pass **all** existing quality, security, live integration, browser E2E and database-forward-compatibility jobs. Any test skips must be investigated.
3. For changes to drivers, protocol defaults or database major versions, run approved read-only tests in a staging environment with representative TLS, authentication, permissions, schema shapes and server versions.
4. Get review approval, merge under branch protection, and release the pinned binary/container artifact with a rollback route. Record verified database/driver pairs in release notes.
5. If a new release breaks compatibility, leave the canary red and create a tracked issue with the server version/image digest, failure logs and required adapter changes. Do not automatically claim support.

## Limits and follow-up acceptance

- Oracle TCPS compatibility tests remain opt-in: `TestOracleTCPSLive` needs an approved real Oracle instance and certificates. The rolling public matrix cannot certify Oracle 19c/21c/23ai or future Oracle releases.
- SQL Server matrix tests use a **loopback-only, disposable, self-signed TLS exception** and are not proof of production CA/hostname verification. Real TLS + AD/Entra authentication still need target-environment testing.
- CockroachDB `--insecure` is **CI-only**, scoped to localhost; production must use TLS.
- MariaDB `lts` follows the latest LTS track rather than every new GA release; add other tracks deliberately when required.
- Redis and ClickHouse use custom adapters rather than third-party Go client modules, so their *protocol support* requires code changes when vendors introduce incompatible features.
- A new database major release may require manual application changes, data migration, new permissions or TLS configuration. Dependabot cannot provide these automatically.
- Scheduled GitHub Actions depend on repository Actions availability and are not a guaranteed immediate release notification. Investigate workflow failures and establish owners/alerting for release-critical support.
