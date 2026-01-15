package ratelimit

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for rate limiting
type Metrics struct {
	requestsAllowed    *prometheus.CounterVec
	requestsBlocked    *prometheus.CounterVec
	redisErrors        *prometheus.CounterVec
	redisLatency       *prometheus.HistogramVec
	activeRateLimits   prometheus.Gauge
	remainingQuota     *prometheus.HistogramVec
	rateLimitResetTime *prometheus.GaugeVec
}

// NewMetrics creates and registers rate limiting specific Prometheus metrics
func NewMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "app"
	}

	return &Metrics{
		requestsAllowed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "ratelimit",
				Name:      "requests_allowed_total",
				Help:      "Total number of requests allowed by rate limiter",
			},
			[]string{"endpoint"},
		),
		requestsBlocked: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "ratelimit",
				Name:      "requests_blocked_total",
				Help:      "Total number of requests blocked by rate limiter",
			},
			[]string{"endpoint", "ip"},
		),
		redisErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "ratelimit",
				Name:      "redis_errors_total",
				Help:      "Total number of Redis errors in rate limiter",
			},
			[]string{"operation", "error_type"},
		),
		redisLatency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "ratelimit",
				Name:      "redis_duration_seconds",
				Help:      "Redis operation latency for rate limiting in seconds",
				Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
			},
			[]string{"operation"},
		),
		activeRateLimits: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "ratelimit",
				Name:      "active_limits",
				Help:      "Current number of active rate limit entries in Redis",
			},
		),
		remainingQuota: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "ratelimit",
				Name:      "remaining_quota_percent",
				Help:      "Distribution of remaining quota percentage for requests",
				Buckets:   prometheus.LinearBuckets(0, 10, 11), // 0-100 in steps of 10
			},
			[]string{"endpoint"},
		),
		rateLimitResetTime: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "ratelimit",
				Name:      "reset_timestamp_seconds",
				Help:      "Unix timestamp when rate limit will reset for each IP",
			},
			[]string{"ip"},
		),
	}
}
