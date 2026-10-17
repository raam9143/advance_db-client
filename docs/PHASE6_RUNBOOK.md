# Phase 6 enterprise release and recovery runbook

**Phase 6 is not ready for production certification until its issue acceptance criteria are closed by an authorized security reviewer.** Passing CI, signing artifacts, an SBOM, a successful restore or OIDC unit tests each validate only their documented scope.

## Identity and access
1. Register the app as an OIDC relying party with an enterprise identity provider. Configure exact HTTPS issuer, redirect URI, client ID, optional client secret and mandatory admin/viewer groups.
2. Configure `OIDC_REQUIRED=true` to disable local password login; require IdP-controlled MFA, conditional access and account lifecycle controls. Choose `OIDC_ADMIN_GROUP` and `OIDC_VIEWER_GROUP` that cannot be self-assigned.
3. Register the exact callback `https://db.example.com/api/auth/oidc/callback`. Keep `APP_SESSION_SECRET` at least 32 random characters and rotate it on confirmed cookie compromise (old sessions become invalid).
4. Browser-session identities are verified with HMAC and forwarded to Go. OIDC viewers are forced to read-only connections; enforce **database-level SELECT-only accounts** because lexical SQL checks cannot safely replace database authorization.
5. Sign-in uses OIDC authorization-code+PKCE with state/nonce, HTTPS issuer metadata, pinned issuer/origin and RS256 JWKS verification. Validate browser/callback redirects, group claims, IDP logout, MFA prompts, failed authorization, tampered JWT, token expiry, rotation of signing keys, and offboarding using an actual identity provider. The project has unit tests but no live IdP acceptance test.
6. Do not horizontally scale until distributed sessions, logout revocation, MFA replay protection and audit sink sharing have been implemented and tested. The optional TOTP remains one Node-process only.

## Secrets and credentials
- Keep `APP_SESSION_SECRET`, `API_TOKEN`, `APP_TOTP_SECRET`, `OIDC_CLIENT_SECRET`, `AI` keys, `AUDIT_HMAC_KEY` and database credentials in a managed secret service with audited access, not GitHub variables, app logs, localStorage or checked-in files.
- Database connection secrets currently live **in Go process memory only** and expire with sessions; no persistent credential vault exists. Do not promise encrypted credential-at-rest or key rotation for stored database secrets without implementing them.
- Rotate API/token secrets through a staged, recorded operator maintenance window. Reset all browser sessions on APP_SESSION_SECRET rotation.
- SSH bastions and target ports must be explicitly allowlisted with out-of-band verified known-hosts fingerprints.

## Security and supply chain
- GitHub security workflow performs npm production vulnerability audit, Go vulnerability audit, PR dependency review, and uploads SPDX SBOM artifacts. Resolve high/critical findings before tagging a release.
- Keep lockfiles exact (`npm ci`; `go mod tidy -diff`). Review action source and pin immutable commit SHAs for release-hardening, rather than trusting mutable tags in production.
- Branch protection/rulesets must require all quality and security checks and human review. This is a GitHub repository setting and is not established just by adding a workflow.
- Perform SAST, image vulnerability scanning, license provenance assessment, penetration testing, secrets scanning, signed-release attestation and threat modeling in the deployment environment.

## Availability and SLOs (proposed, not validated)
- Request availability target: 99.9% per rolling 30-day period for the UI/API endpoints.
- p95 API latency target under 2 seconds for non-database metadata actions (excluding external SQL query timings).
- Successful identity redirects and tenant isolation must be measured separately. Track 4xx/5xx, pending queries, pool saturation, backup ages, TOTP failures, stale sessions, audit write failures and external provider errors.
- Alert on 5xx above 1% for 5 minutes, authentication anomaly spikes, export failure rate, and any audit-integrity failure. Configure this in your observability stack. **These alerts are not provisioned by this repo.**

## Deployment
- Test migration on a sandbox with the same database engines, TLS certificates, IdP tenant groups and network policies as production.
- Put HTTPS reverse proxy in front; Go must be private. Harden container users, resource limits, quotas, egress and TLS.
- Before deployment: snapshot databases and config, verify recovery access, validate SBOM/vulnerability scans, execute functional/permission/IdP checks, and approve release.
- Deployment can use blue/green or canary, with rollback to the last attested image on failed health/latency/SLO checks.

## Recovery and drills
- For SQLite, the offline validated restore CLI retains the original. Test and document a maintenance-window restoration using `docs/SQLITE_RESTORE.md`.
- For PostgreSQL/MySQL/SQL Server/Oracle/MongoDB/ClickHouse, configure the vendor's native backup tooling, retention, encryption and recovery drills. The app does not provide their end-to-end backup/restore.
- Validate both RPO and RTO on disposable data; record actual figures, do not invent them.
- Run a simulated loss of Next.js, Go, secret-store access, IdP outage, compromised token, audit corruption, and SSH tunnel failure. Track the exercise evidence outside the app.

## Release acceptance
Phase 6 closes only after the repository owner can demonstrate all requirements from issue #6: enterprise authentication/RBAC, distributed session handling, key rotation, effective durable audit, observable SLOs, vulnerability/SBOM/license/image gates, E2E/DR/performance tests, and signed/reviewed release. Open items must never be disguised as completed by changing documentation alone.
