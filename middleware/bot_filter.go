package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"rubxy/logger"
)

// Bot detection patterns
var (
	// Suspicious User-Agent patterns
	// Note: Excluded "Go-http-client", "curl", "python-requests" as they're too common
	// in legitimate API integrations. Focus on known malicious scanners.
	suspiciousUserAgents = []string{
		"GenomeCrawlerd",
		"Palo Alto Networks",
		"scanner",
		"bot",
		"crawler",
		"spider",
	}

	// Common vulnerability scanning paths
	// Note: Excluded /admin/ since it's a legitimate route in this application
	suspiciousPaths = []string{
		"/webpages/",
		"/json",
		"/get.php",
		"/cgi-bin/",
		"/download/",
		"/wp-admin/",
		"/phpmyadmin/",
		"/.env",
		"/config.php",
		"/.git/",
		"/.svn/",
		"/login.html",
		"/powershell/",
	}

	// Rate limiting: max requests per IP per time window
	maxRequestsPerWindow = 10
	rateLimitWindow      = 1 * time.Minute
)

// IP request tracking
type ipTracker struct {
	count     int
	windowEnd time.Time
}

type rateLimiter struct {
	ips map[string]*ipTracker
	mu  sync.RWMutex
}

var limiter = &rateLimiter{
	ips: make(map[string]*ipTracker),
}

// Clean up old entries periodically
func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.cleanup()
		}
	}()
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	for ip, tracker := range rl.ips {
		if now.After(tracker.windowEnd) {
			delete(rl.ips, ip)
		}
	}
}

func (rl *rateLimiter) isAllowed(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	tracker, exists := rl.ips[ip]

	if !exists || now.After(tracker.windowEnd) {
		// New IP or window expired, reset
		rl.ips[ip] = &ipTracker{
			count:     1,
			windowEnd: now.Add(rateLimitWindow),
		}
		return true
	}

	if tracker.count >= maxRequestsPerWindow {
		return false
	}

	tracker.count++
	return true
}

// isSuspiciousUserAgent checks if the User-Agent matches known bot patterns
func isSuspiciousUserAgent(userAgent string) bool {
	userAgentLower := strings.ToLower(userAgent)
	for _, pattern := range suspiciousUserAgents {
		if strings.Contains(userAgentLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// isSuspiciousPath checks if the path matches common vulnerability scanning patterns
func isSuspiciousPath(path string) bool {
	pathLower := strings.ToLower(path)
	for _, pattern := range suspiciousPaths {
		if strings.Contains(pathLower, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

// isProxyScanner checks for proxy scanner indicators
func isProxyScanner(r *http.Request) bool {
	// Check for Proxy-Authorization header (common in proxy scanners)
	if r.Header.Get("Proxy-Authorization") != "" {
		return true
	}
	return false
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-Ip header
	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// BotFilter middleware filters out bot/scanner requests and applies rate limiting
func BotFilter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		userAgent := r.Header.Get("User-Agent")
		path := r.URL.Path

		// Check for bot indicators
		isBot := false
		reason := ""

		if isSuspiciousUserAgent(userAgent) {
			isBot = true
			reason = "suspicious_user_agent"
		} else if isSuspiciousPath(path) {
			isBot = true
			reason = "suspicious_path"
		} else if isProxyScanner(r) {
			isBot = true
			reason = "proxy_scanner"
		}

		// If bot detected, check rate limit
		if isBot {
			if !limiter.isAllowed(ip) {
				// Rate limited - return 431
				http.Error(w, "Request Header Fields Too Large", http.StatusRequestHeaderFieldsTooLarge)
				return
			}
			// Log suspicious pattern (summary format)
			logger.InfoLogger.Printf("[BOT] IP: %s, Reason: %s, Path: %s, User-Agent: %s",
				ip, reason, path, userAgent)
			http.Error(w, "Request Header Fields Too Large", http.StatusRequestHeaderFieldsTooLarge)
			return
		}

		// For legitimate requests, continue
		next.ServeHTTP(w, r)
	})
}

