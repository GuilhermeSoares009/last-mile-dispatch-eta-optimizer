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

    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/dispatch"
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

    decision, err := dispatch.SelectCourier(payload.CouriersToDispatch())
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
    })
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
}

func logRequest(_ context.Context, entry logEntry) {
    data, err := json.Marshal(entry)
    if err != nil {
        return
    }
    _, _ = os.Stdout.Write(append(data, '\n'))
}
