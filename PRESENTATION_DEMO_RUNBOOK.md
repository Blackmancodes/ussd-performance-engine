# Presentation Demo Runbook

## Demo goal

Show one complete path from a Postman request to the API receiver, then show the
engine generating a one-million-session batch whose metrics are scraped by
Prometheus and displayed in Grafana.

## Before presenting

Use Windows PowerShell from the project directory. If you are using Command
Prompt, use the `curl` alternatives shown below instead of
`Invoke-WebRequest` and `Invoke-RestMethod`.

```powershell
cd "C:\Users\ASUS\Performance Engine"
```

Confirm Docker Desktop is running. Close any manually started API process that
already owns port `8080` before starting Compose.

## Step 1: Start the complete system

```powershell
docker compose up -d --build
```

Expected result: four containers are running:

- `api` on `http://localhost:8080`
- `engine` on `http://localhost:2112`
- `prometheus` on `http://localhost:9090`
- `grafana` on `http://localhost:3000`

Show the running services:

```powershell
docker compose ps
```

## Step 2: Prove the API is available

```powershell
Invoke-WebRequest http://localhost:8080/healthz -UseBasicParsing
```

Command Prompt alternative:

```bat
curl http://localhost:8080/healthz
```

Expected response:

```json
{"status":"ok"}
```

Say: "The API is live before we send traffic."

## Step 3: Send a real Postman transaction

In Postman, select the `Performance Engine` collection and set:

```text
base_url = http://localhost:8080
```

Run `Receive` first, then `Send`.

Use this JSON body:

```json
{
  "sessionId": "presentation-demo-1",
  "msisdn": "2348031000001",
  "network": "MTN",
  "ussdString": "*737#",
  "input": "",
  "stage": "begin"
}
```

Expected response from both requests:

```json
{
  "sessionId": "presentation-demo-1",
  "response": "END OK",
  "continue": false
}
```

Say: "The send endpoint forwards the session payload to the receiver and
returns the receiver response."

## Step 4: Show the engine workload

The Compose engine is configured for one million sessions and two workers.
Show its live metrics endpoint:

```powershell
Invoke-WebRequest http://localhost:2112/metrics -UseBasicParsing
```

Command Prompt alternative:

```bat
curl http://localhost:2112/metrics
```

Point out these metrics in the response:

- `ussd_requests_total`
- `ussd_request_duration_seconds`
- `ussd_sessions_total`

## Step 5: Show Prometheus is scraping the engine

```powershell
Invoke-RestMethod "http://localhost:9090/api/v1/query?query=up" | ConvertTo-Json -Depth 6
```

Command Prompt alternative:

```bat
curl "http://localhost:9090/api/v1/query?query=up"
```

Expected target result includes:

```text
job=ussd-engine
value=1
```

Say: "The metrics are not just local output; Prometheus is actively scraping
the engine."

## Step 6: Open Grafana

Open:

```text
http://localhost:3000
```

Open the `USSD Performance Overview` dashboard.

Show these cards and panels:

- Request Rate
- Success Rate
- Failed Sessions
- Completed Sessions
- P95 Request Latency
- Request Rate Over Time
- Latency Percentiles
- Sessions by State

Expected result: the values move while the batch is running. After completion,
the cumulative totals remain visible.

## Step 7: Show the alerting capability

In Grafana, open **Alerting -> Alert rules** and show:

- `USSD high error rate`
- `USSD engine metrics unavailable`

Explain:

- The high-error alert evaluates the five-minute error ratio.
- The engine-down alert evaluates the Prometheus `up` metric.
- Rules are provisioned from `deploy/grafana/provisioning/alerting/alert-rules.yml`.

## Step 8: Prove the final totals

Run:

```powershell
$metrics = Invoke-WebRequest http://localhost:2112/metrics -UseBasicParsing
($metrics.Content -split "`n" | Select-String "ussd_requests_total|ussd_sessions_total")
```

Command Prompt alternative:

```bat
curl http://localhost:2112/metrics | findstr "ussd_requests_total ussd_sessions_total"
```

For the standard Compose run, the final completed total should reach
`1e+06` or `1000000`, depending on Prometheus text formatting.

## Presenter closing statement

"We started one reproducible, one-million-session workload. A real request was
sent through Postman, the engine sent traffic to the same receiver, Prometheus
scraped the engine metrics, and Grafana displayed throughput, latency, success,
and failure outcomes. The result is observable from request to dashboard."

## If something fails during the demo

Check the stack:

```powershell
docker compose ps
docker compose logs --tail=40 api engine prometheus grafana
```

Check the required ports:

```powershell
Get-NetTCPConnection -State Listen -ErrorAction SilentlyContinue |
  Where-Object { $_.LocalPort -in 8080,2112,9090,3000 }
```

If port `8080` is owned by a previous local API process, stop that process and
rerun:

```powershell
docker compose up -d --build
```
