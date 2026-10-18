# Security limitations and deployment checklist

This is a **developer-grade migration starter**, not audited enterprise database administration software. Database clients can access production data, so use least-privilege accounts and test against non-production databases first.

## Required

- Set high-entropy, unique `APP_PASSWORD` (at least 12 characters), `APP_SESSION_SECRET` (at least 32), and `API_TOKEN`. Never embed them in the frontend or commit `.env`.
- Serve Next.js behind **HTTPS** on public networks. Keep the Go API reachable only from localhost or a private/container network. Never expose the Go API on an untrusted network without transport security.
- Use actual **read-only database users** for browsing. SQL heuristics and read-only transaction flags are defense in depth, not authorization.
- MySQL/PostgreSQL TLS must be enabled and properly validated where applicable. Turn off SSL only for isolated, trusted development instances.
- Set `SQLITE_ALLOWED_DIR` to a dedicated directory containing only intended files; no world-writable directory. SQLite sandbox path resolution also rejects symlink paths escaping the directory.
- Use a trusted reverse proxy that preserves the expected Host/Origin relationship. Deny cross-origin requests and never serve wildcard CORS for the authenticated API.
- Restrict outbound network reachability from Go if users should not connect to arbitrary database hosts (SSRF/egress controls).

## Current controls

- HMAC-signed HttpOnly SameSite=Strict cookies expire after eight hours; `Secure` is set in production.
- Logout clears the browser cookie; previously stolen cookies are **not** actively revoked until expiration.
- Login attempts are throttled in a single Node process, **not distributed**. Production should add a shared Redis limiter or reverse-proxy controls.
- All browser mutation routes require same-origin verification; Go uses a constant-time Bearer token check.
- Query duration at most 30 seconds; result cap 1000 rows; connection TTL 2h; individual API request max ~1MB.
- Database credentials are never returned in API responses, logged, or stored in browser localStorage. In-memory database handles retain authentication until expiration/disconnect.
- Query history **does** remain in browser localStorage; do not put passwords or secrets into SQL text.

## Missing for production

Multi-user/enterprise MFA, OAuth/OIDC, RBAC, centralized audit evidence, password hashing with persistent user accounts, revocation registry, secrets vault integration, distributed login throttling, TLS termination automation, network policy, stronger SQL parsing and write approval, driver/version integration tests, policy and vulnerability scanning. All require implementation and audit before enterprise rollout.

## Optional single-admin TOTP authenticator
Set `APP_REQUIRE_TOTP=true` and `APP_TOTP_SECRET` to a securely generated Base32 secret decoding to at least 20 bytes. Enroll that secret into an authenticator app through an out-of-band administrative process (never commit it, log it, or share it through URLs). Login requires both the existing password and a 6-digit RFC 6238 SHA-1 code, allowing a ±30-second clock skew. Codes cannot be reused within the same running Node process. CI verifies RFC vectors, replay denial and invalid-secret handling. **This is single-instance only:** the replay ledger and login limiter are not distributed or durable. It does not provide individual users, OIDC/SSO, WebAuthn, recovery codes, or device enrollment, and should not be treated as enterprise MFA.

## Newly added opt-in controls (not enterprise certification)
- [SSH tunnel](SSH_TUNNEL.md): public-key authenticated bastion with strict `known_hosts`, bastion/DB allowlists, and loopback-only database end points. PostgreSQL/MySQL protocol only; no unrestricted TCP forwarding.
- [Audit](PHASE_STATUS.md): `AUDIT_LOG_PATH` and `AUDIT_HMAC_KEY` enable durable local JSONL with keyed integrity chaining. The audit directory must be private (0700). Audits record hashed paths and owners, not SQL or credentials. This is single-process, not truly append-only or cryptographically anchored off-host; external rotation/retention, multi-instance locking, verification during operation and central shipping are not implemented.
- [Offline restore](SQLITE_RESTORE.md): a maintenance-only SQLite CLI verifies database health and retains the original. All writers must be shut down. It does not restore other database engines.
- Optional TOTP under `APP_REQUIRE_TOTP=true` and `APP_TOTP_SECRET`. Verify the provisioning and replay counter limitations; production OIDC/SSO/RBAC is still missing.
- [AI review](AI_REVIEW.md): requests are disabled by default, require user consent, and contain sanitized SQL text only. Table/column identifiers can still reveal organizational structure.

### Credential encryption and rotation
The admin-only encrypted profile store is opt-in. Read [the credential vault runbook](CREDENTIAL_VAULT.md) for AES-256-GCM storage, owner-private filesystem restrictions, profile-bound integrity tags, and the offline old-to-new key re-encryption procedure. Production keys must be injected by an independent secrets service. The vault is not an external KMS/HSM, centralized multi-tenant ACL system or cloud secret broker.

### Admin throttling and observability
With a required Redis-backed session store, the local-admin password login now uses shared atomic rate limits and fails closed on Redis errors. Browsers and reverse proxies cannot reset the counter by submitting alternate `X-Forwarded-For` values. Prometheus endpoint `/v1/metrics/prometheus` requires API authorization and the admin role; review [monitoring integration](OBSERVABILITY.md) before providing a private, authenticated scrape route.

**Phase 6 sign-off warning:** GitHub CI checks and the reference runbooks are not an approved security assessment. The owner must confirm IdP group mappings/offboarding, enforce branch protections, provision TLS Redis and monitoring, review key custody, and execute a real recovery drill before enterprise release.
