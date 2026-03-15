package httpapi

import (
    "net/http"
    "time"

    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/audit"
    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/policy"
    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/ratelimit"
)

type Server struct {
	limiter      *ratelimit.Limiter
	policyClient *policy.Client
	auditStore   *audit.Store
	metrics      *metricsStore
	mux          *http.ServeMux
}

func NewServer(limiter *ratelimit.Limiter, policyClient *policy.Client, auditStore *audit.Store) *Server {
	server := &Server{
		limiter:      limiter,
		policyClient: policyClient,
		auditStore:   auditStore,
		metrics:      newMetricsStore(),
		mux:          http.NewServeMux(),
	}
    server.routes()
    return server
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/v1/health", s.handleHealth)
	s.mux.HandleFunc("/api/v1/dispatch", s.handleDispatch)
	s.mux.HandleFunc("/api/v1/audit/dispatch", s.handleAudit)
	s.mux.HandleFunc("/api/v1/metrics/dispatch", s.handleMetrics)
}

func (s *Server) Handler() http.Handler {
	return s.withMetrics(s.withRateLimit(s.mux))
}

func (s *Server) withRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.limiter.Allow(clientIP(r), time.Now()) {
			writeJSON(w, http.StatusTooManyRequests, errorResponse{Error: "rate limit exceeded"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		durationMs := time.Since(start).Milliseconds()
		s.metrics.Record(r.URL.Path, durationMs, durationMs > latencyBudgetMs)
	})
}
