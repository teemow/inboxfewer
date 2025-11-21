package oauth

import (
	"fmt"
	"log/slog"
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
	logger     *slog.Logger
}

// bucket represents a token bucket for rate limiting
type bucket struct {
	tokens     float64
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
// rate: tokens per second, burst: maximum burst size, trustProxy: whether to trust proxy headers
// cleanupInterval: how often to cleanup inactive limiters, logger: structured logger
func NewRateLimiter(rate, burst int, trustProxy bool, cleanupInterval time.Duration, logger *slog.Logger) *RateLimiter {
	if logger == nil {
		logger = slog.Default()
	}

	rl := &RateLimiter{
		limiters:   make(map[string]*bucket),
		rate:       rate,
		burst:      burst,
		cleanup:    cleanupInterval,
		trustProxy: trustProxy,
		logger:     logger,
	}

	logger.Info("Rate limiter initialized",
		"rate", rate,
		"burst", burst,
		"trust_proxy", trustProxy,
		"cleanup_interval", cleanupInterval)

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
		rl.logger.Debug("Created rate limiter for IP", "ip", ip)
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

	rl.logger.Warn("Rate limit exceeded", "ip", ip, "remaining_tokens", b.tokens)
	return false
}

// cleanupInactiveLimiters removes limiters that haven't been used recently
// Uses optimized locking strategy to prevent deadlocks:
// 1. Collect IPs to delete under read lock (doesn't block Allow())
// 2. Delete collected IPs under write lock (minimal lock duration)
func (rl *RateLimiter) cleanupInactiveLimiters() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		// Phase 1: Identify inactive limiters under read lock
		// This doesn't block Allow() from creating new limiters
		rl.mu.RLock()
		inactiveIPs := []string{}
		for ip, b := range rl.limiters {
			b.mu.Lock()
			lastUpdate := b.lastUpdate
			b.mu.Unlock()

			if now.Sub(lastUpdate) > 10*time.Minute {
				inactiveIPs = append(inactiveIPs, ip)
			}
		}
		rl.mu.RUnlock()

		// Phase 2: Delete inactive limiters under write lock
		// Re-check staleness to handle race conditions
		if len(inactiveIPs) > 0 {
			rl.mu.Lock()
			removed := 0
			for _, ip := range inactiveIPs {
				if b, exists := rl.limiters[ip]; exists {
					b.mu.Lock()
					// Re-check under write lock to avoid deleting recently active limiters
					if now.Sub(b.lastUpdate) > 10*time.Minute {
						delete(rl.limiters, ip)
						removed++
					}
					b.mu.Unlock()
				}
			}
			rl.mu.Unlock()

			if removed > 0 {
				rl.logger.Debug("Cleaned up inactive rate limiters", "count", removed)
			}
		}
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
