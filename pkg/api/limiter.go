package api

import (
	"sync"
	"time"
)

type visitor struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]visitor
	rate     float64
	burst    float64
}

func newRateLimiter(ratePerMinute, burst int) *rateLimiter {
	return &rateLimiter{
		visitors: make(map[string]visitor),
		rate:     float64(ratePerMinute) / 60,
		burst:    float64(burst),
	}
}

func (l *rateLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	current, exists := l.visitors[key]
	if !exists {
		current = visitor{tokens: l.burst, lastRefill: now}
	}
	elapsed := now.Sub(current.lastRefill).Seconds()
	current.tokens += elapsed * l.rate
	if current.tokens > l.burst {
		current.tokens = l.burst
	}
	current.lastRefill = now
	current.lastSeen = now
	allowed := current.tokens >= 1
	if allowed {
		current.tokens--
	}
	l.visitors[key] = current

	if len(l.visitors) > 10_000 {
		cutoff := now.Add(-time.Hour)
		for visitorKey, item := range l.visitors {
			if item.lastSeen.Before(cutoff) {
				delete(l.visitors, visitorKey)
			}
		}
	}
	return allowed
}
