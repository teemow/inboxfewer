package oauth

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter per IP address
type RateLimiter struct {
	mu         sync.RWMutex
	limiters   map[string]*bucket
	rate       int           // tokens per second
	burst      int           // max burst size
	cleanup    time.Duration // cleanup interval for inactive limiters
	trustProxy bool          // whether to trust proxy headers
}

// bucket represents a token bucket for rate limiting
type bucket struct {
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: tokens per second, burst: maximum burst size, trustProxy: whether to trust proxy headers
func NewRateLimiter(rate, burst int, trustProxy bool) *RateLimiter {
	rl := &RateLimiter{
		limiters:   make(map[string]*bucket),
		rate:       rate,
		burst:      burst,
		cleanup:    5 * time.Minute,
		trustProxy: trustProxy,
	}

	// Start cleanup goroutine
	go rl.cleanupInactiveLimiters()

	return rl
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.RLock()
	b, exists := rl.limiters[ip]
	rl.mu.RUnlock()

	if !exists {
		// Create new bucket for this IP
		b = &bucket{
			tokens:     float64(rl.burst),
			lastUpdate: time.Now(),
		}
		rl.mu.Lock()
		rl.limiters[ip] = b
		rl.mu.Unlock()
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastUpdate).Seconds()

	// Add tokens based on elapsed time
	b.tokens += elapsed * float64(rl.rate)
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.lastUpdate = now

	// Check if we have a token available
	if b.tokens >= 1 {
		b.tokens--
		return true
	}

	return false
}

// cleanupInactiveLimiters removes limiters that haven't been used recently
func (rl *RateLimiter) cleanupInactiveLimiters() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, b := range rl.limiters {
			b.mu.Lock()
			if now.Sub(b.lastUpdate) > 10*time.Minute {
				delete(rl.limiters, ip)
			}
			b.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware is middleware that applies rate limiting
func (h *Handler) RateLimitMiddleware(next http.Handler) http.Handler {
	if h.rateLimiter == nil {
		// No rate limiter configured, pass through
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract IP address
		ip := getClientIP(r, h.rateLimiter.trustProxy)

		if !h.rateLimiter.Allow(ip) {
			w.Header().Set("Retry-After", "1")
			h.writeError(w, "rate_limit_exceeded", 
				fmt.Sprintf("Rate limit exceeded for %s. Please try again later", ip),
				http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the client IP address from the request
// trustProxy: if true, trust X-Forwarded-For and X-Real-IP headers (only if behind trusted proxy)
func getClientIP(r *http.Request, trustProxy bool) string {
	// Only trust proxy headers if explicitly configured
	if trustProxy {
		// Check X-Forwarded-For header (set by proxies)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Take the first IP if multiple
			for i := 0; i < len(xff); i++ {
				if xff[i] == ',' {
					return xff[:i]
				}
			}
			return xff
		}

		// Check X-Real-IP header
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
	}

	// Fall back to RemoteAddr (always trusted)
	// RemoteAddr is in format "IP:port", extract just the IP
	return extractIPFromAddr(r.RemoteAddr)
}

// extractIPFromAddr extracts the IP address from "IP:port" format
func extractIPFromAddr(addr string) string {
	for i := 0; i < len(addr); i++ {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

