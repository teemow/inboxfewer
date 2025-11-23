package oauth

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements a per-identifier (IP or user) rate limiter using token buckets
// Uses the battle-tested golang.org/x/time/rate package for token bucket implementation
type RateLimiter struct {
	mu         sync.RWMutex
	limiters   map[string]*limiterEntry
	rate       int           // tokens per second
	burst      int           // max burst size
	cleanup    time.Duration // cleanup interval for inactive limiters
	trustProxy bool          // whether to trust proxy headers (only for IP-based limiting)
	logger     *slog.Logger
}

// limiterEntry tracks a rate limiter and its last access time for cleanup
type limiterEntry struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// NewRateLimiter creates a new rate limiter
// rate: tokens per second, burst: maximum burst size, trustProxy: whether to trust proxy headers
// cleanupInterval: how often to cleanup inactive limiters, logger: structured logger
func NewRateLimiter(ratePerSec, burst int, trustProxy bool, cleanupInterval time.Duration, logger *slog.Logger) *RateLimiter {
	if logger == nil {
		logger = slog.Default()
	}

	rl := &RateLimiter{
		limiters:   make(map[string]*limiterEntry),
		rate:       ratePerSec,
		burst:      burst,
		cleanup:    cleanupInterval,
		trustProxy: trustProxy,
		logger:     logger,
	}

	logger.Info("Rate limiter initialized",
		"rate", ratePerSec,
		"burst", burst,
		"trust_proxy", trustProxy,
		"cleanup_interval", cleanupInterval)

	// Start cleanup goroutine
	go rl.cleanupInactiveLimiters()

	return rl
}

// Allow checks if a request from the given identifier (IP or user email) should be allowed
func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.RLock()
	entry, exists := rl.limiters[identifier]
	rl.mu.RUnlock()

	if !exists {
		// Create new rate limiter for this identifier
		entry = &limiterEntry{
			limiter:    rate.NewLimiter(rate.Limit(rl.rate), rl.burst),
			lastAccess: time.Now(),
		}
		rl.mu.Lock()
		rl.limiters[identifier] = entry
		rl.mu.Unlock()
		rl.logger.Debug("Created rate limiter", "identifier", identifier)
	} else {
		// Update last access time
		entry.lastAccess = time.Now()
	}

	// Check if request is allowed
	allowed := entry.limiter.Allow()
	if !allowed {
		rl.logger.Warn("Rate limit exceeded", "identifier", identifier)
	}

	return allowed
}

// cleanupInactiveLimiters removes limiters that haven't been used recently
// Simplified to use a single lock for background cleanup (KISS principle)
func (rl *RateLimiter) cleanupInactiveLimiters() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		removed := 0

		for identifier, entry := range rl.limiters {
			if now.Sub(entry.lastAccess) > InactiveLimiterCleanupWindow {
				delete(rl.limiters, identifier)
				removed++
			}
		}
		rl.mu.Unlock()

		if removed > 0 {
			rl.logger.Debug("Cleaned up inactive rate limiters", "count", removed)
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
