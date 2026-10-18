# Production security profile — deployment guide

The `ADB_PRODUCTION=true` profile rejects missing or unsafe configuration. **It is not a certificate** and may not be called compliant until all external acceptance evidence passes.

## Runtime safeguards

The Go API now refuses to start if `API_TOKEN` is weak, audit or AES-GCM vault encryption settings are missing, or write queries / loopback plaintext exceptions are enabled. It checks the vault directory/key and requires an operational audit file before listening. Audit writes are exclusive to one process, with a file lock and fail-closed readiness if log persistence fails. A multi-node deployment must use a dedicated audit sink per instance and export evidence to a protected, immutable off-host store.

Next.js checks its production configuration at Node server registration, including `OIDC_REQUIRED=true`, `SESSION_REDIS_REQUIRED=true`, `rediss://`, `APP_PUBLIC_ORIGIN=https://...`, distinct approved OIDC groups, exact callback URL and `OIDC_REQUIRED_AMR`. The OIDC verifier also validates multi-audience `azp` and configurable `amr` evidence. Password login is disabled and CSRF Origin must match the configured public HTTPS origin.

An authenticator/IdP **must actually emit the required `amr` value** (often `mfa`); test using the target IdP. If your IdP does not emit that claim, do **not** remove the check silently—use an IdP integration reviewed for equivalent MFA and adjust the policy with approval.

## Reference Compose deployment

`deploy/compose.production.yml` describes a single-node, non-root deployment with read-only container root filesystems, dropped capabilities, resource quotas and private API networking:

```bash
# Set required variables in an external deployment secret injector.
# Configure ownership and 0700/0600 access to encrypted vault and audit directories.
docker compose -f deploy/compose.production.yml config --quiet
docker compose -f deploy/compose.production.yml up -d --build
```

**Do not use plain .env files holding real production secrets.** Compose environment variables are visible through Docker/container inspection to host administrators; use a managed production orchestrator and secrets/KMS provider for actual release. The reference binds the web container only to host `127.0.0.1:3000`, behind a trusted HTTPS reverse proxy. Configure correct Host/Forwarded headers and TLS certificates. Apply network egress policy permitting only trusted IdP, Redis and approved database targets. The Go API must never be directly exposed.

Deploy `SESSION_REDIS_URL` pointing to an authenticated, certificate-verified Redis service. Control access to `APP_SESSION_SECRET`, `API_TOKEN`, `AUDIT_HMAC_KEY` and `VAULT_ACTIVE_KEY_B64` using separate, audited secret-manager permissions. The read-only production profile prohibits SQL writes through the app and requires genuine database read-only accounts.

## Manual release qualification

The explicit `production-qualification-evidence` workflow has no automatic trigger. It deliberately **fails** until a human-reviewed `deploy/evidence/phase6.json` exists, tied to the exact SHA, with recent HTTPS evidence for independent pen test, live IdP/MFA, offboarding, Redis/KMS recovery, off-host audit, alerts, load, DR, rollback, supply-chain licensing and approval. It also requires success on the exact Git commit of quality, security and release-candidate workflows, plus an active review-enforcing GitHub branch ruleset.

Set up the `production-approval` GitHub Environment with a qualified **independent required reviewer**, deployment restrictions and least privileges. Set the branch protection/ruleset to require pull requests and status checks. Repository settings cannot be proven by source files; an authorized administrator must configure them.

The evidence validator checks only structure, freshness and commit association; it does not independently verify the security conclusions or make a certification claim. Do not put private evidence details into a public Git repository.

## Evidence still needed outside CI

- Real IdP SSO/MFA sign-in, authorization mapping, group revocation, JWKS rollover, and user offboarding.
- Distributed Redis session/logout/replay tests across containers; encryption key rotation and rollback under controlled access.
- Protected off-host audit retention and alert delivery from a live Prometheus/Alertmanager stack.
- Controlled failover/load tests, measured SLOs, and recovery drills with real RPO/RTO.
- Independent security assessment, review approvals and an attested canary/rollback.

See [Phase 6 Runbook](PHASE6_RUNBOOK.md), [Vault](CREDENTIAL_VAULT.md), [Observability](OBSERVABILITY.md) and [Release Provenance](RELEASE_PROVENANCE.md).
