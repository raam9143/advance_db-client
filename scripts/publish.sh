#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
command -v gh >/dev/null || { echo 'GitHub CLI (gh) is required.' >&2; exit 1; }
command -v git >/dev/null || { echo 'git is required.' >&2; exit 1; }
[[ ! -d .git ]] || { echo 'Local Git repository already exists. Inspect before pushing.' >&2; exit 1; }
[[ "$(gh api user --jq .login)" == 'raam9143' ]] || { echo 'Authenticate as raam9143 with gh auth login.' >&2; exit 1; }
if gh repo view raam9143/advance_db-client >/dev/null 2>&1; then
 echo 'Target repository already exists; refusing to overwrite.' >&2; exit 1
fi
git init -b main
git add .
git commit -m 'feat: bootstrap Go and Next.js database client migration'
gh repo create raam9143/advance_db-client --private --source . --remote origin --push
echo 'Published https://github.com/raam9143/advance_db-client'
