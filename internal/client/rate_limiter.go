package client

import (
	"context"
	"sync"
	"time"

	"github.com/neilmartin83/terraform-provider-itunessearchapi/internal/common"
)

// tokenBucket implements a simple token bucket rate limiter.
type tokenBucket struct {
	tokens         float64
	maxTokens      float64
	refillRate     float64
	lastRefillTime time.Time
	mu             sync.Mutex
}

// newTokenBucket creates a new token bucket rate limiter using the configured
// rate limit constants.
func newTokenBucket() *tokenBucket {
	maxRequests := common.RateLimitRequests
	perDuration := common.RateLimitDuration
	refillRate := float64(maxRequests) / perDuration.Seconds()
	return &tokenBucket{
		tokens:         float64(maxRequests),
		maxTokens:      float64(maxRequests),
		refillRate:     refillRate,
		lastRefillTime: time.Now(),
	}
}

// take attempts to take a token from the bucket, blocking until one is available
// or the context is cancelled.
func (tb *tokenBucket) take(ctx context.Context) error {
	for {
		tb.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(tb.lastRefillTime).Seconds()

		tb.tokens += elapsed * tb.refillRate
		if tb.tokens > tb.maxTokens {
			tb.tokens = tb.maxTokens
		}
		tb.lastRefillTime = now

		if tb.tokens >= 1.0 {
			tb.tokens -= 1.0
			tb.mu.Unlock()
			return nil
		}

		tokensNeeded := 1.0 - tb.tokens
		waitDuration := time.Duration(tokensNeeded / tb.refillRate * float64(time.Second))
		tb.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitDuration):
		}
	}
}
