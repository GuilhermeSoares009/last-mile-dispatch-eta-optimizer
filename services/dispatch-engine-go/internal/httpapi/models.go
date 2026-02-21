package httpapi

import (
    "errors"
    "strconv"
    "strings"

    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/audit"
    "github.com/GuilhermeSoares009/last-mile-dispatch-eta-optimizer/dispatch-engine/internal/dispatch"
)

type dispatchRequest struct {
    RequestID string          `json:"requestId"`
    Order     orderRequest    `json:"order"`
    Couriers  []courierInput  `json:"couriers"`
    Priority  string          `json:"priority"`
}

type orderRequest struct {
    ID      string `json:"id"`
    Pickup  string `json:"pickup"`
    Dropoff string `json:"dropoff"`
}

type courierInput struct {
    ID           string  `json:"id"`
    EtaMinutes   int64   `json:"etaMinutes"`
    Availability float64 `json:"availability"`
    Load         float64 `json:"load"`
}

type dispatchResponse struct {
    RequestID string             `json:"requestId"`
    TraceID   string             `json:"traceId"`
    Decision  dispatchDecision   `json:"decision"`
}

type dispatchDecision struct {
    CourierID  string  `json:"courierId"`
    EtaMinutes int64   `json:"etaMinutes"`
    Fallback   bool    `json:"fallback"`
    Reason     string  `json:"reason"`
    Score      float64 `json:"score"`
}

type auditResponse struct {
    Entries []audit.Entry `json:"entries"`
}

type errorResponse struct {
    Error string `json:"error"`
}

func (req dispatchRequest) Validate() error {
    if strings.TrimSpace(req.Order.ID) == "" {
        return errors.New("order.id is required")
    }
    if strings.TrimSpace(req.Order.Pickup) == "" {
        return errors.New("order.pickup is required")
    }
    if strings.TrimSpace(req.Order.Dropoff) == "" {
        return errors.New("order.dropoff is required")
    }
    if len(req.Couriers) == 0 {
        return errors.New("couriers must include at least one courier")
    }
    for idx, courier := range req.Couriers {
        if strings.TrimSpace(courier.ID) == "" {
            return errors.New("couriers[" + strconv.Itoa(idx) + "].id is required")
        }
        if courier.EtaMinutes <= 0 {
            return errors.New("couriers[" + strconv.Itoa(idx) + "].etaMinutes must be > 0")
        }
        if courier.Availability < 0 || courier.Availability > 1 {
            return errors.New("couriers[" + strconv.Itoa(idx) + "].availability must be between 0 and 1")
        }
        if courier.Load < 0 || courier.Load > 1 {
            return errors.New("couriers[" + strconv.Itoa(idx) + "].load must be between 0 and 1")
        }
    }
    return nil
}

func (req dispatchRequest) CouriersToDispatch() []dispatch.Courier {
    couriers := make([]dispatch.Courier, 0, len(req.Couriers))
    for _, courier := range req.Couriers {
        couriers = append(couriers, dispatch.Courier{
            ID:           courier.ID,
            EtaMinutes:   courier.EtaMinutes,
            Availability: courier.Availability,
            Load:         courier.Load,
        })
    }
    return couriers
}

func parseLimit(raw string, fallback int) int {
    if raw == "" {
        return fallback
    }
    value, err := strconv.Atoi(raw)
    if err != nil || value <= 0 {
        return fallback
    }
    return value
}
