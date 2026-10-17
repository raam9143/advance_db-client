# Release-candidate provenance

The `reproducible-release-candidate` workflow builds Go Linux amd64 and arm64 API binaries and a standalone Next.js bundle from locked Go/npm dependencies. It runs code and production dependency checks, produces SHA256SUMS, uploads an artifact and requests GitHub build-provenance attestations. It does **not** automatically publish a tagged release or deploy to any environment.

To verify downloaded files, compare `sha256sum -c SHA256SUMS` in the extracted artifact, inspect the originating Git commit and workflow job, and use GitHub's attestation verification tooling against the exact artifact digest and repository identity. SHA-256 manifests alone are not signatures. Public repositories can use GitHub artifact attestations when the repository and Actions permissions support them; audit each completed workflow result.

The standalone frontend archive is application content, not a full OCI container image. Any deployment package requires base-image SCA, runtime configuration, reverse-proxy TLS and environment-specific validation. Do not deploy an unreviewed candidate to production.

Release creation is intentionally manual until Phase 6's remaining security/DR acceptance criteria are met. Tag protection, branch required checks, CODEOWNERS review, key custody, production E2E, SBOM/license gate, image scanning, rollback validation and actual OIDC provider acceptance must be verified separately.
