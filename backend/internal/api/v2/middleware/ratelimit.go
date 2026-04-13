// Package middleware provides v2-specific HTTP middleware.
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// windowEntry tracks the request timestamps within the current sliding window.
type windowEntry struct {
	mu         sync.Mutex
	timestamps []time.Time
}

// ipRateLimiter implements a sliding-window rate limiter keyed by client IP.
type ipRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*windowEntry
	limit   int
	window  time.Duration
}

// newIPRateLimiter constructs a rate limiter that allows at most limit requests
// per window per IP. The window uses a sliding (not fixed) bucket so bursts at
// the boundary are accounted for correctly.
func newIPRateLimiter(limit int, window time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		entries: make(map[string]*windowEntry),
		limit:   limit,
		window:  window,
	}
}

// allow returns true when the request should be allowed through, and false when
// the IP has exceeded the rate limit. It records the attempt regardless.
func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	entry, ok := l.entries[ip]
	if !ok {
		entry = &windowEntry{}
		l.entries[ip] = entry
	}
	l.mu.Unlock()

	entry.mu.Lock()
	defer entry.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// Evict timestamps older than the window.
	valid := entry.timestamps[:0]
	for _, ts := range entry.timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}
	entry.timestamps = valid

	if len(entry.timestamps) >= l.limit {
		return false
	}

	entry.timestamps = append(entry.timestamps, now)
	return true
}

// IPRateLimit returns a Gin middleware that limits requests to maxPerMin per IP
// per minute. Over-limit requests receive a 429 response with RATE_LIMIT_EXCEEDED.
// Uses Gin's c.ClientIP() which respects the engine's trusted proxy configuration.
func IPRateLimit(maxPerMin int) gin.HandlerFunc {
	limiter := newIPRateLimiter(maxPerMin, time.Minute)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "too many requests",
				},
			})
			return
		}
		c.Next()
	}
}
