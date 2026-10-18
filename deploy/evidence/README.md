# Evidence for manual production qualification

The production-qualification workflow deliberately fails until the repository owner provides a separately reviewed `phase6.json` that matches the exact commit and contains **working HTTPS links to actual evidence** for each key listed in `deploy/qualification/check.mjs`.

Provision `production-approval` GitHub Environment with **independent required reviewers** and protected-branch deployment restrictions. Protect `main` with required quality, security and release-candidate checks and code review. The workflow also checks for an active branch ruleset requiring pull-request reviews.

Do not include credentials, PHI, database rows, passwords, or private report contents in this public repository. Only publish secure links to appropriately access-controlled internal evidence. Links and self-attestation are not equivalent to audit/signature verification.

This code checks structure and recency only; approval must be performed by a qualified human security authority. No certification is issued automatically.
