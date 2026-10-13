package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/troubleman96/ziadapos/internal/config"
)

// Cache is a thin JSON-aware Redis wrapper used across the API for
// response caching, rate limiting and ephemeral state (OTPs, tokens).
type Cache struct {
	client *redis.Client
	ttl    time.Duration
}

// New connects to Redis. It is safe for the server to boot with a degraded
// cache (all operations silently fall back to DB) by wrapping errors in
// helpers, but we fail fast on an unreachable Redis in production.
func New(ctx context.Context, cfg *config.Config) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("cache: redis unavailable (%v)", err)
		return nil, err
	}
	log.Println("cache: connected to Redis")
	return &Cache{client: client, ttl: cfg.CacheTTL}, nil
}

// Client exposes the underlying go-redis client (rate limiter, pub/sub...).
func (c *Cache) Client() *redis.Client { return c.client }

// SetJSON marshals v and stores it with the configured default TTL.
func (c *Cache) SetJSON(ctx context.Context, key string, v interface{}) error {
	return c.SetJSONTTL(ctx, key, v, c.ttl)
}

// SetJSONTTL marshals v and stores it with a specific TTL.
func (c *Cache) SetJSONTTL(ctx context.Context, key string, v interface{}, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, b, ttl).Err()
}

// GetJSON unmarshals the value at key into v. Returns redis.Nil when absent.
func (c *Cache) GetJSON(ctx context.Context, key string, v interface{}) error {
	b, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// Get returns the raw string value at key.
func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// Set stores a raw string value with the default TTL.
func (c *Cache) Set(ctx context.Context, key, value string) error {
	return c.client.Set(ctx, key, value, c.ttl).Err()
}

// SetTTL stores a raw string with a specific TTL.
func (c *Cache) SetTTL(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Del removes one or more keys.
func (c *Cache) Del(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// DeletePattern removes all keys matching a glob pattern (e.g. "inventory:*").
func (c *Cache) DeletePattern(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 100).Iterator()
	var keys []string
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return err
	}
	if len(keys) > 0 {
		return c.client.Del(ctx, keys...).Err()
	}
	return nil
}

// Incr increments a counter (used for OTP attempt limits and login lockouts).
func (c *Cache) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// Expire sets a TTL on an existing key.
func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.client.Expire(ctx, key, ttl).Err()
}

// Close shuts down the Redis connection.
func (c *Cache) Close() error { return c.client.Close() }
