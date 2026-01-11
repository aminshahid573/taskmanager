package ratelimit

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aminshahid573/taskmanager/internal/cache"
	"github.com/aminshahid573/taskmanager/internal/config"
	"github.com/redis/go-redis/v9"
)

// Lua script for atomic sliding window rate limiting
const luaScript = `
local key = KEYS[1]
local limit = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

-- Remove old entries outside the window
redis.call('ZREMRANGEBYSCORE', key, 0, now - window)

-- Count current entries
local count = redis.call('ZCARD', key)

-- Calculate oldest timestamp in window for reset calculation
local oldest = nil
if count > 0 then
    local oldest_entries = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
    if #oldest_entries > 0 then
        oldest = tonumber(oldest_entries[2])
    end
end

if count < limit then
    -- Add new entry
    redis.call('ZADD', key, now, now)
    redis.call('EXPIRE', key, math.ceil(window / 1000))
    local reset_time = oldest and (oldest + window) or (now + window)
    return {1, limit - count - 1, reset_time}
else
    -- Calculate when the oldest entry will expire
    local reset_time = oldest and (oldest + window) or (now + window)
    return {0, 0, reset_time}
end
`

type RateLimiter struct {
	redisClient *cache.RedisClient
	client      *redis.Client // Direct Redis client for Lua scripts
	limit       int
	window      time.Duration
	script      *redis.Script
	metrics     *Metrics

	// For periodic metrics collection
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewRateLimiter creates a new rate limiter using existing cache and config
func NewRateLimiter(cfg *config.Config, redisClient *cache.RedisClient) (*RateLimiter, error) {
	if !cfg.RateLimit.Enabled {
		return nil, fmt.Errorf("rate limiting is disabled in config")
	}

	// Extract the underlying Redis client from cache.RedisClient
	// We need this for Lua script execution
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Determine limit and window
	limit := cfg.RateLimit.RequestsPerMinute
	if limit == 0 {
		limit = 100 // default
	}

	window := time.Duration(cfg.RateLimit.Window) * time.Second
	if window == 0 {
		window = time.Minute // default
	}

	metricsNamespace := cfg.RateLimit.MetricsNamespace
	if metricsNamespace == "" {
		metricsNamespace = cfg.App.Name
	}

	rl := &RateLimiter{
		redisClient: redisClient,
		client:      client,
		limit:       limit,
		window:      window,
		script:      redis.NewScript(luaScript),
		metrics:     NewMetrics(metricsNamespace),
		stopCh:      make(chan struct{}),
	}

	// Start background metrics collection
	rl.startMetricsCollection()

	return rl, nil
}

// startMetricsCollection starts periodic collection of Redis metrics
func (rl *RateLimiter) startMetricsCollection() {
	rl.wg.Add(1)
	go func() {
		defer rl.wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				rl.collectRedisMetrics()
			case <-rl.stopCh:
				return
			}
		}
	}()
}

// collectRedisMetrics collects metrics from Redis
func (rl *RateLimiter) collectRedisMetrics() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Count active rate limit keys
	keys, err := rl.client.Keys(ctx, "rate_limit:*").Result()
	if err != nil {
		log.Printf("Failed to collect Redis metrics: %v", err)
		return
	}

	rl.metrics.activeRateLimits.Set(float64(len(keys)))
}

// Close gracefully shuts down the rate limiter
func (rl *RateLimiter) Close() error {
	close(rl.stopCh)
	rl.wg.Wait()
	return rl.client.Close()
}

// GetStats returns current rate limiter statistics
func (rl *RateLimiter) GetStats(ctx context.Context) (*Stats, error) {
	keys, err := rl.client.Keys(ctx, "rate_limit:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get keys: %w", err)
	}

	stats := &Stats{
		ActiveLimits: len(keys),
		Limits:       make([]LimitInfo, 0, len(keys)),
	}

	// Sample up to 100 keys to avoid overwhelming Redis
	sampleSize := len(keys)
	if sampleSize > 100 {
		sampleSize = 100
	}

	for i := 0; i < sampleSize; i++ {
		key := keys[i]

		count, err := rl.client.ZCard(ctx, key).Result()
		if err != nil {
			continue
		}

		ttl, err := rl.client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}

		ip := strings.TrimPrefix(key, "rate_limit:")
		stats.Limits = append(stats.Limits, LimitInfo{
			IP:        ip,
			Count:     int(count),
			TTL:       ttl,
			Remaining: rl.limit - int(count),
		})
	}

	return stats, nil
}

// Stats holds rate limiter statistics
type Stats struct {
	ActiveLimits int
	Limits       []LimitInfo
}

// LimitInfo holds information about a single rate limit
type LimitInfo struct {
	IP        string
	Count     int
	TTL       time.Duration
	Remaining int
}

