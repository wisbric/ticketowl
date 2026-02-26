package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter limits login attempts per IP using Redis INCR + EXPIRE.
type RateLimiter struct {
	redis      *redis.Client
	maxAttempt int
	window     time.Duration
}

// NewRateLimiter creates a rate limiter. maxAttempt is the max failed attempts
// allowed per IP within the given window.
func NewRateLimiter(rdb *redis.Client, maxAttempt int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		redis:      rdb,
		maxAttempt: maxAttempt,
		window:     window,
	}
}

// RateLimitResult holds the result of a rate limit check.
type RateLimitResult struct {
	Allowed   bool
	Remaining int
	RetryAt   time.Time
}

// Check returns whether the given IP is allowed to attempt a login.
func (rl *RateLimiter) Check(ctx context.Context, ip string) (*RateLimitResult, error) {
	key := fmt.Sprintf("login_ratelimit:%s", ip)

	count, err := rl.redis.Get(ctx, key).Int()
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("checking rate limit: %w", err)
	}

	if count >= rl.maxAttempt {
		ttl, err := rl.redis.TTL(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("getting TTL: %w", err)
		}
		return &RateLimitResult{
			Allowed:   false,
			Remaining: 0,
			RetryAt:   time.Now().Add(ttl),
		}, nil
	}

	return &RateLimitResult{
		Allowed:   true,
		Remaining: rl.maxAttempt - count,
	}, nil
}

// Record records a failed login attempt for the given IP.
func (rl *RateLimiter) Record(ctx context.Context, ip string) error {
	key := fmt.Sprintf("login_ratelimit:%s", ip)

	pipe := rl.redis.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, rl.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("recording rate limit: %w", err)
	}

	// Only set the expiry on the first increment.
	if incr.Val() == 1 {
		rl.redis.Expire(ctx, key, rl.window)
	}

	return nil
}

// Reset clears the rate limit counter for a given IP (on successful login).
func (rl *RateLimiter) Reset(ctx context.Context, ip string) error {
	key := fmt.Sprintf("login_ratelimit:%s", ip)
	return rl.redis.Del(ctx, key).Err()
}
