# Verified SSH TCP tunneling (PostgreSQL / MySQL)

This is a restricted feature: SSH-tunneled connections support `postgres`, `cockroach`, `mysql`, and `mariadb` only, with the database bound on **loopback of the SSH host**. It is not a general-purpose SSH proxy. Authentication uses an unencrypted PEM/OpenSSH private key in memory; passphrase-protected keys, agent forwarding, keyboard-interactive and SSH password login are not supported. Connection credentials and private keys are never returned by the API.

## Secure deployment settings

Set all four backend environment variables:

```bash
SSH_TUNNEL_ENABLED=true
SSH_KNOWN_HOSTS_FILE=/etc/advance-db/ssh_known_hosts
SSH_BASTION_ALLOWLIST=bastion.internal:22
SSH_DB_TARGET_ALLOWLIST=127.0.0.1:5432,127.0.0.1:3306
```

Obtain the bastion's SSH host key and **verify its fingerprint out of band** before adding it to `ssh_known_hosts`. This file must exist at an absolute path and must not be writable by group or others. The backend never auto-trusts unknown SSH hosts.

In the connection form choose the SQL engine, set database hostname to `127.0.0.1` and the database port on the bastion host, select **Use verified SSH tunnel**, and provide the allowlisted SSH bastion, SSH port, SSH username and its private key. The form explicitly switches the DB TCP connection to unencrypted mode because traffic is protected by SSH to the bastion, and the remaining hop must be loopback on that same machine. **Do not use remote/private-network database targets through this mode**.

The Go backend binds a dynamically assigned `127.0.0.1` listener, opens only allowlisted SSH `direct-tcpip` connections, uses validated host keys and public-key authentication, and bounds concurrent forwarded channels at eight per database session. Tunnels close when disconnected, expired, or connection establishment fails. The server's general API token and browser-session namespace restrictions still apply.

Do not enable SSH without restrictive outbound egress rules. Host allowlists and SSH host-key verification are defense in depth, not a substitute for network segmentation. This implementation has unit and in-process SSH protocol tests; **real production bastion compatibility and high-availability failover remain unverified**.
