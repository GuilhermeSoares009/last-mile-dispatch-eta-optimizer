# Spec: Last-Mile Dispatch and ETA Optimizer (MVP)

## Product Vision
- Allocate orders to couriers with low latency and stable SLA outcomes.
- Recalculate ETA as conditions change.
- Provide auditability and operational visibility.

## User Scenarios and Testing

### User Story 1 - Assign a courier quickly (Priority: P1)
As an operations system, I want an assignment decision under strict latency budgets.

**Why this priority**: Dispatch is real-time and latency sensitive.

**Independent Test**: Submit an assignment request and verify p95 latency under 80ms in local tests.

**Acceptance Scenarios**:
1. **Given** a list of candidates, **When** I request an assignment, **Then** I receive the chosen courier and ETA.

---

### User Story 2 - Recalculate ETA (Priority: P2)
As a dispatcher, I want ETA recalculated when conditions change.

**Why this priority**: ETA accuracy impacts customer trust.

**Independent Test**: Trigger ETA recalculation and verify updated ETA is returned.

**Acceptance Scenarios**:
1. **Given** a delay event, **When** I request ETA recalculation, **Then** a new ETA is returned.

---

### User Story 3 - Audit assignment changes (Priority: P3)
As a compliance owner, I want assignment changes recorded for investigation.

**Why this priority**: Auditability enables governance.

**Independent Test**: Update assignment and verify audit entry exists.

**Acceptance Scenarios**:
1. **Given** a reassignment, **When** I query audit history, **Then** I see the change with actor and timestamp.

### Edge Cases
- No available candidates.
- Conflicting assignment requests for the same order.
- ETA recalculation for unknown order.

## Functional Requirements
- FR-01: Register orders and candidates.
- FR-02: Assign best courier using heuristic rules.
- FR-03: Recalculate ETA on events and reassignments.
- FR-04: Persist assignments and audit changes.
- FR-05: Expose /api/v1/health.

## Non-Functional Requirements
- NFR-01: p95 /api/v1/assignments < 80ms in local env.
- NFR-02: 800 requests/sec in local env.
- NFR-03: Structured JSON logs with traceId and orderId.
- NFR-04: OpenTelemetry traces across services.
- NFR-05: 100% Docker local environment.
- NFR-06: API versioned under /api/v1.

## Success Criteria
- SC-01: p95 assignment latency under 80ms in local tests.
- SC-02: ETA recalculation returns a new value for changed conditions.
- SC-03: Audit trail captures all assignment changes.

## API Contracts
- OpenAPI: .specify/specs/001-last-mile-dispatch/contracts/openapi.yaml

## Roadmap
- Milestone 1: Orders + assignment MVP.
- Milestone 2: Rebalance and ETA recalculation.
- Milestone 3: Observability and resilience.
- Milestone 4: Algorithm v2.

## Trade-offs
- Heuristic vs optimization.
- Centralized vs regional dispatch.
- Push vs pull assignment.
- gRPC vs REST.
- Cache vs database.
- Strong consistency vs low latency.
