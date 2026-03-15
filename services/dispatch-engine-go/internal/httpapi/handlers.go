package httpapi

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "errors"
    "io"
    "net"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/audit"
    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/dispatch"
    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/policy"
)

const (
    maxBodySize     = 1 << 20
    latencyBudgetMs = 100
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
    traceID := newID()
    writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
    logRequest(r.Context(), logEntry{
        Message:     "health check",
        TraceID:     traceID,
        RequestID:   newID(),
        CourierID:   "",
        Status:      http.StatusOK,
        Path:        r.URL.Path,
        Method:      r.Method,
        BudgetMs:    latencyBudgetMs,
    })
}

func (s *Server) handleDispatch(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    traceID := newID()

    if r.Method != http.MethodPost {
        writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
        logRequest(r.Context(), logEntry{
            Message:     "method not allowed",
            TraceID:     traceID,
            RequestID:   newID(),
            CourierID:   "",
            Status:      http.StatusMethodNotAllowed,
            Path:        r.URL.Path,
            Method:      r.Method,
            BudgetMs:    latencyBudgetMs,
        })
        return
    }

    var payload dispatchRequest
    if err := readJSON(r, &payload); err != nil {
        writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
        logRequest(r.Context(), logEntry{
            Message:     "invalid request",
            TraceID:     traceID,
            RequestID:   newID(),
            CourierID:   "",
            Status:      http.StatusBadRequest,
            Path:        r.URL.Path,
            Method:      r.Method,
            BudgetMs:    latencyBudgetMs,
        })
        return
    }

    if err := payload.Validate(); err != nil {
        writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
        logRequest(r.Context(), logEntry{
            Message:     "validation failed",
            TraceID:     traceID,
            RequestID:   newID(),
            CourierID:   "",
            Status:      http.StatusBadRequest,
            Path:        r.URL.Path,
            Method:      r.Method,
            BudgetMs:    latencyBudgetMs,
        })
        return
    }

    requestID := payload.RequestID
    if strings.TrimSpace(requestID) == "" {
        requestID = newID()
    }

    couriers := payload.CouriersToDispatch()
    policyApplied := false
    policyTraceID := ""
    adjustedScores := make(map[string]float64)
    if s.policyClient != nil {
        candidates := make([]policy.Candidate, 0, len(couriers))
        for _, courier := range couriers {
            candidates = append(candidates, policy.Candidate{
                CourierID: courier.ID,
                BaseScore: dispatch.ScoreCourier(courier),
            })
        }
        ctx, cancel := context.WithTimeout(r.Context(), 120*time.Millisecond)
        defer cancel()
        response, err := s.policyClient.Evaluate(ctx, policy.EvaluateRequest{
            RequestID:  requestID,
            Priority:   payload.Priority,
            Candidates: candidates,
        })
        if err == nil {
            policyApplied = true
            policyTraceID = response.TraceID
            for _, adjustment := range response.Adjustments {
                adjustedScores[adjustment.CourierID] = adjustment.AdjustedScore
            }
        }
    }

    var decision dispatch.Decision
    var err error
    if policyApplied {
        decision, err = dispatch.SelectWithScores(couriers, adjustedScores)
    } else {
        decision, err = dispatch.SelectCourier(couriers)
    }
    if err != nil {
        status := http.StatusInternalServerError
        message := "dispatch decision failed"
        if errors.Is(err, dispatch.ErrNoCouriers) {
            status = http.StatusBadRequest
            message = "no couriers provided"
        }
        writeJSON(w, status, errorResponse{Error: message})
        logRequest(r.Context(), logEntry{
            Message:     message,
            TraceID:     traceID,
            RequestID:   requestID,
            CourierID:   "",
            Status:      status,
            Path:        r.URL.Path,
            Method:      r.Method,
            BudgetMs:    latencyBudgetMs,
        })
        return
    }

    response := dispatchResponse{
        RequestID: requestID,
        TraceID:   traceID,
        Decision: dispatchDecision{
            CourierID: decision.Courier.ID,
            EtaMinutes: decision.Courier.EtaMinutes,
            Fallback:  decision.Fallback,
            Reason:    decision.Reason,
            Score:     decision.Score,
        },
    }

    writeJSON(w, http.StatusOK, response)

    durationMs := time.Since(start).Milliseconds()
    if s.auditStore != nil {
        s.auditStore.Add(audit.Entry{
            Timestamp:     time.Now().UTC(),
            RequestID:     requestID,
            CourierID:     decision.Courier.ID,
            Score:         decision.Score,
            Reason:        decision.Reason,
            Fallback:      decision.Fallback,
            PolicyApplied: policyApplied,
            PolicyTraceID: policyTraceID,
        })
    }
    logRequest(r.Context(), logEntry{
        Message:        "dispatch decision",
        TraceID:        traceID,
        RequestID:      requestID,
        CourierID:      decision.Courier.ID,
        Status:         http.StatusOK,
        Path:           r.URL.Path,
        Method:         r.Method,
        DurationMs:     durationMs,
        BudgetMs:       latencyBudgetMs,
        BudgetExceeded: durationMs > latencyBudgetMs,
        Fallback:       decision.Fallback,
        PolicyApplied:  policyApplied,
        PolicyTraceID:  policyTraceID,
    })
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	if s.auditStore == nil {
		writeJSON(w, http.StatusOK, auditResponse{Entries: []audit.Entry{}})
		return
	}
	limit := parseLimit(r.URL.Query().Get("limit"), 50)
	writeJSON(w, http.StatusOK, auditResponse{Entries: s.auditStore.List(limit)})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}
	writeJSON(w, http.StatusOK, s.metrics.Snapshot())
}

func readJSON(r *http.Request, dst any) error {
    decoder := json.NewDecoder(io.LimitReader(r.Body, maxBodySize))
    decoder.DisallowUnknownFields()
    if err := decoder.Decode(dst); err != nil {
        return err
    }
    if err := decoder.Decode(&struct{}{}); err != io.EOF {
        return errors.New("unexpected data after json body")
    }
    return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(payload)
}

func clientIP(r *http.Request) string {
    if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
        parts := strings.Split(forwarded, ",")
        return strings.TrimSpace(parts[0])
    }
    if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
        return strings.TrimSpace(realIP)
    }
    host, _, err := net.SplitHostPort(r.RemoteAddr)
    if err == nil {
        return host
    }
    return r.RemoteAddr
}

func newID() string {
    bytes := make([]byte, 16)
    if _, err := rand.Read(bytes); err != nil {
        return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000")))
    }
    return hex.EncodeToString(bytes)
}

type logEntry struct {
    Message        string `json:"message"`
    TraceID        string `json:"traceId"`
    RequestID      string `json:"requestId"`
    CourierID      string `json:"courierId"`
    Status         int    `json:"status"`
    Path           string `json:"path"`
    Method         string `json:"method"`
    DurationMs     int64  `json:"durationMs,omitempty"`
    BudgetMs       int64  `json:"budgetMs"`
    BudgetExceeded bool   `json:"budgetExceeded,omitempty"`
    Fallback       bool   `json:"fallback,omitempty"`
    PolicyApplied  bool   `json:"policyApplied,omitempty"`
    PolicyTraceID  string `json:"policyTraceId,omitempty"`
}

func logRequest(_ context.Context, entry logEntry) {
    data, err := json.Marshal(entry)
    if err != nil {
        return
    }
    _, _ = os.Stdout.Write(append(data, '\n'))
}
