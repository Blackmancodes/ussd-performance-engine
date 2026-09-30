# USSD Performance Engine — 7-Slide Outline

1. **Title and purpose** — Repeatable load testing and live monitoring for USSD services.
2. **Problem** — Customer-critical USSD flows need performance checks that teams can repeat and compare.
3. **How it works** — YAML configuration, seeded session generation, coordinator/worker execution, metrics, and reports.
4. **Workload and scale** — Session lifecycle and network-aware traffic; current Compose profile is configured for 1,000,000 sessions across 2 workers with a mock receiver and 10% simulated errors. This is a workload setting, not a capacity claim.
5. **Observability** — Prometheus metrics and Grafana dashboards/alerts for request rate, latency, and session outcomes.
6. **Results and regression checks** — Reports summarize runs; stored baselines and threshold checks help identify changes. Postman/Newman cover the HTTP API.
7. **Status and next steps** — Engine, API, monitoring, and reporting are available. Validate the production receiver contract, set SLO thresholds, and add CI performance smoke profiles.
