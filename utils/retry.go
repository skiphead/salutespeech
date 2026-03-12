package utils

import (
	"context"
	"time"
)

// RetryConfig represents retry configuration
type RetryConfig struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
}

// DefaultRetryConfig returns default retry config
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	}
}

// RetryWithBackoff retries function with exponential backoff
func RetryWithBackoff(ctx context.Context, fn func() error, cfg RetryConfig, isRetryable func(error) bool) error {
	var lastErr error
	interval := cfg.InitialInterval

	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		if !isRetryable(err) {
			return err
		}

		lastErr = err

		if attempt >= cfg.MaxRetries {
			break
		}

		select {
		case <-time.After(interval):
		case <-ctx.Done():
			return ctx.Err()
		}

		interval = time.Duration(float64(interval) * cfg.Multiplier)
		if interval > cfg.MaxInterval {
			interval = cfg.MaxInterval
		}
	}

	return lastErr
}
