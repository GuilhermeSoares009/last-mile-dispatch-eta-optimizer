package ratelimit

import (
    "sync"
    "time"
)

type Limiter struct {
    mu          sync.Mutex
    window      time.Duration
    maxRequests int
    buckets     map[string]*bucket
}

type bucket struct {
    count       int
    windowStart time.Time
}

func NewLimiter(maxRequests int, window time.Duration) *Limiter {
    return &Limiter{
        window:      window,
        maxRequests: maxRequests,
        buckets:     make(map[string]*bucket),
    }
}

func (l *Limiter) Allow(key string, now time.Time) bool {
    l.mu.Lock()
    defer l.mu.Unlock()

    current, ok := l.buckets[key]
    if !ok {
        l.buckets[key] = &bucket{count: 1, windowStart: now}
        return true
    }

    if now.Sub(current.windowStart) >= l.window {
        current.windowStart = now
        current.count = 1
        return true
    }

    if current.count >= l.maxRequests {
        return false
    }

    current.count++
    return true
}
