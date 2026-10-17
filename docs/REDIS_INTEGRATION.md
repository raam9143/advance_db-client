# Redis inspector — read-only

The Redis engine uses a separate RESP2 connection path, not SQL emulation. Browser sessions retain isolated, ephemeral credentials in memory for at most two hours.

Read commands are allowlisted: SCAN (50-count pages), GET, TYPE, TTL, HGETALL and LRANGE (max 100 items). Responses are bounded to 4 MiB and each bulk value to 1 MiB. Commands like SET, DEL, CONFIG, EVAL, FLUSHALL, KEYS, ACL and SHUTDOWN are not available.

By default, Redis requires TLS 1.2+ with certificate validation and uses port 6380. For **loopback-only development**, set `REDIS_ALLOW_PLAINTEXT_LOOPBACK=true` on the Go API and select `sslMode=disable`. The server enforces a loopback host. Never enable this bypass for public endpoints. Connect with a Redis user whose ACL permits only the required read commands.

CI starts Redis 7.4 and runs basic integration tests. This is not Redis Cluster/Sentinel support, TLS integration against a real remote server, durable credentials, or bulk backup/restore.

Examples after connecting through the UI: inspect key `session:123` with GET, inspect a hash with HGETALL, or enumerate keys with SCAN and pattern `app:*`. Scan cursor values are updated after each page.
