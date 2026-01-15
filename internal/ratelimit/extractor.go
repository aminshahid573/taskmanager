package ratelimit

import (
	"net"
	"net/http"
	"strings"
)

// extractIP extracts the real IP address from the request
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list (client IP)
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr, strip port
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}

	return r.RemoteAddr
}

// classifyError classifies Redis errors for metrics
func classifyError(err error) string {
	if err == nil {
		return "none"
	}

	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "connection refused"):
		return "connection_refused"
	case strings.Contains(errStr, "connection reset"):
		return "connection_reset"
	case strings.Contains(errStr, "EOF"):
		return "eof"
	case strings.Contains(errStr, "pool"):
		return "pool_exhausted"
	default:
		return "unknown"
	}
}
