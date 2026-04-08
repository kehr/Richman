package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// PerUserRateLimit returns a gin middleware that allows at most one
// request per user per `window`. Used by the /analysis/reanalyze-all
// endpoint so a dashboard banner cannot stampede the analysis pipeline
// if the user double-clicks or opens multiple tabs.
//
// The limiter is deliberately minimal: an in-memory map keyed by
// authenticated user id, guarded by a single mutex. No LRU eviction is
// needed because the map grows linearly with active users and Richman's
// single-instance deployment has a user count orders of magnitude below
// any memory concern. Swapping this for a Redis-backed limiter is a
// drop-in replacement when we move to multi-instance.
//
// The middleware MUST be installed AFTER the auth middleware so
// middleware.GetUserID returns the authenticated id. Unauthenticated
// callers get UNAUTHORIZED before the limiter runs.
func PerUserRateLimit(window time.Duration) gin.HandlerFunc {
	var (
		mu       sync.Mutex
		lastHits = make(map[int64]time.Time)
	)
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID <= 0 {
			c.Next()
			return
		}
		now := time.Now()
		mu.Lock()
		last, ok := lastHits[userID]
		remaining := time.Duration(0)
		if ok {
			elapsed := now.Sub(last)
			if elapsed < window {
				remaining = window - elapsed
				mu.Unlock()
				c.Header("Retry-After", formatSeconds(remaining))
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"code":    "RATE_LIMITED",
						"message": "too many requests; try again later",
					},
				})
				return
			}
		}
		lastHits[userID] = now
		mu.Unlock()
		c.Next()
	}
}

// formatSeconds renders a duration as a whole-second integer string for
// the Retry-After header. Values below 1s round up to 1 so clients never
// see a zero and immediately retry.
func formatSeconds(d time.Duration) string {
	seconds := int64(d.Seconds())
	if seconds <= 0 {
		seconds = 1
	}
	return itoa(seconds)
}

// itoa is a dependency-free int-to-string helper so this file does not
// pull in strconv just for the header value. Kept minimal on purpose.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
