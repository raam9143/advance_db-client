# Local read-only MCP bridge

A minimal stdio MCP bridge compatible with clients implementing the **2025-11-25** lifecycle; not a complete upstream DBX MCP port or the newer 2026-07-28 protocol.

Start with Node.js 22+:

```sh
MCP_API_TOKEN='<same-token-as-go-api>' MCP_GO_API_URL=http://127.0.0.1:8080 node mcp/server.mjs
node --test mcp/server.test.mjs
```

Configure the MCP client command to run `node /absolute/path/to/mcp/server.mjs` and set environment variables in the client's trusted configuration (not in source control).

The tools expose temporary connections and allow schema discovery, read-only queries and EXPLAIN only. They do not create connections, execute writes, or manage production credentials. The Go API bearer token can access all in-process connections: keep this MCP bridge local to a trusted user. Restrict database roles to SELECT-only. Avoid submitting secrets in SQL text. 
