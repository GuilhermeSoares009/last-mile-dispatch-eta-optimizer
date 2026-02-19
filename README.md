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

- Healthcheck: http://localhost:8085/api/v1/health

## Contratos de API
- OpenAPI: docs/api/openapi.yaml

## Documentacao
- Project Reference Guide: PROJECT_REFERENCE_GUIDE.md
- Especificacoes: docs/specs/spec.md
- Plano tecnico: docs/specs/plan.md
- Tarefas: docs/specs/tasks.md
- ADRs: docs/adr/
- Trade-offs: docs/trade-offs.md
- Threat model: docs/threat-model.md
- Performance budget: docs/performance-budget.md
- Feature flags: docs/feature-flags.md
- Legacy spec (arquivado): docs/legacy-spec/
