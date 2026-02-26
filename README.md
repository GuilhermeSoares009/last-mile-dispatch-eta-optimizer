# Last-Mile Dispatch and ETA Optimizer

Sistema hibrido de dispatch que aloca entregadores e recalcula ETA com baixa latencia e auditabilidade.

## Capacidades-chave
- Decisoes de alocacao com baixa latencia (motor Go)
- API de politicas e regras (servico Java)
- Trilha de auditoria para mudancas de alocacao
- Logs estruturados e tracing com OpenTelemetry

## Inicio rapido (Docker)
```bash
docker compose up --build
```

- Inclui `docker-compose.yml` e Dockerfile(s).
- Healthcheck: http://localhost:8085/api/v1/health

## API (MVP)

### Dispatch engine (porta 8084)

- `GET /api/v1/health`
- `POST /api/v1/dispatch`
- `GET /api/v1/audit/dispatch`
- `GET /api/v1/metrics/dispatch`

### Policy API (porta 8085)

- `GET /api/v1/health`
- `POST /api/v1/policies/evaluate`
- `GET /api/v1/policies/config`
- `POST /api/v1/policies/config`
- `GET /api/v1/audit/policies`
- `GET /api/v1/audit/policies/config`
- `GET /api/v1/metrics`

### Variaveis de ambiente

- `PORT` (default: 8080)
- `RATE_LIMIT_PER_MIN` (default: 120)
- `POLICY_API_URL` (default: http://policy-api:8080)
- `POLICY_TIMEOUT_MS` (default: 100)
