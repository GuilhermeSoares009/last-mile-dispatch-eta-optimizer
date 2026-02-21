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

### Policy API (porta 8085)

- `GET /api/v1/health`
- `POST /api/v1/policies/evaluate`

### Variaveis de ambiente

- `PORT` (default: 8080)
- `RATE_LIMIT_PER_MIN` (default: 120)

## Qualidade (pre-commit)
Este repositorio usa pre-commit para CR + auditoria ASVS (OWASP ASVS v5.0.0) antes de cada commit.

```bash
pip install pre-commit
pre-commit install
```

Para rodar manualmente:

```bash
pre-commit run --all-files
```
