# Plan: Last-Mile Dispatch and ETA Optimizer

## Architecture
- Go dispatch engine for low latency assignment.
- Java policy API for rules and external access.
- gRPC internal communication.
- Postgres for orders and assignments.
- Redis for candidate and ETA caches.

## Dispatch Logic
- Score candidates by distance, load, and SLA.
- Fallback to secondary candidates when needed.

## Observability
- JSON logs with traceId, orderId, assignmentId.
- OpenTelemetry traces across Go and Java services.

## Security
- Input validation for orders and candidates.
- Rate limiting on assignment endpoints.
- Audit trail for assignment changes.

## Feature Flags
- dispatch_algo_v2
- rebalance_enabled
- eta_model_v2

## Local Dev and CI
- Docker Compose for Postgres, Redis, both services.
- CI runs Go and Java tests.

## ADRs
- Hybrid Go + Java architecture
- Heuristic-first approach
