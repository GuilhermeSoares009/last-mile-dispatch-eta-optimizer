# Last-Mile Dispatch and ETA Optimizer

Two services that decide which courier should take a delivery: a dispatch engine in Go that ranks couriers and picks one, and a policy API in Java and Spring Boot that can adjust that ranking before the choice is made.

## Architecture
- dispatch-engine (Go) exposes the dispatch API, scores couriers and records every decision.
- policy-api (Java, Spring Boot) evaluates business policy and returns score adjustments per courier.
- The engine calls the policy service over HTTP with an explicit timeout, so a slow or failing policy service does not hold the dispatch request open.

## How a courier is chosen
- Each candidate carries an ETA in minutes, an availability figure and a current load.
- Couriers below the minimum availability threshold (0.5) are discarded.
- The remaining ones are scored from ETA, availability and load, and the policy service can override a courier score before the comparison.
- When no courier is available the engine still returns the best of the unavailable set, flagged as a fallback with the reason fallback-no-available-couriers.
- The response carries the chosen courier, the score, the fallback flag and the reason.

## Dispatch engine API
- GET /api/v1/health - liveness check
- POST /api/v1/dispatch - submit candidate couriers and receive the decision
- GET /api/v1/audit/dispatch - recent dispatch decisions
- GET /api/v1/metrics/dispatch - request metrics

Requests are rate limited per client.

## Running locally

```bash
docker compose up -d --build
curl http://localhost:8084/api/v1/health
```

Dispatch engine on port 8084, policy API on 8085, PostgreSQL on 5436, Redis on 6383.

## Tests

```bash
cd services/dispatch-engine-go && go test ./...
```

## Stack

Go, Java with Spring Boot, Docker Compose and GitHub Actions. PostgreSQL and Redis are provisioned in the local stack; the audit trail is currently kept in memory.
