package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "syscall"
    "time"

    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/audit"
    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/httpapi"
    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/policy"
    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/ratelimit"
)

const (
    defaultPort           = "8080"
    defaultRequestsPerMin = 120
    defaultPolicyTimeout  = 100
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    port := getenv("PORT", defaultPort)
    limit := parseInt(getenv("RATE_LIMIT_PER_MIN", ""), defaultRequestsPerMin)
    policyURL := getenv("POLICY_API_URL", "http://policy-api:8080")
    policyTimeout := time.Duration(parseInt(getenv("POLICY_TIMEOUT_MS", ""), defaultPolicyTimeout)) * time.Millisecond

    limiter := ratelimit.NewLimiter(limit, time.Minute)
    policyClient := policy.NewClient(policyURL, policyTimeout)
    auditStore := audit.NewStore()
    server := httpapi.NewServer(limiter, policyClient, auditStore)

    httpServer := &http.Server{
        Addr:              ":" + port,
        Handler:           server.Handler(),
        ReadTimeout:       5 * time.Second,
        ReadHeaderTimeout: 2 * time.Second,
        WriteTimeout:      5 * time.Second,
        IdleTimeout:       30 * time.Second,
    }

    go func() {
        if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("dispatch engine stopped unexpectedly: %v", err)
        }
    }()

    <-ctx.Done()

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    _ = httpServer.Shutdown(shutdownCtx)
}

func getenv(key, fallback string) string {
    value := os.Getenv(key)
    if value == "" {
        return fallback
    }
    return value
}

func parseInt(raw string, fallback int) int {
    if raw == "" {
        return fallback
    }
    value, err := strconv.Atoi(raw)
    if err != nil || value <= 0 {
        return fallback
    }
    return value
}
