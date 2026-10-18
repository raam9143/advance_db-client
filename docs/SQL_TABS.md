# Independent SQL query tabs

The SQL editor supports up to eight in-memory tabs per browser page. Each tab maintains independent SQL text; switching tabs clears the result panel rather than presenting another tab's stale results as current. Executing statements or switching connections does not silently persist SQL text. **Query tabs do not survive page reloads**, and saved SQL remains an explicit user action with separate browser-local storage.

Tabs include accessible labels, keyboard focus indicators, and close/new actions. Closing the final remaining tab is disabled. Tab changes are disabled while execution is running to avoid showing mismatched results. Per-user server-encrypted persistence, syntax highlighting and transaction-local query sessions are **not** yet implemented.
