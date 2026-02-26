package policy

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "net/http"
    "strings"
    "time"
)

type Client struct {
    baseURL    string
    httpClient *http.Client
}

type Candidate struct {
    CourierID string  `json:"courierId"`
    BaseScore float64 `json:"baseScore"`
}

type EvaluateRequest struct {
    RequestID  string      `json:"requestId"`
    Priority   string      `json:"priority"`
    Candidates []Candidate `json:"candidates"`
}

type Adjustment struct {
    CourierID     string  `json:"courierId"`
    AdjustedScore float64 `json:"adjustedScore"`
}

type EvaluateResponse struct {
    RequestID   string       `json:"requestId"`
    TraceID     string       `json:"traceId"`
    Multiplier  float64      `json:"multiplier"`
    Adjustments []Adjustment `json:"adjustments"`
}

func NewClient(baseURL string, timeout time.Duration) *Client {
    return &Client{
        baseURL: strings.TrimRight(baseURL, "/"),
        httpClient: &http.Client{
            Timeout: timeout,
        },
    }
}

func (c *Client) Evaluate(ctx context.Context, request EvaluateRequest) (EvaluateResponse, error) {
    if len(request.Candidates) == 0 {
        return EvaluateResponse{}, errors.New("policy candidates missing")
    }

    body, err := json.Marshal(request)
    if err != nil {
        return EvaluateResponse{}, err
    }

    url := fmt.Sprintf("%s/api/v1/policies/evaluate", c.baseURL)
    httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        return EvaluateResponse{}, err
    }
    httpRequest.Header.Set("Content-Type", "application/json")

    response, err := c.httpClient.Do(httpRequest)
    if err != nil {
        return EvaluateResponse{}, err
    }
    defer response.Body.Close()

    if response.StatusCode < 200 || response.StatusCode >= 300 {
        return EvaluateResponse{}, fmt.Errorf("policy api returned %d", response.StatusCode)
    }

    var parsed EvaluateResponse
    if err := json.NewDecoder(response.Body).Decode(&parsed); err != nil {
        return EvaluateResponse{}, err
    }
    return parsed, nil
}
