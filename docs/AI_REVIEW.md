# Optional AI SQL review

This endpoint performs **no SQL execution**. It is disabled by default and requires `AI_ASSIST_ENABLED=true`, `OPENAI_API_KEY` (server-only), and optionally `AI_MODEL` (default `gpt-4o-mini`).

The user explicitly consents before each request. Only a single bounded (4 KiB), read-only SQL statement can be reviewed. String literal contents are redacted before sending; credential-related identifier patterns cause refusal. **SQL identifiers still reveal table/column structure**, so use only with data-sharing approval. The backend does not send rows, connection passwords, hostnames, database handles, or full schema dumps. The model receives no tools, response storage is requested disabled (`store:false`), and output is capped and presented as advisory text.

API: `POST /v1/ai/review`, `{"sql":"SELECT ...","purpose":"explain","consent":true}`. The endpoint never runs SQL returned by the model. The provider host is fixed to `https://api.openai.com/v1/responses`; clients cannot supply a custom URL. Requests are concurrent-limited to two, timeout after 20 seconds, and no provider errors or token values are exposed.

Tests use a local mock provider. **A live authenticated provider integration has not been performed**, and generated explanations have not been evaluated for SQL semantic accuracy. Schema approval and a model-risk review are required before production use.
