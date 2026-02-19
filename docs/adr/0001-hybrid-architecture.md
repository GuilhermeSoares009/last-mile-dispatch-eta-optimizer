# ADR 0001 - Arquitetura Hibrida Go + Java

## Status
Aceito

## Contexto
Baixa latencia para dispatch e regras complexas para politicas.

## Decisao
Go para dispatch e Java para politicas, integrados via gRPC.

## Consequencias
- Melhor performance no core
- Maior complexidade de deploy
