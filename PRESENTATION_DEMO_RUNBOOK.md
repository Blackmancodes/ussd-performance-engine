# Presentation Demo Runbook

## Demo flow

Show a Postman request handled by the API, then show the separate one-million-session mock workload and its Prometheus/Grafana metrics. The Compose engine uses an in-process mock receiver; it does not load-test the API container.

## 1. Start the stack

From PowerShell in the project directory, with Docker Desktop running:

```powershell
docker compose up -d --build
docker compose ps
```

Wait until `api`, `engine`, `prometheus`, and `grafana` are running. The demo uses:

| Service | URL |
| --- | --- |
| API | `http://localhost:8080` |
| Engine metrics | `http://localhost:2112/metrics` |
| Prometheus | `http://localhost:9090` |
| Grafana | `http://localhost:3000` |

## 2. Check the API and send a Postman request

Check health:

```powershell
Invoke-RestMethod http://localhost:8080/healthz
```

Expected: `status` is `ok`.

In the `Performance Engine` Postman collection, set `base_url` to `http://localhost:8080`. Run **Receive**, then **Send**. Both use the collection's demo payload and should return `END OK` with `continue: false`. **Send** forwards to the API's local simulated receiver.

## 3. Show the running workload

Compose starts a configured batch of 1,000,000 sessions across 2 workers, with a mock receiver and a 10% simulated error rate. These settings describe the demo workload; they are not a capacity benchmark. The error rate means not all sessions should be expected to complete.

Open `http://localhost:2112/metrics` and point out:

- `ussd_requests_total`
- `ussd_request_duration_seconds`
- `ussd_sessions_total`

## 4. Verify Prometheus and Grafana

Check that Prometheus is scraping the engine:

```powershell
Invoke-RestMethod 'http://localhost:9090/api/v1/query?query=up%7Bjob%3D%22ussd-engine%22%7D' |
  ConvertTo-Json -Depth 6
```

The result should contain value `1` for `job="ussd-engine"`.

Open `http://localhost:3000` and select **USSD Pulse Command Center**. Show request rate, error rate, p95 latency, completed/failed sessions, traffic by journey/network, and latency percentiles. In **Alerting → Alert rules**, point out **USSD high error rate** and **USSD engine metrics unavailable**; these rules may be Normal unless their conditions are met.

## 5. Wrap up

Say: “Postman verifies the API flow. Separately, the engine runs a repeatable mock workload, Prometheus scrapes its metrics, and Grafana shows throughput, latency, and session outcomes.”

## If something fails

```powershell
docker compose ps
docker compose logs --tail=40 api engine prometheus grafana
```

Check that ports `8080`, `2112`, `9090`, and `3000` are available. If a previous local API process owns port `8080`, stop it and rerun `docker compose up -d --build`.
