package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a token bucket rate limiter per IP address
type RateLimiter struct {
	mu       sync.RWMutex
	visitors map[string]*visitor

	// Rate limiting configuration
	requestsPerMinute int
	burstSize         int

	// Cleanup
	cleanupInterval time.Duration
	lastCleanup     time.Time
}

// visitor tracks rate limit state for a single IP
type visitor struct {
	tokens         float64
	lastRefill     time.Time
	lastRequest    time.Time
	requestCount   int
}

// NewRateLimiter creates a new rate limiter
// requestsPerMinute: number of requests allowed per minute per IP
// burstSize: maximum burst of requests allowed
func NewRateLimiter(requestsPerMinute, burstSize int) *RateLimiter {
	return &RateLimiter{
		visitors:          make(map[string]*visitor),
		requestsPerMinute: requestsPerMinute,
		burstSize:         burstSize,
		cleanupInterval:   5 * time.Minute,
		lastCleanup:       time.Now(),
	}
}

// Allow checks if a request from the given IP should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Periodic cleanup of old visitors
	if time.Since(rl.lastCleanup) > rl.cleanupInterval {
		rl.cleanup()
	}

	// Get or create visitor
	v, exists := rl.visitors[ip]
	if !exists {
		v = &visitor{
			tokens:      float64(rl.burstSize),
			lastRefill:  time.Now(),
			lastRequest: time.Now(),
		}
		rl.visitors[ip] = v
	}

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(v.lastRefill).Seconds()
	tokensToAdd := elapsed * (float64(rl.requestsPerMinute) / 60.0)

	v.tokens += tokensToAdd
	if v.tokens > float64(rl.burstSize) {
		v.tokens = float64(rl.burstSize)
	}
	v.lastRefill = now

	// Check if request is allowed
	if v.tokens >= 1.0 {
		v.tokens -= 1.0
		v.lastRequest = now
		v.requestCount++
		return true
	}

	return false
}

// cleanup removes visitors that haven't made requests in the last 10 minutes
func (rl *RateLimiter) cleanup() {
	cutoff := time.Now().Add(-10 * time.Minute)
	for ip, v := range rl.visitors {
		if v.lastRequest.Before(cutoff) {
			delete(rl.visitors, ip)
		}
	}
	rl.lastCleanup = time.Now()
}

// GetStats returns statistics about the rate limiter (for monitoring)
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	totalRequests := 0
	for _, v := range rl.visitors {
		totalRequests += v.requestCount
	}

	return map[string]interface{}{
		"active_ips":      len(rl.visitors),
		"total_requests":  totalRequests,
		"requests_per_min": rl.requestsPerMinute,
		"burst_size":      rl.burstSize,
	}
}

// RateLimitMiddleware creates HTTP middleware that enforces rate limiting
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract IP address
			ip := getClientIP(r)

			// Check rate limit
			if !limiter.Allow(ip) {
				http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
				return
			}

			// Add rate limit headers for client visibility
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "...") // Could calculate based on tokens

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the real client IP from the request
// Handles proxies, load balancers, and direct connections
func getClientIP(r *http.Request) string {
	// Try X-Forwarded-For header (set by proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the chain
		ips := parseXForwardedFor(xff)
		if len(ips) > 0 {
			return ips[0]
		}
	}

	// Try X-Real-IP header (set by some proxies)
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// parseXForwardedFor parses the X-Forwarded-For header value
func parseXForwardedFor(header string) []string {
	var ips []string
	for _, ip := range splitAndTrim(header, ",") {
		if ip != "" {
			ips = append(ips, ip)
		}
	}
	return ips
}

// splitAndTrim splits a string and trims whitespace from each part
func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range splitString(s, sep) {
		trimmed := trimWhitespace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// splitString splits a string by separator
func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// trimWhitespace removes leading and trailing whitespace
func trimWhitespace(s string) string {
	start := 0
	end := len(s)

	// Trim leading whitespace
	for start < end && isWhitespace(s[start]) {
		start++
	}

	// Trim trailing whitespace
	for end > start && isWhitespace(s[end-1]) {
		end--
	}

	return s[start:end]
}

// isWhitespace checks if a byte is whitespace
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
