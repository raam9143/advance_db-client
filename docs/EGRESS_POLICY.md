# Production database endpoint allowlist

Set `ADB_PRODUCTION=true` and `DB_ENDPOINT_ALLOWLIST` to a comma-separated list of exact database `host:port` pairs, for example `db.internal:5432,sql.example.net:1433,[2001:db8::3]:2484`. Connections are refused before dialing if their requested host:port is absent. No wildcard matching, CIDR networks, URL components, or link-local addresses are permitted; an empty list fails startup. Inline credentials and encrypted profile references both follow the same restriction.

For SSH tunneling, list the database destination bound on the SSH bastion's loopback (such as `127.0.0.1:5432`) and set the separate strict `SSH_BASTION_ALLOWLIST`, `SSH_DB_TARGET_ALLOWLIST`, and `SSH_KNOWN_HOSTS_FILE`. This allowlist does **not** replace host firewall, DNS-pinning at the connection layer, egress proxy, or TLS name verification: an approved DNS name can change resolution after the application verifies the name. Enforce the same destination restriction with your VPC/firewall egress rules and audit all configuration changes.

The CI tests verify exact matching, production fail-closed behavior, and denial of wildcard/link-local/invalid entries. The default development mode does not impose this policy.
