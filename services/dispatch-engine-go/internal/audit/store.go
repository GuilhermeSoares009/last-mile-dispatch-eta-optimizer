package audit

import (
    "sync"
    "time"
)

type Entry struct {
    Timestamp     time.Time `json:"timestamp"`
    RequestID     string    `json:"requestId"`
    CourierID     string    `json:"courierId"`
    Score         float64   `json:"score"`
    Reason        string    `json:"reason"`
    Fallback      bool      `json:"fallback"`
    PolicyApplied bool      `json:"policyApplied"`
    PolicyTraceID string    `json:"policyTraceId"`
}

type Store struct {
    mu      sync.Mutex
    entries []Entry
}

func NewStore() *Store {
    return &Store{entries: make([]Entry, 0, 200)}
}

func (s *Store) Add(entry Entry) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.entries = append(s.entries, entry)
    if len(s.entries) > 1000 {
        s.entries = s.entries[len(s.entries)-1000:]
    }
}

func (s *Store) List(limit int) []Entry {
    s.mu.Lock()
    defer s.mu.Unlock()

    if limit <= 0 {
        limit = 50
    }
    if limit > len(s.entries) {
        limit = len(s.entries)
    }
    start := len(s.entries) - limit
    if start < 0 {
        start = 0
    }
    result := make([]Entry, 0, limit)
    for i := len(s.entries) - 1; i >= start; i-- {
        result = append(result, s.entries[i])
    }
    return result
}
