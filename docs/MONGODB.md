# MongoDB read-only collections and documents

The MongoDB engine is separate from SQL. It uses the official Go v1 driver and exposes only:

- `POST /v1/connections` with `engine:"mongo"`, `host`, `port`, `database`, `user`, `password`, and `sslMode:"require"` (TLS/certificate validation by default).
- `GET /v1/connections/{id}/schema`: returns up to 1,000 collection names; each appears with the selected database as `schema`.
- `POST /v1/connections/{id}/mongo/find`: `{"collection":"orders","limit":25,"offset":0,"filterField":"status","filterValue":"pending"}`. `filterField`/`filterValue` are optional. Only exact equality with JSON scalar values is supported; no arbitrary BSON operators, aggregation pipelines, updates or DB commands.

`find` is limited to 100 documents per request and 10,000 offset, with a 1 MiB per-document and 4 MiB total JSON limit. Results use Extended JSON to preserve BSON type information. Browse sorted by `_id`; app changes between pages can still affect pagination. Timeout is bounded.

Remote connections must use validated TLS (no `tlsInsecure`). For an **isolated local test only**, `MONGO_ALLOW_PLAINTEXT_LOOPBACK=true` enables `sslMode:"disable"` for loopback/localhost; no non-loopback exceptions exist. CI supplies a disposable MongoDB container. Never use a broad-permission MongoDB credential; use a database read-only role.

Not supported: Atlas SRV/multi-host connection strings, replica-set topology configuration, sharding options, schema inference, CRUD modifications, transactions, imports, full backups, GridFS and aggregates. Real-world replica set/Atlas TLS compatibility still needs validation.
