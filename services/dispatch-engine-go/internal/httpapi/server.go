package httpapi

import (
    "net/http"
    "time"

    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/ratelimit"
)

type Server struct {
    limiter *ratelimit.Limiter
    mux     *http.ServeMux
}

func NewServer(limiter *ratelimit.Limiter) *Server {
    server := &Server{
        limiter: limiter,
        mux:     http.NewServeMux(),
    }
    server.routes()
    return server
}

func (s *Server) routes() {
    s.mux.HandleFunc("/api/v1/health", s.handleHealth)
    s.mux.HandleFunc("/api/v1/dispatch", s.handleDispatch)
}

func (s *Server) Handler() http.Handler {
    return s.withRateLimit(s.mux)
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
