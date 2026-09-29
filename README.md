# USSD Performance Engine

Stage 1 provides a seeded USSD session state machine with YAML configuration,
MNO-aware MSISDN generation, journey selection, and an injectable HTTP sender.
Stage 2 adds Prometheus counters and latency histograms plus a local Docker
Compose stack.

## Run locally

Install Go 1.22 or newer, then run the mock receiver mode:

```powershell
go test ./...
go run ./cmd/engine -config config/example.yaml -mock -sessions 3
```

The mock receiver returns `END`, so each session completes after the first
configured request. Replace `-mock` with a real endpoint in the YAML when the
sender contract is available. `${SENDER_API_KEY}` is expanded from the process
environment and is never stored as a literal secret by the loader.

Stage 1 runs sessions serially. Stage 3 adds deterministic distributed workers,
and Stage 4 adds a provisioned Grafana dashboard. Ramp profiles and the
remaining reporting features are planned in `DESIGN.md`.

## Run the Stage 2 stack

With Docker Desktop running:

```powershell
docker compose up --build
```

The engine exposes metrics at `http://localhost:2112/metrics`, Prometheus is
available at `http://localhost:9090`, and Grafana is available at
`http://localhost:3000`. Grafana automatically loads the **USSD Performance
Overview** dashboard. Useful initial queries are `ussd_requests_total` and
`rate(ussd_request_duration_seconds_count[1m])`.

## Test the real HTTP API

The API contract is documented in [`openapi.yaml`](openapi.yaml). The runnable
Postman collection used by CI is `postman/ci/performance-engine.postman_collection.json`;
it uses `base_url=http://127.0.0.1:8080`. The
API key is only needed when the API is started with `-api-key`; keep its value
in a local Postman environment or CI secret rather than committing it.

The GitHub Actions workflow runs the collection with Newman against a local API
instance. To sync that collection to an existing Postman workspace collection,
add repository secrets `POSTMAN_API_KEY` and `POSTMAN_COLLECTION_ID`. Sync runs
on pushes to `main`; it updates the configured collection in place. The
collection ID identifies its workspace, so no separate workspace ID is needed.

Start the API service locally:

```powershell
go run ./cmd/api -listen-addr :8080
```

In Postman, create a `POST` request to
`http://localhost:8080/api/v1/receive` with `Content-Type: application/json`
and this body:

```json
{
	"sessionId": "postman-demo-1",
	"msisdn": "2348031000001",
	"network": "MTN",
	"ussdString": "*737#",
	"input": "",
	"stage": "begin"
}
```

The receiver returns a terminal response:

```json
{"sessionId":"postman-demo-1","response":"END OK","continue":false}
```

To test the engine's outbound sender path, send the same body to
`http://localhost:8080/api/v1/send`. That endpoint forwards the request to the
receiver endpoint and returns its response. `GET /healthz` checks service
availability. Add `-api-key <value>` when starting the service to require an
`X-API-Key` header on the API routes.

To run the consolidated stack, use Docker Compose:

```powershell
docker compose up -d --build
```

The flow is then `Postman -> API :8080 -> receiver`, while the engine sends
load to the same receiver and exposes metrics on `:2112` for Prometheus and
Grafana. Import the requests under `postman/collections/Performance Engine`
and use the `Globals` environment with `base_url=http://localhost:8080`.

## Run distributed workers

The coordinator creates concurrent workers and assigns sessions using
deterministic round-robin sharding:

```powershell
go run ./cmd/engine -role coordinator -config config/example.yaml -mock -sessions 10 -worker-count 2
```

For independently launched workers, use `-role worker` with the same session
count, a distinct zero-based `-worker-id`, and the shared `-worker-count`.
Together, workers execute each session number exactly once. CLI worker flags
override the YAML values; omitted worker settings preserve single-worker
behavior.

To run separate coordinator and worker processes over HTTP:

```powershell
go run ./cmd/engine -role coordinator -config config/example.yaml -sessions 10 -worker-count 2 -coordinator-http-addr :8081
go run ./cmd/engine -role worker -config config/example.yaml -mock -worker-id 0 -worker-count 2 -coordinator-url http://127.0.0.1:8081
go run ./cmd/engine -role worker -config config/example.yaml -mock -worker-id 1 -worker-count 2 -coordinator-url http://127.0.0.1:8081
```

