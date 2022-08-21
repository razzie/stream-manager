package beepboop

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// ErrRateLimitExceeded ...
var ErrRateLimitExceeded = fmt.Errorf("rate limit exceeded")

// RateLimiter is a simple N/interval/IP rate limiter
type RateLimiter struct {
	ips      map[string]*rate.Limiter
	mtx      sync.RWMutex
	interval rate.Limit
	n        int
}

// NewRateLimiter returns a new RateLimiter
func NewRateLimiter(interval time.Duration, n int) *RateLimiter {
	return &RateLimiter{
		ips:      make(map[string]*rate.Limiter),
		interval: rate.Every(interval),
		n:        n,
	}
}

// Get returns the limiter state for the given IP
func (r *RateLimiter) Get(ip string) *rate.Limiter {
	r.mtx.RLock()
	limiter, ok := r.ips[ip]
	r.mtx.RUnlock()

	if !ok {
		limiter = rate.NewLimiter(r.interval, r.n)
		r.mtx.Lock()
		r.ips[ip] = limiter
		r.mtx.Unlock()
	}

	return limiter
}
