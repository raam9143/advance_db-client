# Offline SQLite snapshot restoration

This is an **offline, operator-run** safety tool, not an HTTP endpoint and not automatic database disaster recovery. It is limited to files already present within the existing configured SQLite root and backup root.

1. Stop the application and every SQLite writer; checkpoint WAL before shutting down. The tool rejects `-wal`, `-shm` and `-journal` sidecars as a defense in depth, but cannot detect every other process that has a database open.
2. Supply the absolute `SQLITE_ALLOWED_DIR` and `SQLITE_BACKUP_DIR` environment variables. Do not use world-writable locations.
3. Place a verified snapshot (for example, from the temporary API SQLite snapshot download) in the backup root with owner-only file permissions.
4. Dry-run and validate both source and destination first:
   ```bash
   cd backend
   go run ./cmd/sqlite-restore -snapshot /secure/backups/snapshot.sqlite -database /secure/data/app.sqlite
   ```
5. Only after approval and a maintenance-window lock, execute with `-confirm RESTORE`. The tool stages a private candidate database beside the destination, verifies SQLite `quick_check`, renames the original to a uniquely named `.adb-before-restore-*.sqlite` file and installs the checked replacement. It never silently discards the original.

Stop immediately if any validation or rename fails. Back up the retained original separately, verify application data and permissions, and resume writers only after checks. File rename atomicity assumes a local filesystem and a single operator. OS crashes and hardware failure cannot be completely guarded against; use durable backup policies, checksums outside this tool and tested disaster-recovery procedures.

**Limitations:** only SQLite, 128 MiB files, no web restore, no online restore, no multi-database consistency point, no automatic rollback after the replacement succeeds, and no enterprise approval workflows.
