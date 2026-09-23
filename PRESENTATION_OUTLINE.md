# USSD Performance Engine — Presentation Outline

## 1. Executive summary
- Purpose: load-test a USSD gateway with realistic session traffic and measurable SLOs
- Key value: deterministic, reproducible, observable, and scalable
- Outcome: we can generate 1M+ request load safely and compare runs against a baseline

## 2. Problem statement
- USSD systems are bursty, session-driven, and sensitive to latency and outage windows
- Existing load testing is usually ad hoc and not reproducible
- Engineering teams need a repeatable way to test real-world session behavior

## 3. Architecture
- Coordinator/worker model for scale
- Deterministic sharding of MSISDN and journeys
- Prometheus + Grafana observability
- Report service for post-run summarization

## 4. Key technical decisions
- YAML-driven config for journeys, MNO pools, thresholds, and behavior
- Seeded randomness for reproducible session generation
- Session-aware requests and response handling
- Multi-stage runtime: local dev, Docker Compose, distributed workers, dashboards

## 5. Metrics and observability
- `ussd_requests_total`
- `ussd_request_duration_seconds`
- `ussd_sessions_total`
- Grafana dashboards showing request rate, latency, journeys, MNOs, session outcomes

## 6. Stage progress
- Stage 1: single-node session engine
- Stage 2: metrics and local stack
- Stage 3: coordinator/worker distributed scaling
- Stage 4: Grafana dashboard
- Stage 5: report generation and baseline comparison
- Stage 6: baseline regression detection
- Stage 7: Kubernetes + CI future path

## 7. Baseline and regression model
- Capture current metrics snapshot as baseline
- Compare future runs using deltas on request volume, failure rate, and latency
- Flag worsening performance before it becomes an outage

## 8. Business impact
- Reduces performance risk before production changes
- Gives engineering and leadership a single source of truth for test quality
- Enables repeatable performance validation for USSD products and partners

## 9. Next steps
- Finalize real receiver contract
- Add stronger SLO thresholds and alerting
- Expand to Kubernetes and CI smoke profiles
- Package standardized run reports and deck-ready summaries

## 10. Closing slide
- Reproducible performance testing for critical customer-facing USSD systems
- Built for scale, observability, and operational confidence
