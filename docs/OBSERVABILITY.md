# Production metrics and operations

The Go API exposes `GET /v1/metrics/prometheus` to an authenticated administrator. It returns only fixed-label metrics: HTTP request totals, 1xx–5xx response counts, bounded duration histogram, and uptime. It contains no SQL, database hostnames, usernames, credentials or session identifiers.

Do not expose the endpoint directly to the public Internet. Configure Prometheus with a private-network scrape and a bearer token sourced from an external secrets manager. Configure `deploy/monitoring/alerts.yml` as a Prometheus rule group to alert on high 5xx rates and missing metrics. Assign notification receivers and silences in your deployment (not configured in this repo). Test alert delivery before production.

The metrics are in-process and reset on restart; use Prometheus persistence and recording rules for historical SLO evaluation. Phase 6 also requires telemetry for identity flows, Redis session failures, audit integrity, DB pool saturation, backup failures, and remote provider errors, but those signals and distributed traces are not yet implemented. These metric and alert definitions are not an operational SLO commitment until your monitoring stack and notification receivers are deployed and exercised.
