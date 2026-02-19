# AGENTS.md

## Setup commands
- Go deps: `go mod tidy`
- Java deps: `./mvnw -q -DskipTests package`
- Start Go service: `go run ./services/dispatch-engine-go/cmd/api`
- Start Java service: `./mvnw -pl services/policy-api-java spring-boot:run`
- Run tests: `go test ./...` and `./mvnw test`

## Code style
- Go: gofmt, golangci-lint
- Java: Java 21, Spring Boot, constructor injection
- Contratos gRPC versionados

## Arquitetura
- API externa /api/v1
- Dispatch engine em Go
- Policy service em Java
- gRPC interno

## Padrões de logging
- JSON com traceId, orderId, assignmentId

## Estrategia de testes
- Unitarios para heuristica e regras
- Integracao com Postgres e Redis
- Contratos gRPC

## Regras de seguranca
- Validacao de pedidos
- Rate limiting
- Auditoria de alteracoes

## Checklist de PR
- Lint e tests ok
- Docs atualizadas
- ADR se mudar regras

## Diretrizes de performance
- p95 /assignments < 80ms
- Timeouts curtos em gRPC
