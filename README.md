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

## Qualidade (pre-commit)
Este repositorio usa pre-commit para CR + auditoria + anti-slop antes de cada commit.

```bash
pip install pre-commit
pre-commit install
```

Para rodar manualmente:

```bash
pre-commit run --all-files
```
