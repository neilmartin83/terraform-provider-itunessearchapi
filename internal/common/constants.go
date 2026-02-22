// Package common provides shared utilities, constants, and helper functions
// used across the provider's packages.
package common

import "time"

// MaxLookupBatchSize is the maximum number of items per iTunes lookup API request.
const MaxLookupBatchSize = 200

// RateLimitRequests is the maximum number of API requests allowed per rate limit window.
const RateLimitRequests = 20

// RateLimitDuration is the time window for the rate limit request allowance.
const RateLimitDuration = 1 * time.Minute

// MaxRetries is the maximum number of retry attempts for rate-limited (HTTP 429) responses.
const MaxRetries = 5

// MaxRetryWait is the maximum duration to wait between retries for rate-limited responses.
const MaxRetryWait = 60 * time.Second

// DefaultReadTimeout is the default timeout for data source read operations.
const DefaultReadTimeout = 90 * time.Second
