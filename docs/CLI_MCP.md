# Read-only CLI and MCP usage

## CLI
Requires Go 1.23+ and a running Go API:
```sh
cd backend
go run ./cmd/adbctl connections
ADB_API_TOKEN='...' ADB_API_URL=http://127.0.0.1:8080 go run ./cmd/adbctl schema -id <48-hex-id>
ADB_API_TOKEN='...' go run ./cmd/adbctl query -id <48-hex-id> -sql 'SELECT ?' -params '[42]'
```

## MCP
Node.js 22+: `MCP_API_TOKEN='...' node mcp/server.mjs`. Configure in a trusted MCP client's stdio server settings. See `mcp/README.md`.

These initial integrations deliberately exclude writes, connection creation, credentials, cloud connections and plugins. `ADB_API_TOKEN` and `MCP_API_TOKEN` are sensitive and must never be included in command arguments or Git history. Session isolation, OIDC/SSO, complete audit, approval workflows and enterprise-grade security are not implemented.
