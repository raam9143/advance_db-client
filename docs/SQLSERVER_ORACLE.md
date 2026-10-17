# Microsoft SQL Server and Oracle connectivity

These engines have driver-backed Go connection support; this is not a complete port of all SQL Server/Oracle capabilities.

## SQL Server (TDS)

- Go driver: `github.com/microsoft/go-mssqldb` **v1.8.2**, pinned for the Go 1.23 project baseline.
- Connection type: `sqlserver`, SQL authentication, default port 1433 and mandatory encrypted traffic with `TrustServerCertificate=false` by default.
- Basic schema enumeration, columns, primary keys and indexes; SQL Server `OFFSET/FETCH` table pagination and `@p1` bind parameters.
- Explanations, SSPI, Microsoft Entra identity, Azure-specific authentication, Always Encrypted, transactional management and vendor-specific features have **not** been integrated.
- A loopback-only test mode `sslMode=local-insecure` requires `SQLSERVER_ALLOW_UNVERIFIED_LOOPBACK=true`, and requires an actual loopback hostname/IP. It still encrypts network traffic but does not validate the local certificate. Do not enable this option in production.

CI provisions disposable Microsoft SQL Server 2022 with a nonproduction password and validates the real API behavior. The test lives in `backend/integration/sqlserver_integration_test.go`.

## Oracle (TCPS)

- Pure-Go driver: `github.com/sijms/go-ora/v2` **v2.9.0**. Oracle Instant Client libraries are not required.
- Connection type: `oracle`, recommended TCPS port 2484. Only certificate-verified TLS is accepted by the API. `Database` contains Oracle Service Name (usually a PDB service) or SID if `serviceMode=sid`.
- Catalog read: `ALL_TABLES`, `ALL_TAB_COLUMNS`, `ALL_CONSTRAINTS`, `ALL_IND_COLUMNS`; default table enumeration scopes to the authenticated schema's tables.
- Oracle 12c+ `OFFSET/FETCH` pagination and `:1` ordinal bind parameters. EXPLAIN, stored procedures, RAC-specific failover, wallet management, and Oracle-native backup/restore are not supported.
- With Oracle TCPS, verify that the issuing certificate authority is trusted by the Go server and the certificate matches the Oracle hostname. Connection failures will remain errors; the application has no disable-verification switch.

An **optional** smoke test exists under `backend/integration/oracle_integration_test.go`. To run only that test, supply `TEST_ORACLE_HOST`, `TEST_ORACLE_SERVICE`, `TEST_ORACLE_USER`, `TEST_ORACLE_PASSWORD`, optional `TEST_ORACLE_PORT` (default 2484) and `TEST_ORACLE_SERVICE_MODE` (`service` or `sid`); then run:

```bash
cd backend
go test -run '^TestOracleTCPSLive$' -v ./integration
```

Run against **a disposable, authorized test database** with a read-only account. No TCPS Oracle instance is provisioned in public GitHub CI; therefore real Oracle connectivity and Oracle metadata compatibility remain **unverified**.

## Safety and correctness

Always use a database login with SELECT-only permissions for read-only access. Some drivers do not support database/sql's `TxOptions.ReadOnly`; the API checks SQL text conservatively, but lexical checks do not replace database permissions. Writes are disabled server-side unless explicitly enabled and confirmed. Avoid passing secrets in SQL text. Backups for these engines are not implemented.
