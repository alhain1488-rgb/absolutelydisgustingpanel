package auth

import (
	"sync"
	"time"
)

// RateLimiter is a small fixed-window limiter keyed by an arbitrary string
// (e.g. client IP). It is safe for concurrent use.
type RateLimiter struct {
	mu       sync.Mutex
	hits     map[string]*window
	limit    int
	interval time.Duration
	now      func() time.Time
}

type window struct {
	count int
	reset time.Time
}

// NewRateLimiter allows up to limit events per interval per key.
func NewRateLimiter(limit int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		hits:     make(map[string]*window),
		limit:    limit,
		interval: interval,
		now:      time.Now,
	}
}

// Allow reports whether an event for key is permitted, counting it if so.
func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	w, ok := l.hits[key]
	if !ok || now.After(w.reset) {
		l.hits[key] = &window{count: 1, reset: now.Add(l.interval)}
		return true
	}
	if w.count >= l.limit {
		return false
	}
	w.count++
	return true
}
