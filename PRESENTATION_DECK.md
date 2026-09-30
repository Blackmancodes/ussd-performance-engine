# USSD Performance Engine

## Slide 1 — USSD Performance Engine
**Repeatable load testing and live monitoring for USSD services**

Generate realistic session traffic, measure system behavior, and compare runs.

## Slide 2 — The problem
- USSD journeys are latency-sensitive and often handle critical transactions.
- Ad hoc tests are difficult to repeat or compare across releases.
- Teams need clear evidence of throughput, latency, and session outcomes before launch.

## Slide 3 — How it works
**Configure → Generate sessions → Send requests → Collect metrics → Review results**
- YAML config defines the target, journeys, networks, and run behavior.
- Seeded generation makes workloads repeatable.
- Coordinator and workers divide sessions across concurrent workers.

## Slide 4 — Workload and scale
- Models begin, continue, and end session flows.
- Generates network-aware MSISDNs and selects configured journeys.
- Compose profile: 1,000,000 sessions, 2 workers, mock receiver, 10% simulated errors.
- The configured session count is a test workload, not a published capacity benchmark.

## Slide 5 — Live observability
- Engine exposes Prometheus metrics for request counts, latency, and session states.
- Prometheus scrapes the engine; Grafana provisions the USSD overview dashboard and alerts.
- Track request rate, latency percentiles, success/failure, and session completion.

## Slide 6 — Run results
- Report tooling summarizes requests, completed/failed sessions, and average latency.
- Save and load baselines; compare runs and evaluate configured regression thresholds.
- Postman exercises the HTTP API; Newman runs the API collection in CI.

## Slide 7 — Status and next steps
**Available:** session engine, deterministic worker sharding, API, Compose monitoring stack, reports, and baseline comparison.

**Next:** validate against the production receiver contract, define SLO thresholds, and add CI performance smoke profiles. Treat Kubernetes deployment as a follow-up scale-out step.

**Takeaway:** a repeatable path from USSD session load to measurable results.
