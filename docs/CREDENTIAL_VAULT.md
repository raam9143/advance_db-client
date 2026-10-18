# Encrypted operator-managed credential vault

The backend supports **opt-in** AES-256-GCM encrypted connection profiles at rest. All records use unique random nonces, profile-bound authenticated associated data and a key ID. The 32-byte keys must come from a trusted external secret manager. Sensitive profile data is not returned by `GET /v1/connections`, browser SQL history, or the vault CLI `list`.

## Installation and provisioning

1. Make a dedicated vault folder: `mkdir -p /var/lib/advance-db-client/vault && chmod 700 /var/lib/advance-db-client/vault`. Restrict its ownership to the Go process/authorized maintenance identity. It must not be a world-readable shared volume.
2. Provision `VAULT_FILE=/var/lib/advance-db-client/vault/credentials.enc` and `VAULT_ACTIVE_KEY_B64` with an independently generated, securely stored Base64 encoding of 32 random bytes (for example, `openssl rand -base64 32`). Inject these at runtime using your KMS/secret-injection platform. Never commit a real key or credentials.
3. From a trusted, offline operator session (backend Go 1.23+), use `go run ./cmd/adb-vault put -name reporting_prod` and supply a JSON object on stdin containing `engine`, `host`, `port`, `database`, `user`, `password`, `sslMode` and optionally `serviceMode`. Avoid command-line arguments, shell history, and logs for secrets.
4. The connection form can use **Encrypted credential reference**, or the authenticated admin may POST `{"name":"Reporting","credentialRef":"reporting_prod"}` to `/v1/connections`. The backend forbids merging arbitrary inline host/password overrides with a stored reference, forces the resulting session **read-only**, and rejects viewers. The database account itself must also have SELECT-only permissions.

## Key rotation without exposing plaintext profiles

1. Back up the encrypted vault file securely and ensure one authorized operator is modifying the file; the local vault is not a multi-writer database.
2. Generate a completely new 32-byte random active key. Set `VAULT_ACTIVE_KEY_B64` to the new key and `VAULT_PREVIOUS_KEYS_B64` to the old key (multiple old keys may be comma-separated, maximum four).
3. Run `go run ./cmd/adb-vault rotate` during a controlled maintenance window. It decrypts **all records**, verifies integrity, and atomically replaces the vault file with entries encrypted under the new active key.
4. Verify connection references work with the new active key alone, then retire the previous key. Keep the encrypted pre-rotation backup and old key according to approved recovery/key custody policy until the rotation has been signed off.

The loader rejects symlinks/unsafe file modes and requires an owner-private (0700) parent directory and private (0600) file. It has tamper, key-mismatch, re-encryption and RBAC tests in CI.

**Limits:** Encryption at rest does not protect secrets from a compromised Go process. The keys are supplied via environment/secret injection; this implementation does not perform cloud KMS/HSM signing, remote vault token lease/renewal, per-user secret ACLs, coordinated key rotation between multiple running Go replicas, or independent penetration testing. The CLI uses single-writer maintenance mode; concurrent operator edits are unsupported.
