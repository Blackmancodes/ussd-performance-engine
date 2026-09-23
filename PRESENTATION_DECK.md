# USSD Performance Engine

## Slide 1 — Title
USSD Performance Engine

Subtitle:
Reproducible, scalable load testing and performance monitoring for USSD services

---

## Slide 2 — Executive summary
- We built a deterministic USSD load-generation engine for testing critical messaging flows
- The platform measures request volume, latency, session outcomes, and journey health
- It supports local Docker-based runs and multi-worker distributed execution
- It produces a report and baseline comparison for regression detection

Key message:
This gives the team a repeatable, observable way to test USSD systems before production changes.

---

## Slide 3 — Problem statement
- USSD is customer-facing and latency-sensitive
- Small production issues can create user-visible service degradation
- Performance testing is often ad hoc, non-repeatable, and hard to compare across runs
- Teams need to answer: How much traffic can this service handle? What changed after a release?

Business impact:
Unplanned outages, failed transactions, and poor customer experience.

---

## Slide 4 — Why this project matters
- USSD remains critical for low-bandwidth, mobile-first user journeys
- Transactional services must sustain high concurrency without silent performance decay
- Reliable testing must cover:
  - request throughput
  - latency percentiles
  - session completion rates
  - failure conditions
  - operator and journey mix

Outcome:
A test harness that behaves like a real service load profile, not a synthetic loop.

---

## Slide 5 — System overview
- Config-driven engine
- Journey and menu simulation
- MNO-aware MSISDN generation
- Coordinator/worker execution model
- Prometheus metrics collection
- Grafana dashboards
- Report service and baseline comparison

Architecture statement:
This is a full performance testing pipeline, not just a traffic generator.

---

## Slide 6 — Core technical design
- YAML-driven configuration for target, journeys, MNO pools, and scaling
- Seeded randomness for reproducible runs
- Session state machine for begin/continue/end behavior
- Deterministic worker sharding to avoid session collisions
- Prometheus instrumentation for requests, latency, and session states

Why this matters:
Reproducibility is the foundation for trustworthy performance analysis.

---

## Slide 7 — Distributed execution model
- Coordinator assigns deterministic work slices to workers
- Workers run session generation concurrently
- No duplicate MSISDN or journey ownership overlaps across workers
- Combined metrics reflect total system performance

Result:
The platform scales beyond single-process capacity while preserving deterministic behavior.

---

## Slide 8 — Observability and monitoring
Metrics collected:
- `ussd_requests_total`
- `ussd_request_duration_seconds`
- `ussd_sessions_total`

Visualization:
- request rate
- latency trends
- journey distribution
- MNO mix
- session completion and failure states

Key point:
The system is designed for live operational insight, not just post-run analysis.

---

## Slide 9 — Grafana dashboard value
- Live monitoring during the test run
- Operational visibility for engineering teams
- Quick diagnosis of latency spikes and failing segments
- Immediate checks for SLO drift

Dashboard example panels:
- requests per second
- latency p50/p95/p99
- successful vs failed sessions
- journey mix
- MNO distribution

---

## Slide 10 — Report service and baseline comparison
- After a run, the engine summarizes the result set
- Report artifact includes total requests, session results, and latency
- A baseline can be stored and compared against future runs
- Deltas highlight regressions or improvements

Example deltas:
- request delta
- completed-session delta
- failed-session delta
- latency delta

---

## Slide 11 — Regression detection
- A new run can be automatically compared with an earlier baseline
- The team can detect if:
  - latency increased unexpectedly
  - session failures rose
  - throughput fell below expectation
  - the platform drifted outside target SLOs

This moves the project from raw data collection to actionable operations intelligence.

---

## Slide 12 — Project status
Completed:
- session engine
- YAML configuration and deterministic journey logic
- metrics emission
- Prometheus integration
- Docker Compose runtime
- distributed workers
- Grafana dashboard provisioning
- reporting layer and baseline comparison foundation

Still ahead:
- deeper SLO policy automation
- Kubernetes deployment manifests
- CI smoke pipeline
- production receiver contract integration

---

## Slide 13 — Business value
- Reduces performance risk before a major release
- Improves confidence in customer-facing USSD flows
- Makes performance validation repeatable and auditable
- Gives operational teams a single source of truth for testing outcomes

This is not just a testing tool; it is an operational capability.

---

## Slide 14 — Roadmap
Phase 1 — Local engine and metrics
Phase 2 — Distributed worker model
Phase 3 — Grafana dashboard and runtime verification
Phase 4 — Report generation and baseline comparison
Phase 5 — Kubernetes deployment and CI smoke testing
Phase 6 — Production receiver integration and SLO enforcement

---

## Slide 15 — Closing
USSD Performance Engine provides a scalable, measurable, and repeatable way to test and monitor critical transaction flows.

We built the foundation for reliable performance engineering, live observability, and regression-aware operations.

Thank you.
