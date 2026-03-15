package httpapi

import (
	"sync"
	"time"
)

type metricsStore struct {
	mu     sync.Mutex
	routes map[string]*routeMetric
}

type routeMetric struct {
	Requests       int64 `json:"requests"`
	BudgetExceeded int64 `json:"budgetExceeded"`
	TotalMs        int64
}

type routeMetricsResponse struct {
	Route          string `json:"route"`
	Requests       int64  `json:"requests"`
	BudgetExceeded int64  `json:"budgetExceeded"`
	AvgDurationMs  int64  `json:"avgDurationMs"`
}

type metricsResponse struct {
	GeneratedAt time.Time              `json:"generatedAt"`
	Routes      []routeMetricsResponse `json:"routes"`
}

func newMetricsStore() *metricsStore {
	return &metricsStore{routes: make(map[string]*routeMetric)}
}

func (m *metricsStore) Record(route string, durationMs int64, budgetExceeded bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metric, ok := m.routes[route]
	if !ok {
		metric = &routeMetric{}
		m.routes[route] = metric
	}
	metric.Requests++
	metric.TotalMs += durationMs
	if budgetExceeded {
		metric.BudgetExceeded++
	}
}

func (m *metricsStore) Snapshot() metricsResponse {
	m.mu.Lock()
	defer m.mu.Unlock()

	routes := make([]routeMetricsResponse, 0, len(m.routes))
	for route, metric := range m.routes {
		avg := int64(0)
		if metric.Requests > 0 {
			avg = metric.TotalMs / metric.Requests
		}
		routes = append(routes, routeMetricsResponse{
			Route:          route,
			Requests:       metric.Requests,
			BudgetExceeded: metric.BudgetExceeded,
			AvgDurationMs:  avg,
		})
	}

	return metricsResponse{
		GeneratedAt: time.Now().UTC(),
		Routes:      routes,
	}
}
