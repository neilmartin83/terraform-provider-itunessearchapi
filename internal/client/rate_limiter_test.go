// Copyright Neil Martin 2026
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"testing"
	"time"
)

func TestTokenBucket_InitialTokensAvailable(t *testing.T) {
	tb := newTokenBucket()
	ctx := context.Background()

	for i := range 5 {
		if err := tb.take(ctx); err != nil {
			t.Fatalf("take %d failed: %v", i, err)
		}
	}
}

func TestTokenBucket_ContextCancellation(t *testing.T) {
	tb := &tokenBucket{
		tokens:         0,
		maxTokens:      1,
		refillRate:     0.001,
		lastRefillTime: time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := tb.take(ctx)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestTokenBucket_RefillOverTime(t *testing.T) {
	tb := &tokenBucket{
		tokens:         0,
		maxTokens:      20,
		refillRate:     100,
		lastRefillTime: time.Now(),
	}

	time.Sleep(50 * time.Millisecond)

	ctx := context.Background()
	if err := tb.take(ctx); err != nil {
		t.Fatalf("take after refill failed: %v", err)
	}
}
