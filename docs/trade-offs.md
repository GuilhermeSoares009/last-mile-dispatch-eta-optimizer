# Trade-offs

- Heuristica vs otimizacao: heuristica no MVP
- Dispatch centralizado vs regional: centralizado inicialmente
- Push vs pull assignment: push para reduzir latencia
- gRPC vs REST: gRPC interno, REST externo
- Cache vs banco: cache para leitura rapida
- Consistencia forte vs baixa latencia: eventual para ETA
